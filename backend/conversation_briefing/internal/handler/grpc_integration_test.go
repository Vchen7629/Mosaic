//go:build integration

package handler_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cb "mosaic-conversation-briefing.com/gen"
	"mosaic-conversation-briefing.com/internal/cache"
	"mosaic-conversation-briefing.com/internal/db"
	"mosaic-conversation-briefing.com/internal/handler"
	"mosaic-conversation-briefing.com/internal/test"
)

var testDB *test.TestDBContainer
var testCache *test.TestCacheContainer

func TestMain(m *testing.M) {
	var dbCleanup func()
	testDB, dbCleanup = test.SetupTestDatabaseForTestMain()
	defer dbCleanup()

	var cacheCleanup func()
	testCache, cacheCleanup = test.SetupTestCacheForTestMain()
	defer cacheCleanup()

	os.Exit(m.Run())
}

func newLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func cleanupSessions(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(context.Background(), "TRUNCATE TABLE sessions")
	if err != nil {
		t.Fatalf("Failed to truncate sessions: %v", err)
	}
}

func makeLLMServer(response string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"message": map[string]string{"role": "assistant", "content": response},
		})
	}))
}

func TestGenerateConversationBriefing_Integration(t *testing.T) {
	pool := testDB.Pool
	logger := newLogger()
	cacheClient := testCache.Client

	t.Run("session resolution - db hit", func(t *testing.T) {
		test.CleanupTables(t, pool)
		cleanupSessions(t, pool)
		test.FlushCache(t, cacheClient)

		embedding := test.MakeEmbedding(0.1, 128)
		profileID := test.SeedProfile(t, pool, embedding)
		visitorID := test.SeedVisitor(t, pool, profileID, "alice", embedding)
		test.SeedConvos(t, pool, profileID, visitorID, "hello from alice")
		sessionToken := test.SeedSession(t, pool, profileID)

		srv := makeLLMServer("Alice is a loyal customer")
		defer srv.Close()

		s := handler.NewConvoBriefingServer(logger, cacheClient, db.NewDBPool(pool, logger), srv.URL)
		resp, err := s.GenerateConversationBriefing(context.Background(), &cb.GenerateConversationBriefingRequest{
			SessionToken: sessionToken,
			VisitorIds:   []int32{visitorID},
		})

		assert.NoError(t, err)
		assert.True(t, resp.Success)
		text, found := test.FetchBriefing(t, pool, profileID, visitorID)
		assert.True(t, found)
		assert.Equal(t, "Alice is a loyal customer", text)
	})

	t.Run("session resolution - cache hit", func(t *testing.T) {
		test.CleanupTables(t, pool)
		cleanupSessions(t, pool)
		test.FlushCache(t, cacheClient)

		embedding := test.MakeEmbedding(0.1, 128)
		profileID := test.SeedProfile(t, pool, embedding)
		visitorID := test.SeedVisitor(t, pool, profileID, "bob", embedding)
		test.SeedConvos(t, pool, profileID, visitorID, "hello from bob")

		sessionToken := "cache-only-token"
		err := cache.CreateNewProfileSession(context.Background(), profileID, cacheClient, sessionToken)
		assert.NoError(t, err)

		srv := makeLLMServer("Bob is a returning customer")
		defer srv.Close()

		s := handler.NewConvoBriefingServer(logger, cacheClient, db.NewDBPool(pool, logger), srv.URL)
		resp, err := s.GenerateConversationBriefing(context.Background(), &cb.GenerateConversationBriefingRequest{
			SessionToken: sessionToken,
			VisitorIds:   []int32{visitorID},
		})

		assert.NoError(t, err)
		assert.True(t, resp.Success)
		text, found := test.FetchBriefing(t, pool, profileID, visitorID)
		assert.True(t, found)
		assert.Equal(t, "Bob is a returning customer", text)
	})

	t.Run("invalid session token not in cache or db returns error", func(t *testing.T) {
		test.CleanupTables(t, pool)
		cleanupSessions(t, pool)
		test.FlushCache(t, cacheClient)

		s := handler.NewConvoBriefingServer(logger, cacheClient, db.NewDBPool(pool, logger), "")
		resp, err := s.GenerateConversationBriefing(context.Background(), &cb.GenerateConversationBriefingRequest{
			SessionToken: "nonexistent-token",
			VisitorIds:   []int32{1},
		})

		assert.Error(t, err)
		assert.False(t, resp.Success)
	})

	t.Run("invalid visitor ids return error", func(t *testing.T) {
		test.CleanupTables(t, pool)
		cleanupSessions(t, pool)
		test.FlushCache(t, cacheClient)

		embedding := test.MakeEmbedding(0.1, 128)
		profileID := test.SeedProfile(t, pool, embedding)
		sessionToken := test.SeedSession(t, pool, profileID)

		tests := []struct {
			name       string
			visitorIDs []int32
		}{
			{"zero visitor ID", []int32{0}},
			{"negative visitor ID", []int32{-5}},
			{"invalid ID among valid ones", []int32{10, 20, -1, 30}},
		}

		s := handler.NewConvoBriefingServer(logger, cacheClient, db.NewDBPool(pool, logger), "")
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				resp, err := s.GenerateConversationBriefing(context.Background(), &cb.GenerateConversationBriefingRequest{
					SessionToken: sessionToken,
					VisitorIds:   tt.visitorIDs,
				})
				assert.Error(t, err)
				assert.False(t, resp.Success)
			})
		}
	})

	t.Run("briefing not written on error", func(t *testing.T) {
		embedding := test.MakeEmbedding(0.1, 128)

		tests := []struct {
			name      string
			seedConvo bool
			llmStatus int
		}{
			{"no conversations exist", false, http.StatusOK},
			{"LLM call fails", true, http.StatusInternalServerError},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				test.CleanupTables(t, pool)
				cleanupSessions(t, pool)
				test.FlushCache(t, cacheClient)

				profileID := test.SeedProfile(t, pool, embedding)
				visitorID := test.SeedVisitor(t, pool, profileID, "visitor", embedding)
				sessionToken := test.SeedSession(t, pool, profileID)
				if tt.seedConvo {
					test.SeedConvos(t, pool, profileID, visitorID, "some message")
				}

				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tt.llmStatus)
				}))
				defer srv.Close()

				s := handler.NewConvoBriefingServer(logger, cacheClient, db.NewDBPool(pool, logger), srv.URL)
				resp, err := s.GenerateConversationBriefing(context.Background(), &cb.GenerateConversationBriefingRequest{
					SessionToken: sessionToken,
					VisitorIds:   []int32{visitorID},
				})

				assert.Error(t, err)
				assert.False(t, resp.Success)
				_, found := test.FetchBriefing(t, pool, profileID, visitorID)
				assert.False(t, found, "no briefing should be written")
			})
		}
	})

	t.Run("db pool failure on FetchRecentConversations returns error", func(t *testing.T) {
		test.CleanupTables(t, pool)
		cleanupSessions(t, pool)
		test.FlushCache(t, cacheClient)

		embedding := test.MakeEmbedding(0.1, 128)
		profileID := test.SeedProfile(t, pool, embedding)
		visitorID := test.SeedVisitor(t, pool, profileID, "eve", embedding)

		// seed session only in cache so the closed pool isn't hit during session lookup
		sessionToken := "closed-pool-token"
		err := cache.CreateNewProfileSession(context.Background(), profileID, cacheClient, sessionToken)
		assert.NoError(t, err)

		// connect a second pool to the same DB and immediately close it
		closedPool, err := pgxpool.Connect(context.Background(), testDB.ConnStr)
		require.NoError(t, err)
		closedPool.Close()

		s := handler.NewConvoBriefingServer(logger, cacheClient, db.NewDBPool(closedPool, logger), "")
		resp, err := s.GenerateConversationBriefing(context.Background(), &cb.GenerateConversationBriefingRequest{
			SessionToken: sessionToken,
			VisitorIds:   []int32{visitorID},
		})

		assert.Error(t, err)
		assert.False(t, resp.Success)
	})

	t.Run("writes correct briefings for multiple visitors", func(t *testing.T) {
		test.CleanupTables(t, pool)
		cleanupSessions(t, pool)
		test.FlushCache(t, cacheClient)

		embedding := test.MakeEmbedding(0.1, 128)
		profileID := test.SeedProfile(t, pool, embedding)
		visitorID1 := test.SeedVisitor(t, pool, profileID, "visitor1", embedding)
		visitorID2 := test.SeedVisitor(t, pool, profileID, "visitor2", embedding)
		test.SeedConvos(t, pool, profileID, visitorID1, "visitor1 message")
		test.SeedConvos(t, pool, profileID, visitorID2, "visitor2 message")
		sessionToken := test.SeedSession(t, pool, profileID)

		call := 0
		responses := []string{"briefing for visitor1", "briefing for visitor2"}
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"message": map[string]string{"content": responses[call]},
			})
			call++
		}))
		defer srv.Close()

		s := handler.NewConvoBriefingServer(logger, cacheClient, db.NewDBPool(pool, logger), srv.URL)
		resp, err := s.GenerateConversationBriefing(context.Background(), &cb.GenerateConversationBriefingRequest{
			SessionToken: sessionToken,
			VisitorIds:   []int32{visitorID1, visitorID2},
		})

		assert.NoError(t, err)
		assert.True(t, resp.Success)

		text1, found1 := test.FetchBriefing(t, pool, profileID, visitorID1)
		assert.True(t, found1)
		text2, found2 := test.FetchBriefing(t, pool, profileID, visitorID2)
		assert.True(t, found2)
		assert.ElementsMatch(t, []string{"briefing for visitor1", "briefing for visitor2"}, []string{text1, text2})
	})
}

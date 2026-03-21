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

	"github.com/stretchr/testify/assert"

	cb "mosaic-conversation-briefing.com/gen"
	"mosaic-conversation-briefing.com/internal/db"
	"mosaic-conversation-briefing.com/internal/handler"
	"mosaic-conversation-briefing.com/internal/test"
)

var testDB *test.TestDBContainer

func TestMain(m *testing.M) {
	var cleanup func()
	testDB, cleanup = test.SetupTestDatabaseForTestMain()
	defer cleanup()

	code := m.Run()

	os.Exit(code)
}

func TestGenerateConversationBriefing_Integration(t *testing.T) {
	pool := testDB.Pool
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})
	logger := slog.New(h).With("service", "conversation_briefing")

	t.Run("fetches conversations and inserts briefing, returns success=true", func(t *testing.T) {
		test.CleanupTables(t, pool)

		embedding := test.MakeEmbedding(0.1, 128)
		profileID := test.SeedProfile(t, pool, embedding)
		visitorID := test.SeedVisitor(t, pool, profileID, "alice", embedding)
		test.SeedConvos(t, pool, profileID, visitorID, "hello from alice")

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"message": map[string]string{"role": "assistant", "content": "Alice is a loyal customer"},
			})
		}))
		defer srv.Close()

		s := handler.NewConvoBriefingServer(logger, db.NewDBPool(pool, logger), srv.URL)
		resp, err := s.GenerateConversationBriefing(context.Background(), &cb.GenerateConversationBriefingRequest{
			ProfileId:  profileID,
			VisitorIds: []int32{visitorID},
		})

		assert.Nil(t, err)
		assert.True(t, resp.Success)

		text, found := test.FetchBriefing(t, pool, profileID, visitorID)
		assert.True(t, found, "briefing should be written to DB")
		assert.Equal(t, "Alice is a loyal customer", text)
	})

	t.Run("returns success=false and no DB write when no conversations exist", func(t *testing.T) {
		test.CleanupTables(t, pool)

		embedding := test.MakeEmbedding(0.1, 128)
		profileID := test.SeedProfile(t, pool, embedding)
		visitorID := test.SeedVisitor(t, pool, profileID, "bob", embedding)

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"message": map[string]string{"content": "should not be called"},
			})
		}))
		defer srv.Close()

		s := handler.NewConvoBriefingServer(logger, db.NewDBPool(pool, logger), srv.URL)
		resp, err := s.GenerateConversationBriefing(context.Background(), &cb.GenerateConversationBriefingRequest{
			ProfileId:  profileID,
			VisitorIds: []int32{visitorID},
		})

		assert.Error(t, err)
		assert.False(t, resp.Success)

		_, found := test.FetchBriefing(t, pool, profileID, visitorID)
		assert.False(t, found, "no briefing should be written when there are no conversations")
	})

	t.Run("returns success=false when LLM call fails", func(t *testing.T) {
		test.CleanupTables(t, pool)

		embedding := test.MakeEmbedding(0.1, 128)
		profileID := test.SeedProfile(t, pool, embedding)
		visitorID := test.SeedVisitor(t, pool, profileID, "carol", embedding)
		test.SeedConvos(t, pool, profileID, visitorID, "hello from carol")

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()

		s := handler.NewConvoBriefingServer(logger, db.NewDBPool(pool, logger), srv.URL)
		resp, err := s.GenerateConversationBriefing(context.Background(), &cb.GenerateConversationBriefingRequest{
			ProfileId:  profileID,
			VisitorIds: []int32{visitorID},
		})

		assert.Error(t, err)
		assert.False(t, resp.Success)

		_, found := test.FetchBriefing(t, pool, profileID, visitorID)
		assert.False(t, found, "no briefing should be written when LLM fails")
	})

	t.Run("writes correct briefings for multiple visitors", func(t *testing.T) {
		test.CleanupTables(t, pool)

		embedding := test.MakeEmbedding(0.1, 128)
		profileID := test.SeedProfile(t, pool, embedding)
		visitorID1 := test.SeedVisitor(t, pool, profileID, "visitor1", embedding)
		visitorID2 := test.SeedVisitor(t, pool, profileID, "visitor2", embedding)
		test.SeedConvos(t, pool, profileID, visitorID1, "visitor1 message")
		test.SeedConvos(t, pool, profileID, visitorID2, "visitor2 message")

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

		s := handler.NewConvoBriefingServer(logger, db.NewDBPool(pool, logger), srv.URL)
		resp, err := s.GenerateConversationBriefing(context.Background(), &cb.GenerateConversationBriefingRequest{
			ProfileId:  profileID,
			VisitorIds: []int32{visitorID1, visitorID2},
		})

		assert.Nil(t, err)
		assert.True(t, resp.Success)

		text1, found1 := test.FetchBriefing(t, pool, profileID, visitorID1)
		assert.True(t, found1)

		text2, found2 := test.FetchBriefing(t, pool, profileID, visitorID2)
		assert.True(t, found2)

		assert.ElementsMatch(t, []string{"briefing for visitor1", "briefing for visitor2"}, []string{text1, text2})
	})
}

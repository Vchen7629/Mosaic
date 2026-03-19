//go:build integration

package db_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"mosaic-conversation-briefing.com/internal/db"
	"mosaic-conversation-briefing.com/internal/test"
)

var testDB *test.TestDBContainer

// Setup and teardown database for all tests in this package
func TestMain(m *testing.M) {
	var cleanup func()
	testDB, cleanup = test.SetupTestDatabaseForTestMain()
	defer cleanup()

	// Run all tests
	code := m.Run()

	os.Exit(code)
}

func TestFetchRecentConversations(t *testing.T) {
	pool := testDB.Pool
	dbPool := db.NewDBPool(pool)

	t.Run("returns all conversations for profile id and visitor id", func(t *testing.T) {
		test.CleanupTables(t, pool)

		embedding := test.MakeEmbedding(0.2, 128)
		profileID := test.SeedProfile(t, pool, embedding)
		visitorID1 := test.SeedVisitor(t, pool, profileID, "visitor 1", embedding)
		visitorID2 := test.SeedVisitor(t, pool, profileID, "visitor 2", embedding)

		test.SeedConvos(t, pool, profileID, visitorID1, "this is convo 1 for visitor 1")
		test.SeedConvos(t, pool, profileID, visitorID1, "this is convo 2 for visitor 1")

		test.SeedConvos(t, pool, profileID, visitorID2, "this is convo 1 for visitor 2")

		visitorData, err := dbPool.FetchRecentConversations(profileID, []int32{visitorID1, visitorID2})

		assert.Nil(t, err)
		assert.Len(t, visitorData, 2)

		for _, v := range visitorData {
			switch v.VisitorID {
			case visitorID1:
				assert.ElementsMatch(t, []string{
					"this is convo 1 for visitor 1",
					"this is convo 2 for visitor 1",
				}, v.ConvoList)
			case visitorID2:
				assert.ElementsMatch(t, []string{"this is convo 1 for visitor 2"}, v.ConvoList)
			default:
				t.Errorf("unexpected visitorID: %d", v.VisitorID)
			}
		}
	})

	t.Run("Only fetches convos for patient id and visitor id specified", func(t *testing.T) {
		test.CleanupTables(t, pool)

		embedding := test.MakeEmbedding(0.2, 128)
		profileID1 := test.SeedProfile(t, pool, embedding)
		profileID2 := test.SeedProfile(t, pool, embedding)
		visitorID1 := test.SeedVisitor(t, pool, profileID1, "visitor 1", embedding)
		visitorID2 := test.SeedVisitor(t, pool, profileID2, "visitor 2", embedding)

		test.SeedConvos(t, pool, profileID1, visitorID1, "this is convo 1 for visitor 1")

		test.SeedConvos(t, pool, profileID2, visitorID2, "this is convo 1 for visitor 2")

		visitorData, err := dbPool.FetchRecentConversations(profileID1, []int32{visitorID1})

		assert.Nil(t, err)
		assert.Len(t, visitorData, 1)

		assert.ElementsMatch(t, []string{"this is convo 1 for visitor 1"}, visitorData[0].ConvoList)
	})

	t.Run("returns empty list when trying to fetch for a patient that doesnt exist", func(t *testing.T) {
		test.CleanupTables(t, pool)
		embedding := test.MakeEmbedding(0.1, 128)

		profileID := test.SeedProfile(t, pool, embedding)
		_ = test.SeedVisitor(t, pool, profileID, "test visitor", embedding)

		visitorData, err := dbPool.FetchRecentConversations(profileID, []int32{99})

		assert.Nil(t, err)
		assert.Equal(t, 0, len(visitorData))
	})
}

func TestInsertBriefing(t *testing.T) {
	pool := testDB.Pool
	dbPool := db.NewDBPool(pool)

	t.Run("inserts briefing for a single visitor", func(t *testing.T) {
		test.CleanupTables(t, pool)

		embedding := test.MakeEmbedding(0.1, 128)
		profileID := test.SeedProfile(t, pool, embedding)
		visitorID := test.SeedVisitor(t, pool, profileID, "visitor 1", embedding)

		err := dbPool.InsertBriefing(profileID, map[int32]string{
			visitorID: "briefing for visitor 1",
		})

		assert.Nil(t, err)
		text, found := test.FetchBriefing(t, pool, profileID, visitorID)
		assert.True(t, found)
		assert.Equal(t, "briefing for visitor 1", text)
	})

	t.Run("inserts briefings for multiple visitors", func(t *testing.T) {
		test.CleanupTables(t, pool)

		embedding := test.MakeEmbedding(0.1, 128)
		profileID := test.SeedProfile(t, pool, embedding)
		visitorID1 := test.SeedVisitor(t, pool, profileID, "visitor 1", embedding)
		visitorID2 := test.SeedVisitor(t, pool, profileID, "visitor 2", embedding)

		err := dbPool.InsertBriefing(profileID, map[int32]string{
			visitorID1: "briefing for visitor 1",
			visitorID2: "briefing for visitor 2",
		})

		assert.Nil(t, err)

		text1, found1 := test.FetchBriefing(t, pool, profileID, visitorID1)
		assert.True(t, found1)
		assert.Equal(t, "briefing for visitor 1", text1)

		text2, found2 := test.FetchBriefing(t, pool, profileID, visitorID2)
		assert.True(t, found2)
		assert.Equal(t, "briefing for visitor 2", text2)
	})

	t.Run("upserts briefing text for existing profile and visitor", func(t *testing.T) {
		test.CleanupTables(t, pool)

		embedding := test.MakeEmbedding(0.1, 128)
		profileID := test.SeedProfile(t, pool, embedding)
		visitorID := test.SeedVisitor(t, pool, profileID, "visitor 1", embedding)

		err := dbPool.InsertBriefing(profileID, map[int32]string{
			visitorID: "original briefing",
		})
		assert.Nil(t, err)

		err = dbPool.InsertBriefing(profileID, map[int32]string{
			visitorID: "updated briefing",
		})
		assert.Nil(t, err)

		text, found := test.FetchBriefing(t, pool, profileID, visitorID)
		assert.True(t, found)
		assert.Equal(t, "updated briefing", text)
	})

	t.Run("returns error when all inserts fail due to invalid profile id", func(t *testing.T) {
		test.CleanupTables(t, pool)

		embedding := test.MakeEmbedding(0.1, 128)
		profileID := test.SeedProfile(t, pool, embedding)
		visitorID := test.SeedVisitor(t, pool, profileID, "visitor 1", embedding)

		err := dbPool.InsertBriefing(99999, map[int32]string{
			visitorID: "briefing",
		})

		assert.NotNil(t, err)
		assert.Equal(t, "all briefing inserts failed", err.Error())
	})
}

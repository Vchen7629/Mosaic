//go:build integration

package db_test

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mosaic-face-detection.com/internal/db"
	"mosaic-face-detection.com/internal/test"
)

func TestUpsertSession(t *testing.T) {
	pool := testDB.Pool
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	dbPool := db.NewDBPool(pool, logger)

	t.Run("inserts session with correct profile_id and session_token", func(t *testing.T) {
		test.CleanupTables(t, pool)

		profileID := test.SeedProfile(t, pool, test.MakeEmbedding(0.1, 128))
		token := "insert-token-abc"

		err := dbPool.UpsertSession(profileID, token)
		require.NoError(t, err)

		result, err := dbPool.FetchProfileIDWithSession(token)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, profileID, *result)
	})

	t.Run("upserts session_token on profile_id conflict", func(t *testing.T) {
		test.CleanupTables(t, pool)

		profileID := test.SeedProfile(t, pool, test.MakeEmbedding(0.1, 128))
		firstToken := "first-token"
		updatedToken := "updated-token"

		err := dbPool.UpsertSession(profileID, firstToken)
		require.NoError(t, err)

		err = dbPool.UpsertSession(profileID, updatedToken)
		require.NoError(t, err)

		result, err := dbPool.FetchSessionWithProfileID(profileID)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, updatedToken, *result)
	})

	t.Run("returns error when pool is closed", func(t *testing.T) {
		test.CleanupTables(t, pool)

		closedPool := test.SetupTestDatabase(t)
		closedPool.Pool.Close()

		closedDB := db.NewDBPool(closedPool.Pool, logger)
		err := closedDB.UpsertSession(1, "some-token")
		assert.Error(t, err)
	})
}

func TestFetchProfileIDWithSession(t *testing.T) {
	pool := testDB.Pool
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	dbPool := db.NewDBPool(pool, logger)

	t.Run("returns profileID for existing session token", func(t *testing.T) {
		test.CleanupTables(t, pool)

		profileID := test.SeedProfile(t, pool, test.MakeEmbedding(0.1, 128))
		token := test.SeedSession(t, pool, profileID)

		result, err := dbPool.FetchProfileIDWithSession(token)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, profileID, *result)
	})

	t.Run("returns error for non-existent session token", func(t *testing.T) {
		test.CleanupTables(t, pool)

		result, err := dbPool.FetchProfileIDWithSession("nonexistent-token")
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestFetchSessionWithProfileID(t *testing.T) {
	pool := testDB.Pool
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	dbPool := db.NewDBPool(pool, logger)

	t.Run("returns session token for existing profile", func(t *testing.T) {
		test.CleanupTables(t, pool)

		profileID := test.SeedProfile(t, pool, test.MakeEmbedding(0.1, 128))
		token := test.SeedSession(t, pool, profileID)

		result, err := dbPool.FetchSessionWithProfileID(profileID)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, token, *result)
	})

	t.Run("returns error for profile with no session", func(t *testing.T) {
		test.CleanupTables(t, pool)

		profileID := test.SeedProfile(t, pool, test.MakeEmbedding(0.1, 128))

		result, err := dbPool.FetchSessionWithProfileID(profileID)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

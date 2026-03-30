//go:build integration

package db_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mosaic-conversation-briefing.com/internal/db"
	"mosaic-conversation-briefing.com/internal/test"
)

func TestFetchProfileIDWithSession(t *testing.T) {
	pool := testDB.Pool
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	dbPool := db.NewDBPool(pool, logger)

	t.Run("returns profileID for existing session token", func(t *testing.T) {
		test.CleanupTables(t, pool)

		profileID := test.SeedProfile(t, pool, test.MakeEmbedding(0.1, 128))
		token := test.SeedSession(t, pool, profileID)

		result, err := dbPool.FetchProfileIDWithSession(context.Background(), token)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, profileID, *result)
	})

	t.Run("returns error for non-existent session token", func(t *testing.T) {
		test.CleanupTables(t, pool)

		result, err := dbPool.FetchProfileIDWithSession(context.Background(), "nonexistent-token")
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

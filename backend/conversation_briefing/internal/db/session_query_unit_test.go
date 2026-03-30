//go:build unit

package db

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestDBPool() *DBPool {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	return &DBPool{pool: nil, logger: logger}
}

func TestFetchProfileIDWithSession_MissingParams(t *testing.T) {
	dbPool := newTestDBPool()

	t.Run("returns error for empty sessionToken", func(t *testing.T) {
		result, err := dbPool.FetchProfileIDWithSession(context.Background(), "")
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, "missing sessionToken param", err.Error())
	})
}

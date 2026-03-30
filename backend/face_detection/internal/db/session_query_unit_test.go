package db

import (
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

func TestUpsertSessionMissingParams(t *testing.T) {
	dbPool := newTestDBPool()
	tests := []struct {
		name                 string
		profileID            int32
		sessionToken, errMsg string
	}{
		{"returns error for 0 profileID", 0, "some-token", "profileID must be positive"},
		{"returns error for negative profileID", -1, "some-token", "profileID must be positive"},
		{"returns error for empty sessionToken", 1, "", "missing sessionToken param"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := dbPool.UpsertSession(tt.profileID, tt.sessionToken)
			require.Error(t, err)
			assert.Equal(t, tt.errMsg, err.Error())
		})
	}
}

func TestFetchProfileIDWithSession_MissingParams(t *testing.T) {
	dbPool := newTestDBPool()

	t.Run("returns error for empty sessionToken", func(t *testing.T) {
		result, err := dbPool.FetchProfileIDWithSession("")
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, "missing sessionToken param", err.Error())
	})
}

func TestFetchSessionWithProfileID_MissingParams(t *testing.T) {
	dbPool := newTestDBPool()

	t.Run("returns error for non-positive profileID", func(t *testing.T) {
		result, err := dbPool.FetchSessionWithProfileID(0)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, "profileID must be positive", err.Error())

		result, err = dbPool.FetchSessionWithProfileID(-1)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, "profileID must be positive", err.Error())
	})
}

//go:build integration

package cache_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mosaic-face-detection.com/internal/cache"
	"mosaic-face-detection.com/internal/observability"
	"mosaic-face-detection.com/internal/test"
)

func TestCreateNewProfileSession(t *testing.T) {
	t.Run("stores session token in cache and can be retrieved", func(t *testing.T) {
		test.FlushCache(t, testCacheContainer.Client)

		ctx := context.Background()
		profileID := int32(42)
		token := "test-token-abc"

		err := cache.CreateNewProfileSession(ctx, profileID, testCacheContainer.Client, token)
		require.NoError(t, err)

		result, err := cache.FetchProfileIDFromCache(ctx, testCacheContainer.Client, token)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, profileID, *result)
	})

	t.Run("different tokens are stored under separate keys", func(t *testing.T) {
		test.FlushCache(t, testCacheContainer.Client)

		ctx := context.Background()

		err := cache.CreateNewProfileSession(ctx, 1, testCacheContainer.Client, "token-1")
		require.NoError(t, err)
		err = cache.CreateNewProfileSession(ctx, 2, testCacheContainer.Client, "token-2")
		require.NoError(t, err)

		r1, err := cache.FetchProfileIDFromCache(ctx, testCacheContainer.Client, "token-1")
		require.NoError(t, err)
		assert.Equal(t, int32(1), *r1)

		r2, err := cache.FetchProfileIDFromCache(ctx, testCacheContainer.Client, "token-2")
		require.NoError(t, err)
		assert.Equal(t, int32(2), *r2)
	})

	t.Run("overwriting a token updates the stored profile ID", func(t *testing.T) {
		test.FlushCache(t, testCacheContainer.Client)

		ctx := context.Background()
		token := "shared-token"

		err := cache.CreateNewProfileSession(ctx, 10, testCacheContainer.Client, token)
		require.NoError(t, err)

		err = cache.CreateNewProfileSession(ctx, 99, testCacheContainer.Client, token)
		require.NoError(t, err)

		result, err := cache.FetchProfileIDFromCache(ctx, testCacheContainer.Client, token)
		require.NoError(t, err)
		assert.Equal(t, int32(99), *result)
	})
}

func TestFetchProfileIDFromCache(t *testing.T) {
	t.Run("returns error and increments error counter for missing key", func(t *testing.T) {
		test.FlushCache(t, testCacheContainer.Client)

		errsBefore := getCounterValue(t, observability.ErrorsTotal, "fetch_profile_id_from_cache")

		result, err := cache.FetchProfileIDFromCache(context.Background(), testCacheContainer.Client, "nonexistent-token")
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, errsBefore+1, getCounterValue(t, observability.ErrorsTotal, "fetch_profile_id_from_cache"))
	})

	t.Run("returns correct profile ID after it was stored", func(t *testing.T) {
		test.FlushCache(t, testCacheContainer.Client)

		ctx := context.Background()
		profileID := int32(7)
		token := "fetch-test-token"

		err := cache.CreateNewProfileSession(ctx, profileID, testCacheContainer.Client, token)
		require.NoError(t, err)

		result, err := cache.FetchProfileIDFromCache(ctx, testCacheContainer.Client, token)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, profileID, *result)
	})
}

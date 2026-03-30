//go:build integration

package cache_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Kagami/go-face"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mosaic-face-detection.com/internal/cache"
	"mosaic-face-detection.com/internal/observability"
	"mosaic-face-detection.com/internal/service"
	"mosaic-face-detection.com/internal/test"
)

func makeVisitorFaces(id int32, name string) service.VisitorFaces {
	var emb face.Descriptor
	emb[0] = float32(id)
	return service.VisitorFaces{ID: id, Name: name, Embedding: emb}
}

func TestFetchAllVisitorDataWithCache(t *testing.T) {
	t.Run("cache hit does not call db and increments hit counter", func(t *testing.T) {
		test.FlushCache(t, testCacheContainer.Client)

		visitors := []service.VisitorFaces{makeVisitorFaces(1, "Alice"), makeVisitorFaces(2, "Bob")}
		dbCallCount := 0
		fetchFn := func() ([]service.VisitorFaces, error) {
			dbCallCount++
			return visitors, nil
		}

		ctx := context.Background()
		cache.FetchAllVisitorDataWithCache(ctx, 10, testCacheContainer.Client, testLogger, fetchFn)

		hitsBefore := getCounterValue(t, observability.CacheHitsTotal, "fetch_all_visitor_data")

		result, err := cache.FetchAllVisitorDataWithCache(ctx, 10, testCacheContainer.Client, testLogger, fetchFn)
		require.NoError(t, err)
		assert.Equal(t, visitors, result)
		assert.Equal(t, 1, dbCallCount, "db should not be called on cache hit")
		assert.Equal(t, hitsBefore+1, getCounterValue(t, observability.CacheHitsTotal, "fetch_all_visitor_data"))
	})

	t.Run("cache miss calls db and increments miss counter", func(t *testing.T) {
		test.FlushCache(t, testCacheContainer.Client)

		visitors := []service.VisitorFaces{makeVisitorFaces(5, "Charlie")}
		dbCallCount := 0
		fetchFn := func() ([]service.VisitorFaces, error) {
			dbCallCount++
			return visitors, nil
		}

		missesBefore := getCounterValue(t, observability.CacheMissesTotal, "fetch_all_visitor_data")

		result, err := cache.FetchAllVisitorDataWithCache(context.Background(), 20, testCacheContainer.Client, testLogger, fetchFn)
		require.NoError(t, err)
		assert.Equal(t, visitors, result)
		assert.Equal(t, 1, dbCallCount)
		assert.Equal(t, missesBefore+1, getCounterValue(t, observability.CacheMissesTotal, "fetch_all_visitor_data"))
	})

	t.Run("db error is returned", func(t *testing.T) {
		test.FlushCache(t, testCacheContainer.Client)

		dbErr := errors.New("visitor db error")
		fetchFn := func() ([]service.VisitorFaces, error) { return nil, dbErr }

		result, err := cache.FetchAllVisitorDataWithCache(context.Background(), 30, testCacheContainer.Client, testLogger, fetchFn)
		assert.ErrorIs(t, err, dbErr)
		assert.Nil(t, result)
	})

	t.Run("different profile IDs are cached under separate keys", func(t *testing.T) {
		test.FlushCache(t, testCacheContainer.Client)

		v1 := []service.VisitorFaces{makeVisitorFaces(1, "Alice")}
		v2 := []service.VisitorFaces{makeVisitorFaces(2, "Bob")}
		calls := map[int32]int{}

		makeFetch := func(id int32, data []service.VisitorFaces) func() ([]service.VisitorFaces, error) {
			return func() ([]service.VisitorFaces, error) {
				calls[id]++
				return data, nil
			}
		}

		ctx := context.Background()
		cache.FetchAllVisitorDataWithCache(ctx, 1, testCacheContainer.Client, testLogger, makeFetch(1, v1))
		cache.FetchAllVisitorDataWithCache(ctx, 2, testCacheContainer.Client, testLogger, makeFetch(2, v2))

		r1, _ := cache.FetchAllVisitorDataWithCache(ctx, 1, testCacheContainer.Client, testLogger, makeFetch(1, v1))
		r2, _ := cache.FetchAllVisitorDataWithCache(ctx, 2, testCacheContainer.Client, testLogger, makeFetch(2, v2))

		assert.Equal(t, v1, r1)
		assert.Equal(t, v2, r2)
		assert.Equal(t, 1, calls[1], "visitor profile 1 db should not be called again")
		assert.Equal(t, 1, calls[2], "visitor profile 2 db should not be called again")
	})
}

func TestAppendToVisitorDataCache(t *testing.T) {
	t.Run("appends entry to existing cache so next fetch is a cache hit with updated data", func(t *testing.T) {
		test.FlushCache(t, testCacheContainer.Client)

		initial := []service.VisitorFaces{makeVisitorFaces(1, "Alice")}
		newEntry := makeVisitorFaces(2, "Bob")

		ctx := context.Background()
		cache.FetchAllVisitorDataWithCache(ctx, 10, testCacheContainer.Client, testLogger, func() ([]service.VisitorFaces, error) {
			return initial, nil
		})

		cache.AppendToVisitorDataCache(ctx, 10, testCacheContainer.Client, newEntry)

		dbCallCount := 0
		result, err := cache.FetchAllVisitorDataWithCache(ctx, 10, testCacheContainer.Client, testLogger, func() ([]service.VisitorFaces, error) {
			dbCallCount++
			return initial, nil
		})

		require.NoError(t, err)
		assert.Equal(t, 0, dbCallCount, "should have hit cache after append")
		assert.Len(t, result, 2)
		assert.Equal(t, int32(1), result[0].ID)
		assert.Equal(t, int32(2), result[1].ID)
	})

	t.Run("is noop when cache key does not exist", func(t *testing.T) {
		test.FlushCache(t, testCacheContainer.Client)

		ctx := context.Background()
		cache.AppendToVisitorDataCache(ctx, 999, testCacheContainer.Client, makeVisitorFaces(99, "Ghost"))

		dbCallCount := 0
		result, err := cache.FetchAllVisitorDataWithCache(ctx, 999, testCacheContainer.Client, testLogger, func() ([]service.VisitorFaces, error) {
			dbCallCount++
			return []service.VisitorFaces{}, nil
		})

		require.NoError(t, err)
		assert.Equal(t, 1, dbCallCount, "db should be called since append to empty cache is a noop")
		assert.Empty(t, result)
	})

	t.Run("does not affect cache for a different profile ID", func(t *testing.T) {
		test.FlushCache(t, testCacheContainer.Client)

		v1 := []service.VisitorFaces{makeVisitorFaces(1, "Alice")}
		v2 := []service.VisitorFaces{makeVisitorFaces(2, "Bob")}

		ctx := context.Background()
		cache.FetchAllVisitorDataWithCache(ctx, 1, testCacheContainer.Client, testLogger, func() ([]service.VisitorFaces, error) { return v1, nil })
		cache.FetchAllVisitorDataWithCache(ctx, 2, testCacheContainer.Client, testLogger, func() ([]service.VisitorFaces, error) { return v2, nil })

		cache.AppendToVisitorDataCache(ctx, 1, testCacheContainer.Client, makeVisitorFaces(99, "Extra"))

		dbCallCount := 0
		result2, err := cache.FetchAllVisitorDataWithCache(ctx, 2, testCacheContainer.Client, testLogger, func() ([]service.VisitorFaces, error) {
			dbCallCount++
			return v2, nil
		})

		require.NoError(t, err)
		assert.Equal(t, 0, dbCallCount, "profile 2 cache should be unaffected")
		assert.Len(t, result2, 1)
		assert.Equal(t, int32(2), result2[0].ID)
	})
}

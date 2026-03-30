//go:build integration

package cache_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/Kagami/go-face"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mosaic-face-detection.com/internal/cache"
	"mosaic-face-detection.com/internal/observability"
	"mosaic-face-detection.com/internal/service"
	"mosaic-face-detection.com/internal/test"
)

var (
	testCacheContainer *test.TestCacheContainer
	testLogger         *slog.Logger
)

func TestMain(m *testing.M) {
	testLogger = slog.New(slog.NewTextHandler(os.Stderr, nil))

	observability.CacheHitsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "test_cache_hits_total",
	}, []string{"operation"})
	observability.CacheMissesTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "test_cache_misses_total",
	}, []string{"operation"})
	observability.ErrorsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "test_errors_total",
	}, []string{"operation"})
	observability.ProfileEmbFetchDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "test_profile_emb_fetch_duration_milliseconds",
	})

	var cleanup func()
	testCacheContainer, cleanup = test.SetupTestCacheForTestMain()
	defer cleanup()

	os.Exit(m.Run())
}

func getCounterValue(t *testing.T, counter *prometheus.CounterVec, labelValue string) float64 {
	t.Helper()
	var m dto.Metric
	err := counter.WithLabelValues(labelValue).Write(&m)
	require.NoError(t, err)
	return m.Counter.GetValue()
}

func makeProfileFaces(id int32) service.ProfileFaces {
	var emb face.Descriptor
	emb[0] = float32(id)
	return service.ProfileFaces{ID: id, Embedding: emb}
}

func TestFetchProfileFaceEmbForIDWithCache(t *testing.T) {
	t.Run("cache hit does not call db and increments hit counter", func(t *testing.T) {
		test.FlushCache(t, testCacheContainer.Client)

		profiles := []service.ProfileFaces{makeProfileFaces(1), makeProfileFaces(2)}
		dbCallCount := 0
		fetchFn := func() ([]service.ProfileFaces, error) {
			dbCallCount++
			return profiles, nil
		}

		ctx := context.Background()
		cache.FetchProfileFaceEmbForIDWithCache(ctx, 10, testCacheContainer.Client, testLogger, fetchFn)

		hitsBefore := getCounterValue(t, observability.CacheHitsTotal, "fetch_profile_face_emb_for_id")

		result, err := cache.FetchProfileFaceEmbForIDWithCache(ctx, 10, testCacheContainer.Client, testLogger, fetchFn)
		require.NoError(t, err)
		assert.Equal(t, profiles, result)
		assert.Equal(t, 1, dbCallCount, "db should not be called on cache hit")
		assert.Equal(t, hitsBefore+1, getCounterValue(t, observability.CacheHitsTotal, "fetch_profile_face_emb_for_id"))
	})

	t.Run("cache miss calls db and increments miss counter", func(t *testing.T) {
		test.FlushCache(t, testCacheContainer.Client)

		profiles := []service.ProfileFaces{makeProfileFaces(99)}
		dbCallCount := 0
		fetchFn := func() ([]service.ProfileFaces, error) {
			dbCallCount++
			return profiles, nil
		}

		missesBefore := getCounterValue(t, observability.CacheMissesTotal, "fetch_profile_face_emb_for_id")

		result, err := cache.FetchProfileFaceEmbForIDWithCache(context.Background(), 20, testCacheContainer.Client, testLogger, fetchFn)
		require.NoError(t, err)
		assert.Equal(t, profiles, result)
		assert.Equal(t, 1, dbCallCount)
		assert.Equal(t, missesBefore+1, getCounterValue(t, observability.CacheMissesTotal, "fetch_profile_face_emb_for_id"))
	})

	t.Run("db error is returned and increments error counter", func(t *testing.T) {
		test.FlushCache(t, testCacheContainer.Client)

		dbErr := errors.New("db connection failed")
		fetchFn := func() ([]service.ProfileFaces, error) { return nil, dbErr }

		errsBefore := getCounterValue(t, observability.ErrorsTotal, "fetch_profile_embeddings")

		result, err := cache.FetchProfileFaceEmbForIDWithCache(context.Background(), 30, testCacheContainer.Client, testLogger, fetchFn)
		assert.ErrorIs(t, err, dbErr)
		assert.Nil(t, result)
		assert.Equal(t, errsBefore+1, getCounterValue(t, observability.ErrorsTotal, "fetch_profile_embeddings"))
	})

	t.Run("different profile IDs are cached under separate keys", func(t *testing.T) {
		test.FlushCache(t, testCacheContainer.Client)

		profiles1 := []service.ProfileFaces{makeProfileFaces(1)}
		profiles2 := []service.ProfileFaces{makeProfileFaces(2)}
		calls := map[int32]int{}

		makeFetch := func(id int32, data []service.ProfileFaces) func() ([]service.ProfileFaces, error) {
			return func() ([]service.ProfileFaces, error) {
				calls[id]++
				return data, nil
			}
		}

		ctx := context.Background()
		cache.FetchProfileFaceEmbForIDWithCache(ctx, 1, testCacheContainer.Client, testLogger, makeFetch(1, profiles1))
		cache.FetchProfileFaceEmbForIDWithCache(ctx, 2, testCacheContainer.Client, testLogger, makeFetch(2, profiles2))

		r1, _ := cache.FetchProfileFaceEmbForIDWithCache(ctx, 1, testCacheContainer.Client, testLogger, makeFetch(1, profiles1))
		r2, _ := cache.FetchProfileFaceEmbForIDWithCache(ctx, 2, testCacheContainer.Client, testLogger, makeFetch(2, profiles2))

		assert.Equal(t, profiles1, r1)
		assert.Equal(t, profiles2, r2)
		assert.Equal(t, 1, calls[1], "profile 1 db should not be called again")
		assert.Equal(t, 1, calls[2], "profile 2 db should not be called again")
	})
}

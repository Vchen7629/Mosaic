//go:build integration

package service_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"mosaic-face-detection.com/internal/service"
)

func TestNewRecognizerPool(t *testing.T) {
	t.Run("creates pool with correct number of instances", func(t *testing.T) {
		pool, err := service.NewRecognizerPool(testModelsDir, 3)
		assert.NoError(t, err)
		defer pool.Close()

		// drain all instances to verify there are exactly 3
		rec1 := pool.Acquire()
		rec2 := pool.Acquire()
		rec3 := pool.Acquire()

		assert.NotNil(t, rec1)
		assert.NotNil(t, rec2)
		assert.NotNil(t, rec3)

		pool.Release(rec1)
		pool.Release(rec2)
		pool.Release(rec3)
	})

	t.Run("returns error for invalid models dir", func(t *testing.T) {
		pool, err := service.NewRecognizerPool("/idk", 1)

		assert.Error(t, err)
		assert.Nil(t, pool)
	})
}

func TestRecognizerPoolAcquireRelease(t *testing.T) {
	t.Run("acquire returns non-nil recognizer and release returns same instance", func(t *testing.T) {
		pool, err := service.NewRecognizerPool(testModelsDir, 1)
		assert.NoError(t, err)
		defer pool.Close()
		
		rec1 := pool.Acquire()
		pool.Release(rec1)

		rec2 := pool.Acquire()
		defer pool.Release(rec2)

		assert.NotNil(t, rec1)
		assert.Same(t, rec1, rec2, "should get back same pointer after release")
	})

	t.Run("blocks when exhausted, unblocks on release", func(t *testing.T) {
		pool, err := service.NewRecognizerPool(testModelsDir, 1)
		assert.NoError(t, err)
		defer pool.Close()

		rec := pool.Acquire()
		acquired := make(chan struct{})
		go func() {
			r := pool.Acquire()
			close(acquired)
			pool.Release(r)
		}()

		select {
		case <-acquired:
			t.Fatal("should have blocked while pool was exhausted")
		case <-time.After(50 * time.Millisecond):
		}

		pool.Release(rec)

		select {
		case <-acquired:
		case <-time.After(500 * time.Millisecond):
			t.Fatal("should have unblocked after release")
		}
	})

	t.Run("concurrent goroutines never share the same instance", func(t *testing.T) {
		pool, err := service.NewRecognizerPool(testModelsDir, 3)
		assert.NoError(t, err)
		defer pool.Close()

		var mu sync.Mutex
		active := make(map[string]bool)
		overlap := false

		var wg sync.WaitGroup
		for range 10 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				rec := pool.Acquire()
				ptr := fmt.Sprintf("%p", rec)

				mu.Lock()
				if active[ptr] {
					overlap = true
				}
				active[ptr] = true
				mu.Unlock()

				time.Sleep(10 * time.Millisecond)

				mu.Lock()
				delete(active, ptr)
				mu.Unlock()

				pool.Release(rec)
			}()
		}

		wg.Wait()
		assert.False(t, overlap, "same instance held by two goroutines simultaneously")
	})
}
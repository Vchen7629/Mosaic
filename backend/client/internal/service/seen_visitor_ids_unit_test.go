//go:build unit

package service

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddSeenVisitor(t *testing.T) {
	t.Run("Added visitor is marked as seen", func(t *testing.T) {
		t.Cleanup(ClearSeenVisitors)

		AddSeenVisitor(1)

		assert.True(t, HasSeenVisitor(1), "visitor should be marked as seen after AddSeenVisitor")
	})

	t.Run("Adding multiple distinct visitors marks all as seen", func(t *testing.T) {
		t.Cleanup(ClearSeenVisitors)

		AddSeenVisitor(1)
		AddSeenVisitor(2)
		AddSeenVisitor(3)

		assert.True(t, HasSeenVisitor(1))
		assert.True(t, HasSeenVisitor(2))
		assert.True(t, HasSeenVisitor(3))
	})

	t.Run("Adding the same visitor twice does not cause errors", func(t *testing.T) {
		t.Cleanup(ClearSeenVisitors)

		AddSeenVisitor(1)
		AddSeenVisitor(1)

		assert.True(t, HasSeenVisitor(1), "visitor should still be marked as seen")
	})

	t.Run("Concurrent adds do not produce data races", func(t *testing.T) {
		t.Cleanup(ClearSeenVisitors)

		const goroutines = 50
		var wg sync.WaitGroup
		wg.Add(goroutines)

		for i := range goroutines {
			go func(id int32) {
				defer wg.Done()
				AddSeenVisitor(id)
			}(int32(i))
		}

		wg.Wait()

		for i := range goroutines {
			assert.True(t, HasSeenVisitor(int32(i)), "visitor %d should be marked as seen", i)
		}
	})
}

func TestHasSeenVisitor(t *testing.T) {
	t.Run("Returns false for visitor that has not been added", func(t *testing.T) {
		t.Cleanup(ClearSeenVisitors)

		assert.False(t, HasSeenVisitor(99), "unseen visitor should return false")
	})

	t.Run("Returns true for visitor that has been added", func(t *testing.T) {
		t.Cleanup(ClearSeenVisitors)

		AddSeenVisitor(42)

		assert.True(t, HasSeenVisitor(42), "seen visitor should return true")
	})

	t.Run("Concurrent reads do not produce data races", func(t *testing.T) {
		t.Cleanup(ClearSeenVisitors)
		AddSeenVisitor(1)

		const goroutines = 50
		var wg sync.WaitGroup
		wg.Add(goroutines)

		for range goroutines {
			go func() {
				defer wg.Done()
				HasSeenVisitor(1)
			}()
		}

		wg.Wait()
	})
}

func TestGetSeenVisitors(t *testing.T) {
	t.Run("Returns empty slice when no visitors have been added", func(t *testing.T) {
		t.Cleanup(ClearSeenVisitors)

		assert.Empty(t, GetSeenVisitors(), "should return empty slice when map is empty")
	})

	t.Run("Returns all added visitor IDs", func(t *testing.T) {
		t.Cleanup(ClearSeenVisitors)

		AddSeenVisitor(1)
		AddSeenVisitor(2)
		AddSeenVisitor(3)

		assert.ElementsMatch(t, []int32{1, 2, 3}, GetSeenVisitors(), "should return all added visitor IDs")
	})

	t.Run("Concurrent reads do not produce data races", func(t *testing.T) {
		t.Cleanup(ClearSeenVisitors)
		AddSeenVisitor(1)
		AddSeenVisitor(2)

		const goroutines = 50
		var wg sync.WaitGroup
		wg.Add(goroutines)

		for range goroutines {
			go func() {
				defer wg.Done()
				GetSeenVisitors()
			}()
		}

		wg.Wait()
	})
}

func TestClearSeenVisitors(t *testing.T) {
	t.Run("Clears all visitors from the map", func(t *testing.T) {
		AddSeenVisitor(1)
		AddSeenVisitor(2)

		ClearSeenVisitors()

		assert.Empty(t, GetSeenVisitors(), "map should be empty after clear")
		assert.False(t, HasSeenVisitor(1), "previously added visitor should no longer be seen")
		assert.False(t, HasSeenVisitor(2), "previously added visitor should no longer be seen")
	})

	t.Run("Clearing an already empty map does not panic", func(t *testing.T) {
		t.Cleanup(ClearSeenVisitors)
		ClearSeenVisitors()

		assert.NotPanics(t, ClearSeenVisitors, "clearing an empty map should not panic")
	})

	t.Run("Concurrent clears do not produce data races", func(t *testing.T) {
		t.Cleanup(ClearSeenVisitors)

		const goroutines = 20
		var wg sync.WaitGroup
		wg.Add(goroutines)

		for range goroutines {
			go func() {
				defer wg.Done()
				ClearSeenVisitors()
			}()
		}

		wg.Wait()
	})
}

// cross-function tests for concurrent access across Add, Get, Has, and Clear
func TestSeenFaceIds_CrossFunction(t *testing.T) {
	t.Run("Concurrent Add and HasSeenVisitor do not produce data races", func(t *testing.T) {
		t.Cleanup(ClearSeenVisitors)

		const goroutines = 50
		var wg sync.WaitGroup
		wg.Add(goroutines * 2)

		for i := range goroutines {
			go func(id int32) {
				defer wg.Done()
				AddSeenVisitor(id)
			}(int32(i))

			go func(id int32) {
				defer wg.Done()
				HasSeenVisitor(id)
			}(int32(i))
		}

		wg.Wait()
	})

	t.Run("Concurrent Add and GetSeenVisitors do not produce data races", func(t *testing.T) {
		t.Cleanup(ClearSeenVisitors)

		const goroutines = 50
		var wg sync.WaitGroup
		wg.Add(goroutines * 2)

		for i := range goroutines {
			go func(id int32) {
				defer wg.Done()
				AddSeenVisitor(id)
			}(int32(i))

			go func() {
				defer wg.Done()
				GetSeenVisitors()
			}()
		}

		wg.Wait()
	})

	t.Run("Concurrent Add and ClearSeenVisitors do not produce data races", func(t *testing.T) {
		t.Cleanup(ClearSeenVisitors)

		const goroutines = 50
		var wg sync.WaitGroup
		wg.Add(goroutines * 2)

		for i := range goroutines {
			go func(id int32) {
				defer wg.Done()
				AddSeenVisitor(id)
			}(int32(i))

			go func() {
				defer wg.Done()
				ClearSeenVisitors()
			}()
		}

		wg.Wait()
	})

	t.Run("Visitors added after a clear are correctly tracked", func(t *testing.T) {
		t.Cleanup(ClearSeenVisitors)

		AddSeenVisitor(1)
		AddSeenVisitor(2)
		ClearSeenVisitors()
		AddSeenVisitor(3)

		assert.False(t, HasSeenVisitor(1), "visitor added before clear should no longer be seen")
		assert.False(t, HasSeenVisitor(2), "visitor added before clear should no longer be seen")
		assert.True(t, HasSeenVisitor(3), "visitor added after clear should be seen")
	})
}

//go:build unit

package service

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddNewProfileSession(t *testing.T) {
	t.Run("Added session is retrievable by both profileID and token", func(t *testing.T) {
		t.Cleanup(ClearSessions)

		AddNewProfileSession("token1", int32(1))

		assert.Equal(t, "token1", FetchSessionToken(int32(1)))
		profileID, exists := FetchProfileIDWithSession("token1")
		assert.True(t, exists)
		assert.Equal(t, int32(1), profileID)
	})

	t.Run("Adding multiple distinct sessions stores all correctly", func(t *testing.T) {
		t.Cleanup(ClearSessions)

		AddNewProfileSession("token1", int32(1))
		AddNewProfileSession("token2", int32(2))
		AddNewProfileSession("token3", int32(3))

		assert.Equal(t, "token1", FetchSessionToken(int32(1)))
		assert.Equal(t, "token2", FetchSessionToken(int32(2)))
		assert.Equal(t, "token3", FetchSessionToken(int32(3)))

		id1, ok1 := FetchProfileIDWithSession("token1")
		id2, ok2 := FetchProfileIDWithSession("token2")
		id3, ok3 := FetchProfileIDWithSession("token3")
		assert.True(t, ok1)
		assert.True(t, ok2)
		assert.True(t, ok3)
		assert.Equal(t, int32(1), id1)
		assert.Equal(t, int32(2), id2)
		assert.Equal(t, int32(3), id3)
	})

	t.Run("Overwriting a profileID updates the token in both maps", func(t *testing.T) {
		t.Cleanup(ClearSessions)

		AddNewProfileSession("old-token", int32(1))
		AddNewProfileSession("new-token", int32(1))

		assert.Equal(t, "new-token", FetchSessionToken(int32(1)))
		_, oldExists := FetchProfileIDWithSession("old-token")
		id, newExists := FetchProfileIDWithSession("new-token")
		assert.False(t, oldExists, "old token should be overwritten")
		assert.True(t, newExists)
		assert.Equal(t, int32(1), id)
	})

	t.Run("Concurrent adds do not produce data races", func(t *testing.T) {
		t.Cleanup(ClearSessions)

		const goroutines = 50
		var wg sync.WaitGroup
		wg.Add(goroutines)

		for i := range goroutines {
			go func(id int32) {
				defer wg.Done()
				AddNewProfileSession("token", id)
			}(int32(i))
		}

		wg.Wait()
	})
}

func TestFetchSessionToken(t *testing.T) {
	t.Run("Returns empty string for unknown profileID", func(t *testing.T) {
		t.Cleanup(ClearSessions)

		assert.Empty(t, FetchSessionToken(int32(99)))
	})

	t.Run("Returns correct token for known profileID", func(t *testing.T) {
		t.Cleanup(ClearSessions)

		AddNewProfileSession("my-token", int32(5))

		assert.Equal(t, "my-token", FetchSessionToken(int32(5)))
	})

	t.Run("Concurrent reads do not produce data races", func(t *testing.T) {
		t.Cleanup(ClearSessions)
		AddNewProfileSession("token", int32(1))

		const goroutines = 50
		var wg sync.WaitGroup
		wg.Add(goroutines)

		for range goroutines {
			go func() {
				defer wg.Done()
				FetchSessionToken(int32(1))
			}()
		}

		wg.Wait()
	})
}

func TestFetchProfileIDWithSession(t *testing.T) {
	t.Run("Returns false for unknown session token", func(t *testing.T) {
		t.Cleanup(ClearSessions)

		_, exists := FetchProfileIDWithSession("nonexistent")
		assert.False(t, exists)
	})

	t.Run("Returns correct profileID and true for known token", func(t *testing.T) {
		t.Cleanup(ClearSessions)

		AddNewProfileSession("my-token", int32(7))

		profileID, exists := FetchProfileIDWithSession("my-token")
		assert.True(t, exists)
		assert.Equal(t, int32(7), profileID)
	})

	t.Run("Concurrent reads do not produce data races", func(t *testing.T) {
		t.Cleanup(ClearSessions)
		AddNewProfileSession("token", int32(1))

		const goroutines = 50
		var wg sync.WaitGroup
		wg.Add(goroutines)

		for range goroutines {
			go func() {
				defer wg.Done()
				FetchProfileIDWithSession("token")
			}()
		}

		wg.Wait()
	})
}

func TestClearSessions(t *testing.T) {
	t.Run("Clears all sessions from both maps", func(t *testing.T) {
		AddNewProfileSession("token1", int32(1))
		AddNewProfileSession("token2", int32(2))

		ClearSessions()

		assert.Empty(t, FetchSessionToken(int32(1)))
		assert.Empty(t, FetchSessionToken(int32(2)))
		_, ok1 := FetchProfileIDWithSession("token1")
		_, ok2 := FetchProfileIDWithSession("token2")
		assert.False(t, ok1)
		assert.False(t, ok2)
	})

	t.Run("Clearing an already empty map does not panic", func(t *testing.T) {
		t.Cleanup(ClearSessions)
		ClearSessions()

		assert.NotPanics(t, ClearSessions)
	})

	t.Run("Sessions added after a clear are correctly tracked", func(t *testing.T) {
		t.Cleanup(ClearSessions)

		AddNewProfileSession("token1", int32(1))
		ClearSessions()
		AddNewProfileSession("token2", int32(2))

		assert.Empty(t, FetchSessionToken(int32(1)), "session added before clear should be gone")
		_, ok1 := FetchProfileIDWithSession("token1")
		assert.False(t, ok1, "token added before clear should be gone")

		assert.Equal(t, "token2", FetchSessionToken(int32(2)))
		id, ok2 := FetchProfileIDWithSession("token2")
		assert.True(t, ok2)
		assert.Equal(t, int32(2), id)
	})

	t.Run("Concurrent clears do not produce data races", func(t *testing.T) {
		t.Cleanup(ClearSessions)

		const goroutines = 20
		var wg sync.WaitGroup
		wg.Add(goroutines)

		for range goroutines {
			go func() {
				defer wg.Done()
				ClearSessions()
			}()
		}

		wg.Wait()
	})
}

func TestSessions_CrossFunction(t *testing.T) {
	t.Run("Concurrent Add and FetchSessionToken do not produce data races", func(t *testing.T) {
		t.Cleanup(ClearSessions)

		const goroutines = 50
		var wg sync.WaitGroup
		wg.Add(goroutines * 2)

		for i := range goroutines {
			go func(id int32) {
				defer wg.Done()
				AddNewProfileSession("token", id)
			}(int32(i))

			go func(id int32) {
				defer wg.Done()
				FetchSessionToken(id)
			}(int32(i))
		}

		wg.Wait()
	})

	t.Run("Concurrent Add and FetchProfileIDWithSession do not produce data races", func(t *testing.T) {
		t.Cleanup(ClearSessions)

		const goroutines = 50
		var wg sync.WaitGroup
		wg.Add(goroutines * 2)

		for i := range goroutines {
			go func(id int32) {
				defer wg.Done()
				AddNewProfileSession("token", id)
			}(int32(i))

			go func() {
				defer wg.Done()
				FetchProfileIDWithSession("token")
			}()
		}

		wg.Wait()
	})

	t.Run("Concurrent Add and ClearSessions do not produce data races", func(t *testing.T) {
		t.Cleanup(ClearSessions)

		const goroutines = 50
		var wg sync.WaitGroup
		wg.Add(goroutines * 2)

		for i := range goroutines {
			go func(id int32) {
				defer wg.Done()
				AddNewProfileSession("token", id)
			}(int32(i))

			go func() {
				defer wg.Done()
				ClearSessions()
			}()
		}

		wg.Wait()
	})
}

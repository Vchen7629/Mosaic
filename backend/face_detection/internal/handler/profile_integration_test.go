//go:build integration

package handler_test

import (
	"testing"

	"mosaic-face-detection.com/internal/db"
	"mosaic-face-detection.com/internal/test"
)

var testDB *test.TestDBContainer

func TestSyncProfile(t *testing.T) {
	pool := testDB.Pool
	dbPool := db.NewDBPool(pool)
	t.Run("returns error for negative profileID", func(t *testing.T) {
		
	})
}
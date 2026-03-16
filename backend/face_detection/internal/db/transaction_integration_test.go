//go:build integration

package db_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v4"
	"github.com/stretchr/testify/assert"
	"mosaic-face-detection.com/internal/db"
	"mosaic-face-detection.com/internal/test"
)

func TestWithTransaction(t *testing.T) {
	pool := testDB.Pool

	t.Run("rolls back all operations when one fails mid-transaction", func(t *testing.T) {
		test.CleanupTables(t, pool)

		ctx := context.Background()
		var capturedProfileID int32

		err := db.WithTransaction(ctx, pool, func(tx pgx.Tx) error {
			err := tx.QueryRow(ctx, `INSERT INTO profiles DEFAULT VALUES RETURNING id`).Scan(&capturedProfileID)
			if err != nil {
				return err
			}
			
			// failing db operation
			_, err = tx.Exec(ctx, `INSERT INTO nonexistent_table (col) VALUES ($1)`, capturedProfileID)
			return err
		})

		assert.Error(t, err)

		var count int
		countErr := pool.QueryRow(ctx, `SELECT COUNT(*) FROM profiles WHERE id = $1`, capturedProfileID).Scan(&count)
		assert.Nil(t, countErr)
		assert.Equal(t, 0, count, "profile row should have been rolled back")
	})

	t.Run("commits successfully when all operations succeed", func(t *testing.T) {
		test.CleanupTables(t, pool)

		ctx := context.Background()
		var profileID int32

		err := db.WithTransaction(ctx, pool, func(tx pgx.Tx) error {
			return tx.QueryRow(ctx, `INSERT INTO profiles DEFAULT VALUES RETURNING id`).Scan(&profileID)
		})

		assert.Nil(t, err)

		var count int
		countErr := pool.QueryRow(ctx, `SELECT COUNT(*) FROM profiles WHERE id = $1`, profileID).Scan(&count)
		assert.Nil(t, countErr)
		assert.Equal(t, 1, count, "should be commited")
	})
}

package db

import (
	"context"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

// Helper method to execute a sql function within a transaction
// handles commit and rollback on errors
func WithTransaction(ctx context.Context, pool *pgxpool.Pool, fn func(pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}

	// defer so rollback is always called when any one of the returns is called
	// It will rollback on errors and when commit succeeds.
	defer func() { _ = tx.Rollback(ctx) }()

	err = fn(tx)
	if err != nil {
		return err 
	}

	return tx.Commit(ctx)
}
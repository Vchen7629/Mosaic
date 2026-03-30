//go:build unit

package db_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgconn"
	"github.com/stretchr/testify/assert"
	"mosaic-conversation-briefing.com/internal/db"
)

// overriding default so tests dont take forever
var fastConfig = db.RetryConfig{
	MaxAttempts: 3,
	InitialWait: 1 * time.Millisecond,
	MaxWait:     4 * time.Millisecond,
}

func TestRetryWithBackoff(t *testing.T) {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(handler).With("service", "conversation_briefing")
	t.Run("does not retry on non-transient error", func(t *testing.T) {
		calls := 0
		permanent := errors.New("some perm error")

		err := db.RetryWithBackoff(logger, context.Background(), fastConfig, func() error {
			calls++
			return permanent
		})

		assert.Equal(t, permanent, err)
		assert.Equal(t, 1, calls, "should bail immediately without retrying")
	})

	t.Run("exhausts all attempts on persistent transient error", func(t *testing.T) {
		calls := 0
		transient := &pgconn.PgError{Code: "40P01"}

		err := db.RetryWithBackoff(logger, context.Background(), fastConfig, func() error {
			calls++
			return transient
		})

		assert.Equal(t, transient, err)
		assert.Equal(t, fastConfig.MaxAttempts, calls)
	})

	t.Run("returns context error when context is cancelled during backoff", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		calls := 0
		transient := &pgconn.PgError{Code: "08006"}

		// cancel immediately so first backoff sleep is interrupted
		cancel()

		err := db.RetryWithBackoff(logger, ctx, fastConfig, func() error {
			calls++
			return transient
		})

		assert.Equal(t, context.Canceled, err)
		assert.Equal(t, 1, calls, "should not retry after context is cancelled")
	})

	t.Run("returns context error when context deadline exceeded during backoff", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		// use long backoff so context deadline fires before retry
		slowConfig := db.RetryConfig{
			MaxAttempts: 5,
			InitialWait: 500 * time.Millisecond,
			MaxWait:     2 * time.Second,
		}
		transient := &pgconn.PgError{Code: "40001"}

		err := db.RetryWithBackoff(logger, ctx, slowConfig, func() error { return transient })

		assert.Equal(t, context.DeadlineExceeded, err)
	})

	t.Run("string-matched transient error retries correctly", func(t *testing.T) {
		calls := 0
		transient := errors.New("connection refused by server")

		err := db.RetryWithBackoff(logger, context.Background(), fastConfig, func() error {
			calls++
			return transient
		})

		assert.Equal(t, transient, err)
		assert.Equal(t, fastConfig.MaxAttempts, calls)
	})

	t.Run("MaxAttempts of 1 does not retry at all", func(t *testing.T) {
		singleAttempt := db.RetryConfig{
			MaxAttempts: 1,
			InitialWait: 1 * time.Millisecond,
			MaxWait:     4 * time.Millisecond,
		}
		calls := 0
		transient := &pgconn.PgError{Code: "40001"}

		err := db.RetryWithBackoff(logger, context.Background(), singleAttempt, func() error {
			calls++
			return transient
		})

		assert.Equal(t, transient, err)
		assert.Equal(t, 1, calls)
	})

	t.Run("succeeds on first attempt", func(t *testing.T) {
		calls := 0
		err := db.RetryWithBackoff(logger, context.Background(), fastConfig, func() error {
			calls++
			return nil
		})

		assert.Nil(t, err)
		assert.Equal(t, 1, calls)
	})

	t.Run("succeeds after transient error on second attempt", func(t *testing.T) {
		calls := 0
		transient := &pgconn.PgError{Code: "40001"}

		err := db.RetryWithBackoff(logger, context.Background(), fastConfig, func() error {
			calls++
			if calls < 2 {
				return transient
			}
			return nil
		})

		assert.Nil(t, err)
		assert.Equal(t, 2, calls)
	})
}

package db

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgconn"
)

type RetryConfig struct {
	MaxAttempts int
	InitialWait time.Duration
	MaxWait     time.Duration
}

// PostgreSQL transient error codes for lookup
var transientErrorCodes = map[string]bool{
	"40001": true, // Serialization failure
	"40P01": true, // Deadlock detected
	"08006": true, // Connection failure
	"08003": true, // Connection does not exist
	"08000": true, // Connection exception
}

var transientErrorPatterns = map[string]bool{
	"connection refused":            true,
	"connection reset":              true,
	"i/o timeout":                   true,
	"no more connections available": true,
	"pool exhausted":                true,
	"connection closed":             true,
	"broken pipe":                   true,
}

// retries a function with exponential backoff for transient errors
func RetryWithBackoff(ctx context.Context, config RetryConfig, fn func() error) error {
	var lastErr error
	wait := config.InitialWait

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// fail immediately if not transient error
		if !isTransientError(err) {
			return err
		}

		// prevent retrying after last attempt
		if attempt == config.MaxAttempts {
			break
		}

		log.Printf(
			"Transient error on attempt %d/%d: %v. Retrying in %v",
			attempt, config.MaxAttempts, err, wait,
		)

		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return ctx.Err()
		}

		wait = wait * 2
		if wait > config.MaxWait {
			wait = config.MaxWait
		}
	}

	return lastErr
}

// Sets and returns sensible defaults for transient
// db error retries
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts: 3,
		InitialWait: 100 * time.Millisecond,
		MaxWait:     2 * time.Second,
	}
}

// checks if an error is a transient db error
func isTransientError(err error) bool {
	if err == nil {
		return false
	}

	// connection pool errors
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	// PostgreSQL specific errors
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return transientErrorCodes[pgErr.Code]
	}

	errMsg := strings.ToLower(err.Error())
	for pattern := range transientErrorPatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}

	return false
}

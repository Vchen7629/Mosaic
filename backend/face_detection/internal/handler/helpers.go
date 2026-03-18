package handler

import (
	"context"

	"mosaic-face-detection.com/internal/db"
)

func retryDB[T any](ctx context.Context, fn func() (T, error)) (T, error) {
	var result T
	err := db.RetryWithBackoff(ctx, db.DefaultRetryConfig(), func() error {
		var err error
		result, err = fn()
		return err
	})

	return result, err
}
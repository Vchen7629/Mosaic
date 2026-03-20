package handler

import (
	"context"
	"log/slog"

	"mosaic-face-detection.com/internal/db"
)

func retryDB[T any](logger *slog.Logger, ctx context.Context, fn func() (T, error)) (T, error) {
	var result T
	err := db.RetryWithBackoff(logger, ctx, db.DefaultRetryConfig(), func() error {
		var err error
		result, err = fn()
		return err
	})

	return result, err
}

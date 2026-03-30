//go:build unit

package observability_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"mosaic-conversation-briefing.com/internal/observability"
)

func TestStructuredLogger(t *testing.T) {

	t.Run("prod mode set to false should enable debug level", func(t *testing.T) {
		logger := observability.StructuredLogger(false)

		assert.True(t, logger.Enabled(context.Background(), slog.LevelDebug))
	})

	t.Run("prod mode set to true should disable debug level", func(t *testing.T) {
		logger := observability.StructuredLogger(true)

		assert.False(t, logger.Enabled(context.Background(), slog.LevelDebug))
		assert.True(t, logger.Enabled(context.Background(), slog.LevelInfo))
	})
}

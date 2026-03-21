//go:build unit

package handler_test

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"mosaic-conversation-briefing.com/internal/db"
	"mosaic-conversation-briefing.com/internal/handler"
)

func newTestLogger() *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})
	return slog.New(handler).With("service", "conversation_briefing")
}

// unit tests for GenerateBriefings function
func TestGenerateBriefings(t *testing.T) {
	t.Run("returns error when conversation list is empty", func(t *testing.T) {
		briefings, err := handler.GenerateBriefings("model", newTestLogger(), "http://localhost", []db.Conversations{})

		assert.Error(t, err, "empty conversation list should return an error")
		assert.Empty(t, briefings, "briefings should be empty when conversation list is empty")
	})

	t.Run("returns error when BuildPrompt fails due to empty convo list", func(t *testing.T) {
		convos := []db.Conversations{
			{ProfileID: 1, VisitorID: 10, ConvoList: []string{}},
		}

		briefings, err := handler.GenerateBriefings("model", newTestLogger(), "http://localhost", convos)

		assert.Error(t, err, "empty ConvoList should cause BuildPrompt to fail")
		assert.Nil(t, briefings)
	})
}

//go:build unit

package handler_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	cb "mosaic-conversation-briefing.com/gen"
	"mosaic-conversation-briefing.com/internal/handler"
)

func newTestServer() *handler.ConvoBriefingServer {
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})
	return handler.NewConvoBriefingServer(slog.New(h), nil, nil, "")
}

// Only the empty session token check fires before touching cache/db,
// so it's the only input validation testable without real dependencies.
func TestEmptySessionToken(t *testing.T) {
	s := newTestServer()

	resp, err := s.GenerateConversationBriefing(context.Background(), &cb.GenerateConversationBriefingRequest{
		SessionToken: "",
		VisitorIds:   []int32{1},
	})

	assert.Error(t, err)
	assert.False(t, resp.Success)
}

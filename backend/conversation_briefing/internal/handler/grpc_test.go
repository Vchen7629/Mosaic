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
	handlers := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})
	return handler.NewConvoBriefingServer(slog.New(handlers), nil, "")
}

func TestGenerateConversationBriefing(t *testing.T) {
	t.Run("returns error and success=false when profile ID is zero", func(t *testing.T) {
		s := newTestServer()

		resp, err := s.GenerateConversationBriefing(context.Background(), &cb.GenerateConversationBriefingRequest{
			ProfileId:  0,
			VisitorIds: []int32{1},
		})

		assert.Error(t, err, "zero profile ID should return an error")
		assert.False(t, resp.Success, "success should be false for invalid profile ID")
	})

	t.Run("returns error and success=false when profile ID is negative", func(t *testing.T) {
		s := newTestServer()

		resp, err := s.GenerateConversationBriefing(context.Background(), &cb.GenerateConversationBriefingRequest{
			ProfileId:  -1,
			VisitorIds: []int32{1},
		})

		assert.Error(t, err, "negative profile ID should return an error")
		assert.False(t, resp.Success, "success should be false for invalid profile ID")
	})

	t.Run("returns error and success=false when a visitor ID is zero", func(t *testing.T) {
		s := newTestServer()

		resp, err := s.GenerateConversationBriefing(context.Background(), &cb.GenerateConversationBriefingRequest{
			ProfileId:  1,
			VisitorIds: []int32{0},
		})

		assert.Error(t, err, "zero visitor ID should return an error")
		assert.False(t, resp.Success, "success should be false for invalid visitor ID")
	})

	t.Run("returns error and success=false when a visitor ID is negative", func(t *testing.T) {
		s := newTestServer()

		resp, err := s.GenerateConversationBriefing(context.Background(), &cb.GenerateConversationBriefingRequest{
			ProfileId:  1,
			VisitorIds: []int32{-5},
		})

		assert.Error(t, err, "negative visitor ID should return an error")
		assert.False(t, resp.Success, "success should be false for invalid visitor ID")
	})

	t.Run("validates all visitor IDs and returns error on first invalid one", func(t *testing.T) {
		s := newTestServer()

		resp, err := s.GenerateConversationBriefing(context.Background(), &cb.GenerateConversationBriefingRequest{
			ProfileId:  1,
			VisitorIds: []int32{10, 20, -1, 30},
		})

		assert.Error(t, err, "any invalid visitor ID should return an error")
		assert.False(t, resp.Success)
	})
}

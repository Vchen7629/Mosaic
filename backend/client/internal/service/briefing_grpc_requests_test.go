//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	cb "mosaic-client.com/gen/conversation_briefing"
	"mosaic-client.com/internal/test"
)

// unit tests for GenerateConversationBriefing function
func TestGenerateConversationBriefing(t *testing.T) {
	t.Run("Returns nil on successful response", func(t *testing.T) {
		client := &test.MockBriefingClient{}

		err := GenerateConversationBriefing("1", []int32{10, 20}, client)

		assert.NoError(t, err, "successful response should return nil")
		assert.Equal(t, 1, client.GenerateCalls, "GenerateConversationBriefing should be called once")
	})

	t.Run("Returns error on gRPC client error", func(t *testing.T) {
		client := &test.MockBriefingClient{
			GenerateFunc: func(_ *cb.GenerateConversationBriefingRequest) (*cb.GenerateConversationBriefingResponse, error) {
				return nil, status.Error(codes.Internal, "internal error")
			},
		}

		err := GenerateConversationBriefing("1", []int32{10}, client)

		assert.ErrorContains(t, err, "GenerateConversationBriefing gRPC error", "gRPC error should be propagated")
	})

	t.Run("Returns error when resp.Success is false", func(t *testing.T) {
		client := &test.MockBriefingClient{
			GenerateFunc: func(_ *cb.GenerateConversationBriefingRequest) (*cb.GenerateConversationBriefingResponse, error) {
				return &cb.GenerateConversationBriefingResponse{Success: false}, nil
			},
		}

		err := GenerateConversationBriefing("1", []int32{10}, client)

		assert.ErrorContains(t, err, "GenerateConversationBriefing processing error", "success=false should return an error")
	})

	t.Run("Sends correct profile ID and visitor IDs in request", func(t *testing.T) {
		var captured *cb.GenerateConversationBriefingRequest
		client := &test.MockBriefingClient{
			GenerateFunc: func(req *cb.GenerateConversationBriefingRequest) (*cb.GenerateConversationBriefingResponse, error) {
				captured = req
				return &cb.GenerateConversationBriefingResponse{Success: true}, nil
			},
		}

		err := GenerateConversationBriefing("7", []int32{10, 20, 30}, client)

		assert.NoError(t, err)
		assert.Equal(t, int32(7), captured.ProfileId, "profile ID should be forwarded to gRPC request")
		assert.Equal(t, []int32{10, 20, 30}, captured.VisitorIds, "visitor IDs should be forwarded to gRPC request")
	})

	t.Run("Returns nil with empty visitor ID list", func(t *testing.T) {
		client := &test.MockBriefingClient{}

		err := GenerateConversationBriefing("1", []int32{}, client)

		assert.NoError(t, err, "empty visitor list should still succeed")
	})

	t.Run("Returns error on unavailable gRPC service", func(t *testing.T) {
		client := &test.MockBriefingClient{
			GenerateFunc: func(_ *cb.GenerateConversationBriefingRequest) (*cb.GenerateConversationBriefingResponse, error) {
				return nil, status.Error(codes.Unavailable, "service unavailable")
			},
		}

		err := GenerateConversationBriefing("1", []int32{10}, client)

		assert.ErrorContains(t, err, "GenerateConversationBriefing gRPC error", "unavailable error should be propagated")
	})
}

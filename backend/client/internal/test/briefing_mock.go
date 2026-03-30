package test

import (
	"context"
	"sync"

	"google.golang.org/grpc"

	cb "mosaic-client.com/gen/conversation_briefing"
)

// MockBriefingClient is a configurable test double for cb.ConversationBriefingServiceClient.
// Set GenerateFunc to override the default success response.
type MockBriefingClient struct {
	Mu              sync.Mutex
	GenerateCalls   int

	GenerateFunc func(*cb.GenerateConversationBriefingRequest) (*cb.GenerateConversationBriefingResponse, error)
}

func (m *MockBriefingClient) GenerateConversationBriefing(
	_ context.Context,
	req *cb.GenerateConversationBriefingRequest,
	_ ...grpc.CallOption,
) (*cb.GenerateConversationBriefingResponse, error) {
	m.Mu.Lock()
	m.GenerateCalls++
	fn := m.GenerateFunc
	m.Mu.Unlock()

	if fn != nil {
		return fn(req)
	}
	return &cb.GenerateConversationBriefingResponse{Success: true}, nil
}

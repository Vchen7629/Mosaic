package handler

import (
	"context"
	"errors"

	cb "mosaic-conversation-briefing.com/gen"
	"mosaic-conversation-briefing.com/internal/db"
)

type ConvoBriefingServer struct {
	cb.UnimplementedConversationBriefingServiceServer
	llmBaseURL string
	pool       *db.DBPool
}

func NewConvoBriefingServer(
	dbPool *db.DBPool, llmBaseUrl string,
) *ConvoBriefingServer {
	return &ConvoBriefingServer{
		pool:       dbPool,
		llmBaseURL: llmBaseUrl,
	}
}

// Handler to generate conversation briefings
func (s *ConvoBriefingServer) GenerateConversationBriefing(
	ctx context.Context,
	req *cb.GenerateConversationBriefingRequest,
) (*cb.GenerateConversationBriefingResponse, error) {
	if req.ProfileId <= 0 {
		return &cb.GenerateConversationBriefingResponse{
			Success: false,
		}, errors.New("Invalid profile id in the req, less than or equal to 0")
	}
	for visitorID := range req.VisitorIds {
		if visitorID <= 0 {
			return &cb.GenerateConversationBriefingResponse{
				Success: false,
			}, errors.New("Invalid visitor id in the req, less than or equal to 0")
		}
	}

	var conversationList []db.Conversations
	err := db.RetryWithBackoff(ctx, db.DefaultRetryConfig(), func() error {
		var err error
		conversationList, err = s.pool.FetchRecentConversations(req.ProfileId, req.VisitorIds)
		return err
	})
	if err != nil {
		return &cb.GenerateConversationBriefingResponse{
			Success: false,
		}, err
	}

	briefings, err := SummarizeWithLLM("qwen2.5:3b", s.llmBaseURL, conversationList)
	if err != nil {
		return &cb.GenerateConversationBriefingResponse{
			Success: false,
		}, err
	}

	briefingMap := make(map[int32]string, len(briefings))
	for _, briefing := range briefings {
		briefingMap[briefing.VisitorID] = briefing.Briefing
	}

	err = db.RetryWithBackoff(ctx, db.DefaultRetryConfig(), func() error {
		err = s.pool.InsertBriefing(req.ProfileId, briefingMap)
		return err
	})
	if err != nil {
		return &cb.GenerateConversationBriefingResponse{
			Success: false,
		}, err
	}

	return &cb.GenerateConversationBriefingResponse{
		Success: true,
	}, nil
}

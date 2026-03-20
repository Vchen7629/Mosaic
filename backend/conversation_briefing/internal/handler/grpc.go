package handler

import (
	"context"
	"errors"
	"log/slog"

	cb "mosaic-conversation-briefing.com/gen"
	"mosaic-conversation-briefing.com/internal/db"
)

type ConvoBriefingServer struct {
	cb.UnimplementedConversationBriefingServiceServer
	logger *slog.Logger
	llmBaseURL string
	pool       *db.DBPool
}

func NewConvoBriefingServer(
	logger *slog.Logger, dbPool *db.DBPool, llmBaseUrl string,
) *ConvoBriefingServer {
	return &ConvoBriefingServer{
		logger: 	logger,
		pool:       dbPool,
		llmBaseURL: llmBaseUrl,
	}
}

// Handler to generate conversation briefings
func (s *ConvoBriefingServer) GenerateConversationBriefing(
	ctx context.Context,
	req *cb.GenerateConversationBriefingRequest,
) (*cb.GenerateConversationBriefingResponse, error) {
	s.logger.Debug("GenerateConversationBriefing called", "profile_id", req.ProfileId, "visitor_ids", req.VisitorIds)
	if req.ProfileId <= 0 {
		s.logger.Error("Invalid profile id in the req, less than or equal to 0", )
		return &cb.GenerateConversationBriefingResponse{
			Success: false,
		}, errors.New("Invalid profile id in the req, less than or equal to 0")
	}

	for _, visitorID := range req.VisitorIds {
		if visitorID <= 0 {
			s.logger.Error("Invalid visitor id in the req, less than or equal to 0", )
			return &cb.GenerateConversationBriefingResponse{
				Success: false,
			}, errors.New("Invalid visitor id in the req, less than or equal to 0")
		}
	}

	var conversationList []db.Conversations
	err := db.RetryWithBackoff(s.logger, ctx, db.DefaultRetryConfig(), func() error {
		var err error
		conversationList, err = s.pool.FetchRecentConversations(req.ProfileId, req.VisitorIds)
		return err
	})
	if err != nil {
		s.logger.Error("Error fetching recent conversations", "err", err)

		return &cb.GenerateConversationBriefingResponse{
			Success: false,
		}, err
	}

	briefings, err := GenerateBriefings("qwen2.5:3b", s.logger, s.llmBaseURL, conversationList)
	if err != nil {
		s.logger.Error("Error generating briefings", "err", err)

		return &cb.GenerateConversationBriefingResponse{
			Success: false,
		}, err
	}

	briefingMap := make(map[int32]string, len(briefings))
	for _, briefing := range briefings {
		briefingMap[briefing.VisitorID] = briefing.Briefing
	}

	err = db.RetryWithBackoff(s.logger, ctx, db.DefaultRetryConfig(), func() error {
		err = s.pool.InsertBriefing(req.ProfileId, briefingMap)
		return err
	})
	if err != nil {
		s.logger.Error("error inserting briefing", "err", err)
		return &cb.GenerateConversationBriefingResponse{
			Success: false,
		}, err
	}

	s.logger.Info(
		"Saved Conversation briefing", 
		"profile_id", req.ProfileId, 
		"visitor_id", req.VisitorIds,
	)
	return &cb.GenerateConversationBriefingResponse{
		Success: true,
	}, nil
}

package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/valkey-io/valkey-go"
	cb "mosaic-conversation-briefing.com/gen"
	"mosaic-conversation-briefing.com/internal/cache"
	"mosaic-conversation-briefing.com/internal/db"
	"mosaic-conversation-briefing.com/internal/observability"
)

type ConvoBriefingServer struct {
	cb.UnimplementedConversationBriefingServiceServer
	logger     *slog.Logger
	cache      valkey.Client
	llmBaseURL string
	pool       *db.DBPool
}

func NewConvoBriefingServer(
	logger *slog.Logger, cacheClient valkey.Client, dbPool *db.DBPool, llmBaseUrl string,
) *ConvoBriefingServer {
	return &ConvoBriefingServer{
		logger:     logger,
		cache:      cacheClient,
		pool:       dbPool,
		llmBaseURL: llmBaseUrl,
	}
}

// Handler to generate conversation briefings
func (s *ConvoBriefingServer) GenerateConversationBriefing(
	ctx context.Context,
	req *cb.GenerateConversationBriefingRequest,
) (*cb.GenerateConversationBriefingResponse, error) {
	s.logger.Debug("GenerateConversationBriefing called", "session_token", req.SessionToken, "visitor_ids", req.VisitorIds)
	if req.SessionToken == "" {
		s.logger.Error("invalid empty session token in the req")
		return &cb.GenerateConversationBriefingResponse{
			Success: false,
		}, errors.New("invalid empty session token in the req")
	}

	profileID, err := cache.FetchProfileIDFromCache(ctx, s.cache, req.SessionToken)
	if err != nil {
		var profileIDPtr *int32
		dbErr := db.RetryWithBackoff(s.logger, ctx, db.DefaultRetryConfig(), func() error {
			var err error
			profileIDPtr, err = s.pool.FetchProfileIDWithSession(req.SessionToken)
			return err
		})
		if dbErr != nil {
			observability.ErrorsTotal.WithLabelValues("fetch_profile_id_with_session").Inc()
			s.logger.Error("session token not found", "session_token", req.SessionToken)
			return &cb.GenerateConversationBriefingResponse{Success: false}, fmt.Errorf("invalid session token")
		}
		profileID = profileIDPtr
	}

	for _, visitorID := range req.VisitorIds {
		if visitorID <= 0 {
			s.logger.Error("invalid visitor id in the req, less than or equal to 0")
			return &cb.GenerateConversationBriefingResponse{
				Success: false,
			}, errors.New("invalid visitor id in the req, less than or equal to 0")
		}
	}

	fetchConvoStart := time.Now()
	var conversationList []db.Conversations
	err = db.RetryWithBackoff(s.logger, ctx, db.DefaultRetryConfig(), func() error {
		var err error
		conversationList, err = s.pool.FetchRecentConversations(*profileID, req.VisitorIds)
		return err
	})
	observability.ConvoFetchDuration.Observe(float64(time.Since(fetchConvoStart).Milliseconds()))
	if err != nil {
		observability.ErrorsTotal.WithLabelValues("fetch_conversation").Inc()
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

	insertBriefingStart := time.Now()
	err = db.RetryWithBackoff(s.logger, ctx, db.DefaultRetryConfig(), func() error {
		err = s.pool.InsertBriefing(*profileID, briefingMap)
		return err
	})
	observability.BriefingInsertDuration.Observe(float64(time.Since(insertBriefingStart).Milliseconds()))
	if err != nil {
		observability.ErrorsTotal.WithLabelValues("insert_briefing").Inc()
		s.logger.Error("error inserting briefing", "err", err)
		return &cb.GenerateConversationBriefingResponse{
			Success: false,
		}, err
	}

	s.logger.Info(
		"Saved Conversation briefing",
		"profile_id", profileID,
		"visitor_id", req.VisitorIds,
	)
	return &cb.GenerateConversationBriefingResponse{
		Success: true,
	}, nil
}

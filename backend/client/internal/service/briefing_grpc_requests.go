package service

import (
	"context"
	"errors"
	"fmt"

	cb "mosaic-client.com/gen/conversation_briefing"
)

// call the generate convo briefing gRPC server to process
func GenerateConversationBriefing(
	sessionToken string,
	visitorIDs []int32,
	client cb.ConversationBriefingServiceClient,
) error {
	ctx := context.Background()

	resp, err := client.GenerateConversationBriefing(ctx, &cb.GenerateConversationBriefingRequest{
		SessionToken: sessionToken,
		VisitorIds:   visitorIDs,
	})
	if err != nil {
		return fmt.Errorf("generateConversationBriefing gRPC error: %w", err)
	}

	if !resp.Success {
		return errors.New("generateConversationBriefing processing error")
	}

	return nil
}

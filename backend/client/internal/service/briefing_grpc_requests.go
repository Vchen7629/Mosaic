package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	cb "mosaic-client.com/gen/conversation_briefing"
)

// call the generate convo briefing gRPC server to process
func GenerateConversationBriefing(
	profileID string,
	visitorIDs []int32,
	client cb.ConversationBriefingServiceClient,
) error {
	id64, err := strconv.ParseInt(profileID, 10, 32)
	ctx := context.Background()

	if err != nil {
		return fmt.Errorf("error converting string to int64: %w", err)
	}
	resp, err := client.GenerateConversationBriefing(ctx, &cb.GenerateConversationBriefingRequest{
		ProfileId:  int32(id64),
		VisitorIds: visitorIDs,
	})
	if err != nil {
		return fmt.Errorf("generateConversationBriefing gRPC error: %w", err)
	}

	if !resp.Success {
		return errors.New("GenerateConversationBriefing processing error")
	}

	return nil
}

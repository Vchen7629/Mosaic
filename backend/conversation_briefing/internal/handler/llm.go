package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"mosaic-conversation-briefing.com/internal/db"
	"mosaic-conversation-briefing.com/internal/service"
)

type briefing struct {
	VisitorID int32
	Briefing  string
}

type ollamaChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"context"`
}

type ollamaChatRequest struct {
	Model    string              `json:"model"`
	Messages []ollamaChatMessage `json:"messages"`
	Stream   bool                `json:"stream"`
}

type ollamaChatResponse struct {
	Message ollamaChatMessage `json:"message"`
}

// handler to send conversations to llm to be summarized
func SummarizeWithLLM(
	model string,
	llmBaseURL string,
	conversationList []db.Conversations,
) ([]briefing, error) {
	if len(conversationList) == 0 {
		return []briefing{}, errors.New("conversationList is empty")
	}

	briefings := make([]briefing, 0, len(conversationList))
	for _, conv := range conversationList {
		prompt, err := service.BuildPrompt(conv.ConvoList)
		if err != nil {
			return nil, err
		}

		reqBody := ollamaChatRequest{
			Model: model,
			Messages: []ollamaChatMessage{
				{
					Role:    "system",
					Content: "You are a helpful assistant that summarizes user visitor conversations into concise briefings so they can view it later to see what they talked about",
				},
				{Role: "user", Content: *prompt},
			},
			Stream: false,
		}

		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request for visitor %d: %w", conv.VisitorID, err)
		}

		resp, err := http.Post(llmBaseURL+"/api/chat", "application/json", bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, fmt.Errorf("failed to call LLM for visitor %d:%w", conv.VisitorID, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("LLM returned status %d for visitor %d", resp.StatusCode, conv.VisitorID)
		}

		var ollamaResp ollamaChatResponse
		err = json.NewDecoder(resp.Body).Decode(&ollamaResp)
		if err != nil {
			return nil, fmt.Errorf("failed to decode LLM response for visitor %d: %w", conv.VisitorID, err)
		}

		briefings = append(briefings, briefing{
			VisitorID: conv.VisitorID,
			Briefing:  ollamaResp.Message.Content,
		})
	}

	return briefings, nil
}

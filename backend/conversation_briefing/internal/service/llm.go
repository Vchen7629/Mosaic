package service

import (
	"errors"
	"fmt"
	"strings"
)

// formats the conversation list into a summariation prompt
func BuildPrompt(convos []string) (*string, error) {
	if len(convos) == 0 {
		return nil, errors.New("missing conversation strings")
	}
	var sb strings.Builder

	sb.WriteString(
		"Please summarize the follow visitor conversations into a concise briefing:\n\n",
	)
	for i, convo := range convos {
		sb.WriteString(fmt.Sprintf("Conversation %d:\n%s\n\n", i + 1, convo))
	}

	sb.WriteString("Provide a 3-4 sentence summary of the key topics, and conversation starters")
	
	result := sb.String()
	return &result, nil
}
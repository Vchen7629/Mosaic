//go:build unit

package service_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"mosaic-conversation-briefing.com/internal/service"
)

func TestBuildPrompt(t *testing.T) {

	t.Run("missing convo text returns an error", func(t *testing.T) {
		prompt, err := service.BuildPrompt([]string{})

		assert.Nil(t, prompt)
		assert.Equal(t, "missing conversation strings", err.Error())
	})

	t.Run("it builds prompt with the conversation text", func(t *testing.T) {
		prompt, err := service.BuildPrompt([]string{"test prompt"})

		assert.Equal(t, "Please summarize the follow visitor conversations into a concise briefing:\n\nConversation 1:\ntest prompt\n\nProvide a 3-4 sentence summary of the key topics, and conversation starters", *prompt)
		assert.Nil(t, err)
	})

	t.Run("it builds prompt with multiple conversations", func(t *testing.T) {
		prompt, err := service.BuildPrompt([]string{"convo 1", "convo 2"})

		assert.Nil(t, err)
		assert.Equal(t, "Please summarize the follow visitor conversations into a concise briefing:\n\nConversation 1:\nconvo 1\n\nConversation 2:\nconvo 2\n\nProvide a 3-4 sentence summary of the key topics, and conversation starters", *prompt)
	})
}

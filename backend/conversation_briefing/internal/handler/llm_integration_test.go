//go:build integration

package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"mosaic-conversation-briefing.com/internal/db"
	"mosaic-conversation-briefing.com/internal/observability"
)

func newTestLogger() *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})
	return slog.New(handler).With("service", "conversation_briefing")
}

// startLLMServer spins up a test HTTP server that returns the given ollama
// response body for every request.
func startLLMServer(t *testing.T, statusCode int, body ollamaChatResponse) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(body)
	}))
}

// unit tests for GenerateBriefings function
func TestGenerateBriefings(t *testing.T) {
	t.Run("returns error and increments metric when HTTP call to LLM fails", func(t *testing.T) {
		before := testutil.ToFloat64(observability.ErrorsTotal.WithLabelValues("llm_response"))
		convos := []db.Conversations{
			{ProfileID: 1, VisitorID: 10, ConvoList: []string{"hello world"}},
		}

		briefings, err := GenerateBriefings("model", newTestLogger(), "http://127.0.0.1:0", convos)

		assert.Error(t, err, "failed HTTP call should return an error")
		assert.Nil(t, briefings)
		assert.Equal(t, before+1, testutil.ToFloat64(observability.ErrorsTotal.WithLabelValues("llm_response")),
			"llm_response error counter should be incremented")
	})

	t.Run("returns error when LLM returns non-200 status", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()

		convos := []db.Conversations{
			{ProfileID: 1, VisitorID: 10, ConvoList: []string{"hello world"}},
		}

		briefings, err := GenerateBriefings("model", newTestLogger(), srv.URL, convos)

		assert.Error(t, err, "non-200 status should return an error")
		assert.Nil(t, briefings)
	})

	t.Run("returns error when LLM response body is invalid JSON", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("not json"))
		}))
		defer srv.Close()

		convos := []db.Conversations{
			{ProfileID: 1, VisitorID: 10, ConvoList: []string{"hello world"}},
		}

		briefings, err := GenerateBriefings("model", newTestLogger(), srv.URL, convos)

		assert.Error(t, err, "invalid JSON response should return an error")
		assert.Nil(t, briefings)
	})

	t.Run("sends request to /api/chat endpoint", func(t *testing.T) {
		var capturedPath string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedPath = r.URL.Path
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(ollamaChatResponse{
				Message: ollamaChatMessage{Content: "briefing"},
			})
		}))
		defer srv.Close()

		convos := []db.Conversations{
			{ProfileID: 1, VisitorID: 10, ConvoList: []string{"some convo"}},
		}

		_, err := GenerateBriefings("model", newTestLogger(), srv.URL, convos)

		assert.NoError(t, err)
		assert.Equal(t, "/api/chat", capturedPath, "request should be sent to /api/chat")
	})

	t.Run("returns correct briefing content for a single conversation", func(t *testing.T) {
		srv := startLLMServer(t, http.StatusOK, ollamaChatResponse{
			Message: ollamaChatMessage{Role: "assistant", Content: "Alice is a frequent buyer"},
		})
		defer srv.Close()

		convos := []db.Conversations{
			{ProfileID: 1, VisitorID: 42, ConvoList: []string{"I want to buy something"}},
		}

		briefings, err := GenerateBriefings("model", newTestLogger(), srv.URL, convos)

		assert.NoError(t, err)
		assert.Len(t, briefings, 1, "should return one briefing")
		assert.Equal(t, int32(42), briefings[0].VisitorID, "visitor ID should match")
		assert.Equal(t, "Alice is a frequent buyer", briefings[0].Briefing, "briefing content should match LLM response")
	})

	t.Run("returns one briefing per conversation for multiple conversations", func(t *testing.T) {
		call := 0
		responses := []string{"briefing for visitor 1", "briefing for visitor 2"}
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(ollamaChatResponse{
				Message: ollamaChatMessage{Content: responses[call]},
			})
			call++
		}))
		defer srv.Close()

		convos := []db.Conversations{
			{ProfileID: 1, VisitorID: 10, ConvoList: []string{"convo with visitor 1"}},
			{ProfileID: 1, VisitorID: 20, ConvoList: []string{"convo with visitor 2"}},
		}

		briefings, err := GenerateBriefings("model", newTestLogger(), srv.URL, convos)

		assert.NoError(t, err)
		assert.Len(t, briefings, 2, "should return one briefing per conversation")
	})

}

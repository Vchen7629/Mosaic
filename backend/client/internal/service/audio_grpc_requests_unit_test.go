//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	at "mosaic-client.com/gen/audio_transcription"
	"mosaic-client.com/internal/test"
)

// patchSleep replaces the package-level sleepFn with a no-op for the duration
// of the test, then restores it via t.Cleanup.
func patchSleep(t *testing.T) {
	t.Helper()
	original := SleepFn
	SleepFn = func(time.Duration) {}
	t.Cleanup(func() { SleepFn = original })
}

func TestSaveTranscriptWithRetry(t *testing.T) {
	t.Run("Returns nil and calls SaveTranscript once on first attempt success", func(t *testing.T) {
		client := &test.MockAudioClient{}

		err := SaveTranscriptWithRetry(context.Background(), "1", []string{"10", "20"}, client)

		assert.NoError(t, err, "successful first attempt should return nil")
		assert.Equal(t, 1, client.SaveCalls, "SaveTranscript should be called exactly once")
	})

	t.Run("Sends correct session token and visitor IDs in request", func(t *testing.T) {
		var captured *at.SaveTranscriptRequest
		client := &test.MockAudioClient{
			SaveFunc: func(req *at.SaveTranscriptRequest) (*at.SaveTranscriptResponse, error) {
				captured = req
				return &at.SaveTranscriptResponse{Success: true}, nil
			},
		}

		err := SaveTranscriptWithRetry(context.Background(), "7", []string{"10", "20", "30"}, client)

		assert.NoError(t, err)
		assert.Equal(t, "7", captured.SessionToken, "session token should be forwarded to SaveTranscript")
		assert.Equal(t, []int32{10, 20, 30}, captured.VisitorIds, "visitor IDs should be forwarded to SaveTranscript")
	})

	t.Run("Retries and succeeds on second attempt", func(t *testing.T) {
		patchSleep(t)
		attempt := 0
		client := &test.MockAudioClient{
			SaveFunc: func(_ *at.SaveTranscriptRequest) (*at.SaveTranscriptResponse, error) {
				attempt++
				if attempt == 1 {
					return nil, status.Error(codes.Unavailable, "temporarily unavailable")
				}
				return &at.SaveTranscriptResponse{Success: true}, nil
			},
		}

		err := SaveTranscriptWithRetry(context.Background(), "1", []string{"10"}, client)

		assert.NoError(t, err, "should succeed on second attempt")
		assert.Equal(t, 2, client.SaveCalls, "SaveTranscript should be called twice")
	})

	t.Run("Returns error after all 3 attempts fail", func(t *testing.T) {
		patchSleep(t)
		client := &test.MockAudioClient{
			SaveFunc: func(_ *at.SaveTranscriptRequest) (*at.SaveTranscriptResponse, error) {
				return nil, status.Error(codes.Unavailable, "db down")
			},
		}

		err := SaveTranscriptWithRetry(context.Background(), "1", []string{"10"}, client)

		assert.ErrorContains(t, err, "3 attempts", "error should mention retry count")
		assert.Equal(t, 3, client.SaveCalls, "SaveTranscript should be called 3 times")
	})

	t.Run("Treats resp.Success false as a failure and retries", func(t *testing.T) {
		patchSleep(t)
		attempt := 0
		client := &test.MockAudioClient{
			SaveFunc: func(_ *at.SaveTranscriptRequest) (*at.SaveTranscriptResponse, error) {
				attempt++
				if attempt == 1 {
					return &at.SaveTranscriptResponse{Success: false}, nil
				}
				return &at.SaveTranscriptResponse{Success: true}, nil
			},
		}

		err := SaveTranscriptWithRetry(context.Background(), "1", []string{"10"}, client)

		assert.NoError(t, err, "should succeed on retry after success=false")
		assert.Equal(t, 2, client.SaveCalls, "SaveTranscript should be called twice")
	})
}

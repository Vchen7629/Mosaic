package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	at "mosaic-client.com/gen/audio_transcription"
)

var SleepFn = time.Sleep

// Method for sending gRPC request to save the transcript for the
// sessionToken to the database, handles retries with exp backoff
func SaveTranscriptWithRetry(
	ctx context.Context,
	sessionToken string,
	visitorIDs []string,
	client at.AudioTranscriptionServiceClient,
) error {
	visitorIDList := make([]int32, 0, len(visitorIDs))
	for _, visitorID := range visitorIDs {
		visitorID64, err := strconv.ParseInt(visitorID, 10, 32)
		if err != nil {
			return fmt.Errorf("error converting visitorID string to int64: %w", err)
		}
		visitorIDList = append(visitorIDList, int32(visitorID64))
	}

	var lastErr error
	for attempt := range 3 {
		resp, rpcErr := client.SaveTranscript(ctx, &at.SaveTranscriptRequest{
			SessionToken: sessionToken,
			VisitorIds:   visitorIDList,
		})
		if rpcErr == nil && resp.Success {
			return nil
		}
		if rpcErr != nil {
			lastErr = rpcErr
		} else {
			lastErr = fmt.Errorf("save transcript returned success=false")
		}

		wait := time.Duration(1<<attempt) * time.Second // 1s, 2s, 4s
		SleepFn(wait)
	}

	return fmt.Errorf("save transcript failed after 3 attempts: %w", lastErr)
}

// Helper function that sends the audio batch to whisper service for transcription
// handles retries with exponential backoff
func transcribeWithRetry(
	ctx context.Context,
	client at.AudioTranscriptionServiceClient,
	batch []float32,
	sessionToken string,
) error {
	var err error

	for attempt := range 3 {
		resp, rpcErr := client.TranscribeAudio(ctx, &at.TranscribeAudioRequest{
			AudioBytes:   batch,
			SessionToken: sessionToken,
		})
		if rpcErr == nil && resp.Success {
			return nil
		}
		if rpcErr != nil {
			code := status.Code(rpcErr)
			if code != codes.ResourceExhausted && code != codes.Unavailable {
				return fmt.Errorf("transcription failed (non-retryable): %w", rpcErr)
			}
			err = rpcErr
		} else {
			err = fmt.Errorf("transcription returned success=false")
		}
		wait := time.Duration(1<<attempt) * time.Second // 1s, 2s, 4s
		SleepFn(wait)
	}

	return fmt.Errorf("transcription failed after 3 attempts: %w", err)
}

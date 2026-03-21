package service

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	at "mosaic-client.com/gen/audio_transcription"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	audioBuffer []float32
	bufferMutex sync.Mutex
	Wg          sync.WaitGroup
)

const (
	silenceThreshold = 0.02 // lower bound, prevents silence
	loudThreshold    = 0.5 // upper bound, prevents loud noises
	sampleRate		 = 16000 // 16khz required for whisper
	batchDuration	 = 2.5 // seconds
	batchSize 		 = sampleRate * batchDuration
)

// Method for sending gRPC request to save the transcript for the
// profileID to the database, handles retries with exp backoff
func SaveTranscriptWithRetry(
	ctx context.Context,
	profileID string,
	visitorIDs []string,
	client at.AudioTranscriptionServiceClient,
) error {
	profileID64, err := strconv.ParseInt(profileID, 10, 32)
	if err != nil {
		return fmt.Errorf("Error converting profileID string to int64: %w", err)
	}
	visitorIDList := make([]int32, 0, len(visitorIDs))
	for _, visitorID := range visitorIDs {
		visitorID64, err := strconv.ParseInt(visitorID, 10, 32)
		if err != nil {
			return fmt.Errorf("Error converting profileID string to int64: %w", err)
		}
		visitorIDList = append(visitorIDList, int32(visitorID64))
	}
	
	var lastErr error
	for attempt := range 3 {
		resp, rpcErr := client.SaveTranscript(ctx, &at.SaveTranscriptRequest{ 
			ProfileId: int32(profileID64), 
			VisitorIds: visitorIDList,
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
		time.Sleep(wait)
	}

	return fmt.Errorf("save transcript failed after 3 attempts: %w", lastErr)
}

// Helper function that sends the audio batch to whisper service for transcription
// handles retries with exponential backoff
func transcribeWithRetry(
	ctx context.Context, 
	client at.AudioTranscriptionServiceClient,
	batch []float32,
	profileID int32,
) error {
	var err error

	for attempt := range 3 {
		resp, rpcErr := client.TranscribeAudio(ctx, &at.TranscribeAudioRequest{
			AudioBytes: batch,
			ProfileId: profileID,
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
		time.Sleep(wait)
	}

	return fmt.Errorf("transcription failed after 3 attempts: %w", err)
}

package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"sync"

	at "mosaic-client.com/gen/audio_transcription"
)

var (
	audioBuffer []float32
	bufferMutex sync.Mutex
	Wg          sync.WaitGroup
)

const (
	silenceThreshold = 0.02  // lower bound, prevents silence
	loudThreshold    = 0.5   // upper bound, prevents loud noises
	sampleRate       = 16000 // 16khz required for whisper
	batchDuration    = 5     // seconds
	batchSize        = sampleRate * batchDuration
)

func ProcessAudio(
	logger *slog.Logger,
	audioData string,
	sessionToken string,
	client at.AudioTranscriptionServiceClient,
) error {
	ctx := context.Background()

	// Decode base64
	audioBytes, err := base64.StdEncoding.DecodeString(audioData)
	if err != nil {
		return fmt.Errorf("audio decode error: %w", err)
	}

	float32Samples := bytesToFloat32(audioBytes)

	// RMS filtering to filter out loud background noise or silence
	if !isAudioValid(float32Samples) {
		return nil
	}

	bufferMutex.Lock()
	audioBuffer = append(audioBuffer, float32Samples...)
	currentLength := len(audioBuffer)
	bufferMutex.Unlock()

	if currentLength >= batchSize {
		bufferMutex.Lock()
		batch := audioBuffer[:batchSize]
		audioBuffer = audioBuffer[batchSize:]
		bufferMutex.Unlock()

		logger.Debug("[ProcessAudio] Sending batch", "batch_bytes", len(batch), "remaining_bytes", len(audioBuffer))

		err := transcribeWithRetry(ctx, client, batch, sessionToken)
		if err != nil {
			return fmt.Errorf("error processing audio: %v", err)
		}
	}

	return nil
}

// Method for flushing remaining audio bytes to gRPC service
// prevents audio sent before 10s batch from being lost when
// user clicks stop recording
func FlushAudio(
	logger *slog.Logger,
	ctx context.Context,
	sessionToken string,
	client at.AudioTranscriptionServiceClient,
) error {
	Wg.Wait()

	bufferMutex.Lock()
	remaining := audioBuffer
	audioBuffer = nil
	bufferMutex.Unlock()

	// handles case where there is audio less than 1 second
	if len(remaining) < sampleRate {
		return nil
	}

	err := transcribeWithRetry(ctx, client, remaining, sessionToken)
	if err != nil {
		return fmt.Errorf("error flushing audio: %v", err)
	}

	return nil
}

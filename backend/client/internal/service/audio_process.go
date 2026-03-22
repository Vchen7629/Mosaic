package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"strconv"

	at "mosaic-client.com/gen/audio_transcription"
)

func ProcessAudio(
	logger *slog.Logger,
	audioData string,
	profileID string,
	client at.AudioTranscriptionServiceClient,
) error {
	ctx := context.Background()
	id64, err := strconv.ParseInt(profileID, 10, 32)
	if err != nil {
		return fmt.Errorf("converting string to int error: %w", err)
	}

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

		err := transcribeWithRetry(ctx, client, batch, int32(id64))
		if err != nil {
			return fmt.Errorf("error processing audio: %v", err)
		}
	}

	// send to whisper for transcribing audio to text via gRPC
	return nil
}

// Method for flushing remaining audio bytes to gRPC service
// prevents audio sent before 10s batch from being lost when
// user clicks stop recording
func FlushAudio(
	logger *slog.Logger,
	ctx context.Context,
	profileID string,
	client at.AudioTranscriptionServiceClient,
) error {
	id64, err := strconv.ParseInt(profileID, 10, 32)
	if err != nil {
		return fmt.Errorf("error converting string to int: %w", err)
	}

	Wg.Wait()

	bufferMutex.Lock()
	remaining := audioBuffer
	audioBuffer = nil
	bufferMutex.Unlock()

	// handles case where there is audio less than 1 second
	if len(remaining) < sampleRate {
		return nil
	}

	err = transcribeWithRetry(ctx, client, remaining, int32(id64))
	if err != nil {
		return fmt.Errorf("error flushing audio: %v", err)
	}

	return nil
}

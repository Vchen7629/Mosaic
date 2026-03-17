package service

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"strconv"
	"sync"
	"time"

	at "mosaic-client.com/gen/audio_transcription"
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
	batchDuration	 = 10 // seconds
	batchSize 		 = sampleRate * batchDuration
)


func ProcessAudio(
	audioData string, 
	profileID string,
	client at.AudioTranscriptionServiceClient,
) error {
	ctx := context.Background()
	id64, err := strconv.ParseInt(profileID, 10, 32)

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

		
		log.Printf("[ProcessAudio] Sending batch: %d audio bytes, remaining: %d audio bytes", len(batch), len(audioBuffer))

		err := transcribeWithRetry(ctx, client, batch, int32(id64))
		if err != nil {
			log.Printf("Error transcribing: %v", err)
			return err
		}
	}

	// send to whisper for transcribing audio to text via gRPC
	return nil
}

// Method for flushing remaining audio bytes to gRPC service
// prevents audio sent before 10s batch from being lost when
// user clicks stop recording
func FlushAudio(
	ctx context.Context,
	profileID string, 
	client at.AudioTranscriptionServiceClient,
) {
	id64, err := strconv.ParseInt(profileID, 10, 32)

	Wg.Wait()

	bufferMutex.Lock()
	remaining := audioBuffer
	audioBuffer = nil
	bufferMutex.Unlock()

	// handles case where there is audio less than 1 second
	if len(remaining) < sampleRate {
		return
	}

	err = transcribeWithRetry(ctx, client, remaining, int32(id64))
	if err != nil {
		log.Printf("Error flushing audio: %v", err)
	}
}

// Method for sending gRPC request to save the transcript for the
// profileID to the database, handles retries with exp backoff
func SaveTranscriptWithRetry(
	ctx context.Context,
	profileID string,
	visitorID string,
	client at.AudioTranscriptionServiceClient,
) error {
	profileID64, err := strconv.ParseInt(profileID, 10, 32)
	if err != nil {
		return fmt.Errorf("Error converting profileID string to int64: %w", err)
	}
	visitorID64, err := strconv.ParseInt(visitorID, 10, 32)
	if err != nil {
		return fmt.Errorf("Error converting visitorID string to int64: %w", err)
	}
	
	var lastErr error
	for attempt := range 3 {
		resp, rpcErr := client.SaveTranscript(ctx, &at.SaveTranscriptRequest{ 
			ProfileId: int32(profileID64), 
			VisitorId: int32(visitorID64),
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
			err = rpcErr
		} else {
			err = fmt.Errorf("transcription returned success=false")
		}
		wait := time.Duration(1<<attempt) * time.Second // 1s, 2s, 4s
		time.Sleep(wait)
	}

	return fmt.Errorf("transcription failed after 3 attempts: %w", err)
}

func isAudioValid(audioBytes []float32) bool {
	rms := calculateRMS(audioBytes)

	if rms < silenceThreshold || rms > loudThreshold {
		return false
	}
	
	return true
}

// computes root mean square of audio samples
// assumes 16-bit PCM audio
func calculateRMS(audioBytes []float32) float64 {
	if len(audioBytes) < 2 {
		return 0.0
	}

	var sumSquares float64
	for _, sample := range audioBytes {
		sumSquares += float64(sample) * float64(sample)
	}
	
	return math.Sqrt(sumSquares / float64(len(audioBytes)))
}

func bytesToFloat32(data []byte) []float32 {
	samples := make([]float32, len(data)/4)
	for i := range samples {
			bits := binary.LittleEndian.Uint32(data[i*4:])
			samples[i] = math.Float32frombits(bits)
	}
	return samples
}

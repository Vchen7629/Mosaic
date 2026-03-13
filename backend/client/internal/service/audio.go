package service

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"sync"
)

var (
	audioBuffer []float32
	bufferMutex sync.Mutex
)

const (
	silenceThreshold = 0.02 // lower bound, prevents silence
	loudThreshold    = 0.5 // upper bound, prevents loud noises
	sampleRate		 = 16000 // 16khz required for whisper
	batchDuration	 = 10 // seconds
	batchSize 		 = sampleRate * batchDuration
)

func ProcessAudio(audioData string, patientID string) error {
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

		return nil
	}

	// send to whisper for transcribing audio to text via gRPC
	return nil
}

func isAudioValid(audioBytes []float32) bool {
	rms := calculateRMS(audioBytes)

	if rms < silenceThreshold {
		log.Printf("Silence (rms=%.4f), skipping", rms)
		return false
	}

	if rms > loudThreshold {
		log.Printf("Audio too loud (rms=%.4f), skipping", rms)
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

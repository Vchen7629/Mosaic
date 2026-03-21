package service

import (
	"encoding/binary"
	"math"
)

// checks if the audio is too loud or too silent and returns false
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

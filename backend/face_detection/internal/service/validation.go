package service

import (
	"errors"
	"math"

	"github.com/Kagami/go-face"
)

// validation for embeddings parameter
func ValidateEmbedding(embedding face.Descriptor) error {
	// Check if embedding is all zeros
	allZero := true
	for _, val := range embedding {
		// Check for NaN or infinity
		if math.IsNaN(float64(val)) || math.IsInf(float64(val), 0) {
			return errors.New("embedding contains NaN or infinity values")
		}
		if val != 0 {
			allZero = false
			break
		}
	}

	if allZero {
		return errors.New("embedding cannot be all zeros")
	}

	return nil
}

// Validates embedding slice received from protobuf
func ValidateEmbeddingSlice(embeddingSlice []float32) error {
	const embeddingDimension = 128

	if len(embeddingSlice) != embeddingDimension {
		return errors.New("face embedding must be 128 dimensions")
	}
	return nil
}

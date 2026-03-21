package service

import (
	"fmt"
	"strconv"
	"strings"
)

// Helper method to convert an embedding string to []float32
func ParseEmbedding(embString string) ([]float32, error) {
	embString = strings.TrimSpace(embString)
	embString = strings.TrimPrefix(embString, "[")
	embString = strings.TrimSuffix(embString, "]")

	parts := strings.Split(embString, ",")
	res := make([]float32, len(parts))
	for i, p := range parts {
		val, err := strconv.ParseFloat(strings.TrimSpace(p), 32)
		if err != nil {
			return nil, fmt.Errorf("Invalid embedding val at index %d: %w", i, err)
		}
		res[i] = float32(val)
	}
	return res, nil
}

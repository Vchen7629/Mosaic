//go:build unit

package service_test

import (
	"math"
	"testing"

	"github.com/Kagami/go-face"
	"github.com/stretchr/testify/assert"
	"mosaic-face-detection.com/internal/service"
	"mosaic-face-detection.com/internal/test"
)

func TestValidateEmbedding(t *testing.T) {

	t.Run("Errors if its Nan", func(t *testing.T) {
		rawEmbedding := test.MakeEmbedding(float32(math.NaN()), 128)
		var embedding face.Descriptor
		copy(embedding[:], rawEmbedding)

		err := service.ValidateEmbedding(embedding)
		assert.Error(t, err)
	})

	t.Run("Errors if its positive Infinity values", func(t *testing.T) {
		rawEmbedding := test.MakeEmbedding(float32(math.Inf(1)), 128)
		var embedding face.Descriptor
		copy(embedding[:], rawEmbedding)

		err := service.ValidateEmbedding(embedding)
		assert.Error(t, err)
	})

	t.Run("Errors if its negative Infinity values", func(t *testing.T) {
		rawEmbedding := test.MakeEmbedding(float32(math.Inf(-1)), 128)
		var embedding face.Descriptor
		copy(embedding[:], rawEmbedding)

		err := service.ValidateEmbedding(embedding)
		assert.Error(t, err)
	})
	
	t.Run("Errors if its Zeros", func(t *testing.T) {
		rawEmbedding := test.MakeEmbedding(0, 128)
		var embedding face.Descriptor
		copy(embedding[:], rawEmbedding)

		err := service.ValidateEmbedding(embedding)
		assert.Error(t, err)
	})

	t.Run("Doesnt error if non Zero", func(t *testing.T) {
		rawEmbedding := test.MakeEmbedding(0.5, 128)
		var embedding face.Descriptor
		copy(embedding[:], rawEmbedding)

		err := service.ValidateEmbedding(embedding)
		assert.Nil(t, err)
	})
}

func TestValidateEmbeddingSlice(t *testing.T) {

	t.Run("Errors if its not 128 dim", func(t *testing.T) {
		embedding := test.MakeEmbedding(0.2, 256)

		err := service.ValidateEmbeddingSlice(embedding)
		assert.Error(t, err)
	})

	t.Run("Doesnt error if its 128 dim", func(t *testing.T) {
		embedding := test.MakeEmbedding(0.2, 128)

		err := service.ValidateEmbeddingSlice(embedding)
		assert.Nil(t, err)
	})
}
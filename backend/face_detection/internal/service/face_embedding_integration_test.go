//go:build integration

package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"mosaic-face-detection.com/internal/service"
	"mosaic-face-detection.com/internal/test"
)

const testModelsDir = "../models"

func TestGenerateFaceEmbeddings(t *testing.T) {
	recPool, err := service.NewRecognizerPool(testModelsDir, 5)
	assert.NoError(t, err)

	t.Run("returns one embedding for single face image", func(t *testing.T) {
		rec, _ := recPool.Acquire(context.Background())
		defer recPool.Release(rec)

		embeddings, err := service.GenerateFaceEmbeddings(rec, test.LoadImage(t, "bona.jpg"))

		assert.NoError(t, err)
		assert.Len(t, embeddings, 1)
	})

	t.Run("returns multiple embeddings for group photo", func(t *testing.T) {
		rec, _ := recPool.Acquire(context.Background())
		defer recPool.Release(rec)

		embeddings, err := service.GenerateFaceEmbeddings(rec, test.LoadImage(t, "group.jpeg"))

		assert.NoError(t, err)
		assert.Greater(t, len(embeddings), 1)
	})

	t.Run("returns empty slice for image with no faces", func(t *testing.T) {
		rec, _ := recPool.Acquire(context.Background())
		defer recPool.Release(rec)

		embeddings, err := service.GenerateFaceEmbeddings(rec, test.LoadImage(t, "halfdome.jpg"))

		assert.NoError(t, err)
		assert.Empty(t, embeddings)
	})

	t.Run("returns error for invalid image bytes", func(t *testing.T) {
		rec, _ := recPool.Acquire(context.Background())
		defer recPool.Release(rec)

		invalidBytes := []byte("not an image")
		embeddings, err := service.GenerateFaceEmbeddings(rec, invalidBytes)

		assert.Error(t, err)
		assert.Nil(t, embeddings)
		assert.Contains(t, err.Error(), "recognition failed")
	})

	t.Run("returns error for empty bytes", func(t *testing.T) {
		rec, _ := recPool.Acquire(context.Background())
		defer recPool.Release(rec)

		embeddings, err := service.GenerateFaceEmbeddings(rec, []byte{})

		assert.Error(t, err)
		assert.Nil(t, embeddings)
	})

	t.Run("returns same embeddings for same image called twice", func(t *testing.T) {
		rec, _ := recPool.Acquire(context.Background())
		defer recPool.Release(rec)

		imgBytes := test.LoadImage(t, "bona.jpg")

		emb1, err1 := service.GenerateFaceEmbeddings(rec, imgBytes)
		emb2, err2 := service.GenerateFaceEmbeddings(rec, imgBytes)

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.Equal(t, emb1, emb2, "same image should produce identical embeddings")
	})
}

// debug test code
/*func TestFaceDistances(t *testing.T) {
	rec, err := service.InitializeFaceDetector(testModelsDir)
	assert.NoError(t, err)

	images := map[string]face.Descriptor{
		"bona":     getEmbedding(t, rec, "bona.jpg"),
		"bona2":    getEmbedding(t, rec, "bona2.jpg"),
		"bona3":    getEmbedding(t, rec, "bona3.jpg"),
		"man1": getEmbedding(t, rec, "man1.jpg"),
		"eunsoo":    getEmbedding(t, rec, "eunseo1.jpg"),
	}

	names := []string{"bona", "bona2", "bona3", "man1", "eunsoo"}
	t.Log("\nPairwise euclidean distances (threshold=0.6 means match):")
	for i, a := range names {
		for _, b := range names[i+1:] {
			t.Logf("  %s <-> %s: %.4f", a, b, euclideanDistance(images[a], images[b]))
		}
	}
}

func euclideanDistance(a, b face.Descriptor) float64 {
	var sum float64
	for i := range a {
		diff := float64(a[i] - b[i])
		sum += diff * diff
	}
	return sum
}*/

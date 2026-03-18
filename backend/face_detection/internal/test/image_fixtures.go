package test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Kagami/go-face"
	"github.com/stretchr/testify/assert"
	"mosaic-face-detection.com/internal/service"
)


func imagesDir() string {
	_, currentFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(currentFile), "images")
}

func LoadImage(t *testing.T, filename string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(imagesDir(), filename))
	assert.Nil(t, err)
	return data
}

// helper to load an image and extract first face embedding
func GetEmbedding(t *testing.T, rec *face.Recognizer, filename string) face.Descriptor {
	t.Helper()
	embeddings, err := service.GenerateFaceEmbeddings(rec, LoadImage(t, filename))
	assert.NoError(t, err)
	assert.Len(t, embeddings, 1)
	return embeddings[0]
}

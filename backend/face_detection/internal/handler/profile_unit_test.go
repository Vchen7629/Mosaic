//go:build unit

package handler_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	fd "mosaic-face-detection.com/gen"
	"mosaic-face-detection.com/internal/handler"
	"mosaic-face-detection.com/internal/test"
)

func newUnitServer() *handler.FaceDetectionServer {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	return handler.NewFaceDetectionServer(logger, nil, nil, nil)
}

func TestRegisterProfileFaceInvalidInput(t *testing.T) {
	server := newUnitServer()

	tests := []struct {
		name string
		req  *fd.RegisterProfileFaceRequest
	}{
		{
			"empty embeddings slice returns error",
			&fd.RegisterProfileFaceRequest{FaceEmbedding: []*fd.FaceEmbedding{}},
		},
		{
			"invalid embedding length returns error",
			&fd.RegisterProfileFaceRequest{
				FaceEmbedding: []*fd.FaceEmbedding{
					{FaceEmbedding: test.MakeEmbedding(0.5, 256)},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := server.RegisterProfileFace(context.Background(), tt.req)
			assert.Error(t, err)
		})
	}
}

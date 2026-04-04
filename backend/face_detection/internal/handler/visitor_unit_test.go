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

func TestRegisterVisitorFace_InvalidInput(t *testing.T) {
	server := newUnitServer()

	tests := []struct {
		name string
		req  *fd.RegisterVisitorFaceRequest
	}{
		{
			"empty embeddings slice returns error",
			&fd.RegisterVisitorFaceRequest{
				FaceEmbedding: []float32{},
				SessionToken:  "any-token",
				VisitorName:   "test_visitor",
			},
		},
		{
			"invalid embedding length returns error",
			&fd.RegisterVisitorFaceRequest{
				FaceEmbedding: test.MakeEmbedding(0.5, 256),
				SessionToken:  "any-token",
				VisitorName:   "test_visitor",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := server.RegisterVisitorFace(context.Background(), tt.req)
			assert.Error(t, err)
		})
	}
}

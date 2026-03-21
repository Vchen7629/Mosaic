//go:build unit

package service

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fd "mosaic-client.com/gen/face_detection"
	"mosaic-client.com/internal/test"
)

func TestSyncProfile(t *testing.T) {
	t.Run("Returns error on invalid base64 frame", func(t *testing.T) {
		conn, _ := newSafeConn(t)
		client := &test.MockFaceClient{}

		err := SyncProfile([]string{"not-valid-base64!!!"}, conn, client)

		assert.ErrorContains(t, err, "frame decode error", "invalid base64 should return a decode error")
	})

	t.Run("Returns error on SyncProfile gRPC error", func(t *testing.T) {
		conn, _ := newSafeConn(t)
		client := &test.MockFaceClient{
			SyncProfileFunc: func(_ *fd.SyncProfileRequest) (*fd.SyncProfileResponse, error) {
				return nil, status.Error(codes.Internal, "internal error")
			},
		}

		frame := base64.StdEncoding.EncodeToString([]byte("frame"))
		err := SyncProfile([]string{frame}, conn, client)

		assert.ErrorContains(t, err, "Sync gRPC error", "gRPC error should be propagated")
	})

	t.Run("Returns nil and no WriteJSON when no face detected", func(t *testing.T) {
		conn, capture := newSafeConn(t)
		client := &test.MockFaceClient{
			SyncProfileFunc: func(_ *fd.SyncProfileRequest) (*fd.SyncProfileResponse, error) {
				return &fd.SyncProfileResponse{FaceDetected: false}, nil
			},
		}

		frame := base64.StdEncoding.EncodeToString([]byte("frame"))
		err := SyncProfile([]string{frame}, conn, client)
		drainMessages()

		assert.NoError(t, err, "no face detected should return nil")
		assert.Empty(t, capture.Messages(), "no message should be written to frontend")
	})

	t.Run("Writes profile_face_response for existing known face", func(t *testing.T) {
		conn, capture := newSafeConn(t)
		client := &test.MockFaceClient{
			SyncProfileFunc: func(_ *fd.SyncProfileRequest) (*fd.SyncProfileResponse, error) {
				return &fd.SyncProfileResponse{FaceDetected: true, NewFace: false, ProfileId: 3}, nil
			},
		}

		frame := base64.StdEncoding.EncodeToString([]byte("frame"))
		err := SyncProfile([]string{frame}, conn, client)
		drainMessages()

		assert.NoError(t, err)
		msgs := capture.Messages()
		assert.Len(t, msgs, 1, "should write exactly one message")
		assert.Equal(t, "profile_face_response", jsonType(t, msgs[0]))
	})

	t.Run("Registers new face and writes profile_face_response for new face", func(t *testing.T) {
		conn, capture := newSafeConn(t)
		client := &test.MockFaceClient{
			SyncProfileFunc: func(_ *fd.SyncProfileRequest) (*fd.SyncProfileResponse, error) {
				return &fd.SyncProfileResponse{
					FaceDetected:  true,
					NewFace:       true,
					FaceEmbedding: []*fd.FaceEmbedding{{FaceEmbedding: []float32{0.1, 0.2}}},
				}, nil
			},
			RegisterProfileFaceFunc: func(_ *fd.RegisterProfileFaceRequest) (*fd.RegisterProfileFaceResponse, error) {
				return &fd.RegisterProfileFaceResponse{Success: true, ProfileId: 5}, nil
			},
		}

		frame := base64.StdEncoding.EncodeToString([]byte("frame"))
		err := SyncProfile([]string{frame}, conn, client)
		drainMessages()

		assert.NoError(t, err)
		msgs := capture.Messages()
		assert.Len(t, msgs, 1, "should write exactly one message")
		assert.Equal(t, "profile_face_response", jsonType(t, msgs[0]))
	})

	t.Run("Returns error on RegisterProfileFace gRPC error", func(t *testing.T) {
		conn, _ := newSafeConn(t)
		client := &test.MockFaceClient{
			SyncProfileFunc: func(_ *fd.SyncProfileRequest) (*fd.SyncProfileResponse, error) {
				return &fd.SyncProfileResponse{
					FaceDetected:  true,
					NewFace:       true,
					FaceEmbedding: []*fd.FaceEmbedding{{FaceEmbedding: []float32{0.1}}},
				}, nil
			},
			RegisterProfileFaceFunc: func(_ *fd.RegisterProfileFaceRequest) (*fd.RegisterProfileFaceResponse, error) {
				return nil, status.Error(codes.Internal, "register failed")
			},
		}

		frame := base64.StdEncoding.EncodeToString([]byte("frame"))
		err := SyncProfile([]string{frame}, conn, client)

		assert.ErrorContains(t, err, "RegisterProfileFace gRPC error", "gRPC error should be propagated")
	})
}
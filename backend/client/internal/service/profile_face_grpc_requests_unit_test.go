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

var validFrame = base64.StdEncoding.EncodeToString([]byte("frame"))

func TestSyncProfile(t *testing.T) {
	t.Run("Error cases", func(t *testing.T) {
		tests := []struct {
			name   string
			frames []string
			client *test.MockFaceClient
			errMsg string
		}{
			{
				name:   "invalid base64 frame",
				frames: []string{"not-valid-base64!!!"},
				client: &test.MockFaceClient{},
				errMsg: "frame decode error",
			},
			{
				name:   "SyncProfile gRPC error",
				frames: []string{validFrame},
				client: &test.MockFaceClient{
					SyncProfileFunc: func(_ *fd.SyncProfileRequest) (*fd.SyncProfileResponse, error) {
						return nil, status.Error(codes.Internal, "internal error")
					},
				},
				errMsg: "sync gRPC error",
			},
			{
				name:   "RegisterProfileFace gRPC error on new face",
				frames: []string{validFrame},
				client: &test.MockFaceClient{
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
				},
				errMsg: "RegisterProfileFace gRPC error",
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				conn, _ := newSafeConn(t)
				err := SyncProfile(tc.frames, conn, tc.client)
				assert.ErrorContains(t, err, tc.errMsg)
			})
		}
	})

	t.Run("Response handling", func(t *testing.T) {
		tests := []struct {
			name         string
			syncResp     *fd.SyncProfileResponse
			wantMsgCount int
			wantMsgType  string
		}{
			{
				name:         "no face detected writes nothing to frontend",
				syncResp:     &fd.SyncProfileResponse{FaceDetected: false},
				wantMsgCount: 0,
			},
			{
				name:         "known existing face writes profile_face_response",
				syncResp:     &fd.SyncProfileResponse{FaceDetected: true, NewFace: false, SessionToken: "tok_existing"},
				wantMsgCount: 1,
				wantMsgType:  "profile_face_response",
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				conn, capture := newSafeConn(t)
				client := &test.MockFaceClient{
					SyncProfileFunc: func(_ *fd.SyncProfileRequest) (*fd.SyncProfileResponse, error) {
						return tc.syncResp, nil
					},
				}

				err := SyncProfile([]string{validFrame}, conn, client)
				drainMessages()

				assert.NoError(t, err)
				msgs := capture.Messages()
				assert.Len(t, msgs, tc.wantMsgCount)
				if tc.wantMsgType != "" {
					assert.Equal(t, tc.wantMsgType, jsonType(t, msgs[0]))
				}
			})
		}
	})

	t.Run("New face registers and writes profile_face_response", func(t *testing.T) {
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
				return &fd.RegisterProfileFaceResponse{Success: true, SessionToken: "tok_new"}, nil
			},
		}

		err := SyncProfile([]string{validFrame}, conn, client)
		drainMessages()

		assert.NoError(t, err)
		msgs := capture.Messages()
		assert.Len(t, msgs, 1)
		assert.Equal(t, "profile_face_response", jsonType(t, msgs[0]))
	})
}

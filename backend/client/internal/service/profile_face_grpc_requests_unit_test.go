//go:build unit

package service

import (
	"encoding/base64"
	"encoding/json"
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
				name:   "empty frames slice still calls SyncProfile",
				frames: []string{},
				client: &test.MockFaceClient{
					SyncProfileFunc: func(_ *fd.SyncProfileRequest) (*fd.SyncProfileResponse, error) {
						return nil, status.Error(codes.Internal, "internal error")
					},
				},
				errMsg: "sync gRPC error",
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
				name:         "no face detected writes sync_profile_retry to frontend",
				syncResp:     &fd.SyncProfileResponse{FaceDetected: false},
				wantMsgCount: 1,
				wantMsgType:  "sync_profile_retry",
			},
			{
				name:         "existing face writes profile_face_response",
				syncResp:     &fd.SyncProfileResponse{FaceDetected: true, SessionToken: "tok_existing"},
				wantMsgCount: 1,
				wantMsgType:  "profile_face_response",
			},
			{
				name:         "new face registered internally writes profile_face_response",
				syncResp:     &fd.SyncProfileResponse{FaceDetected: true, Success: true, SessionToken: "tok_new"},
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

	t.Run("Session token is forwarded to frontend", func(t *testing.T) {
		conn, capture := newSafeConn(t)
		client := &test.MockFaceClient{
			SyncProfileFunc: func(_ *fd.SyncProfileRequest) (*fd.SyncProfileResponse, error) {
				return &fd.SyncProfileResponse{FaceDetected: true, SessionToken: "tok_abc123"}, nil
			},
		}

		err := SyncProfile([]string{validFrame}, conn, client)
		drainMessages()

		assert.NoError(t, err)
		msgs := capture.Messages()
		assert.Len(t, msgs, 1)

		var res ProfileSyncRes
		assert.NoError(t, json.Unmarshal(msgs[0], &res))
		assert.Equal(t, "profile_face_response", res.Type)
		assert.Equal(t, "tok_abc123", res.SessionToken)
	})

	t.Run("Multiple frames are all decoded and sent", func(t *testing.T) {
		frame1 := base64.StdEncoding.EncodeToString([]byte("frame1"))
		frame2 := base64.StdEncoding.EncodeToString([]byte("frame2"))

		var capturedReq *fd.SyncProfileRequest
		conn, _ := newSafeConn(t)
		client := &test.MockFaceClient{
			SyncProfileFunc: func(req *fd.SyncProfileRequest) (*fd.SyncProfileResponse, error) {
				capturedReq = req
				return &fd.SyncProfileResponse{FaceDetected: true, SessionToken: "tok"}, nil
			},
		}

		err := SyncProfile([]string{frame1, frame2}, conn, client)
		drainMessages()

		assert.NoError(t, err)
		assert.NotNil(t, capturedReq)
		assert.Len(t, capturedReq.FaceBytes, 2)
		assert.Equal(t, []byte("frame1"), capturedReq.FaceBytes[0])
		assert.Equal(t, []byte("frame2"), capturedReq.FaceBytes[1])
	})
}

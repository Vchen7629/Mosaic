package handler_test

import (
	"encoding/base64"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	fd "mosaic-client.com/gen/face_detection"
	"mosaic-client.com/internal/handler"
	"mosaic-client.com/internal/test"
)

func dialWS(t *testing.T, h http.Handler) (*websocket.Conn, func()) {
	t.Helper()
	srv := httptest.NewServer(h)
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	return conn, srv.Close
}

func newTestHandler(face *test.MockFaceClient, audio *test.MockAudioClient, briefing *test.MockBriefingClient) *handler.WebSocketHandler {
	return &handler.WebSocketHandler{
		Logger:         slog.Default(),
		FaceClient:     face,
		AudioClient:    audio,
		BriefingClient: briefing,
	}
}

// verifies each message type reaches the correct gRPC client.
func TestHandleWebSocketRouting(t *testing.T) {
	tests := []struct {
		name  string
		msg   handler.Message
		setup func(*test.MockFaceClient, *test.MockAudioClient, *test.MockBriefingClient) func() bool
	}{
		{
			name: "sync_profile routes to SyncProfile",
			msg: handler.Message{
				Type:         "sync_profile",
				SessionToken: "tok",
				Frames:       []string{base64.StdEncoding.EncodeToString([]byte("frame"))},
			},
			setup: func(face *test.MockFaceClient, _ *test.MockAudioClient, _ *test.MockBriefingClient) func() bool {
				var called atomic.Int32
				face.SyncProfileFunc = func(_ *fd.SyncProfileRequest) (*fd.SyncProfileResponse, error) {
					called.Add(1)
					return &fd.SyncProfileResponse{Success: true}, nil
				}
				return func() bool { return called.Load() >= 1 }
			},
		},
		{
			name: "visitor_face routes to ProcessVisitorFaces",
			msg: handler.Message{
				Type:         "visitor_face",
				SessionToken: "tok",
				FaceBytes:    base64.StdEncoding.EncodeToString([]byte("face")),
			},
			setup: func(face *test.MockFaceClient, _ *test.MockAudioClient, _ *test.MockBriefingClient) func() bool {
				var called atomic.Int32
				face.ProcessVisitorFacesFunc = func(_ *fd.ProcessVisitorFacesRequest) (*fd.ProcessVisitorFacesResponse, error) {
					called.Add(1)
					return &fd.ProcessVisitorFacesResponse{FaceDetected: false}, nil
				}
				return func() bool { return called.Load() >= 1 }
			},
		},
		{
			name: "new_visitor_face routes to RegisterVisitorFace",
			msg: handler.Message{
				Type:          "new_visitor_face",
				SessionToken:  "tok",
				VisitorName:   "Alice",
				FaceEmbedding: "0.1,0.2,0.3,0.4",
			},
			setup: func(face *test.MockFaceClient, _ *test.MockAudioClient, _ *test.MockBriefingClient) func() bool {
				var called atomic.Int32
				face.RegisterVisitorFaceFunc = func(_ *fd.RegisterVisitorFaceRequest) (*fd.RegisterVisitorFaceResponse, error) {
					called.Add(1)
					return &fd.RegisterVisitorFaceResponse{Success: true}, nil
				}
				return func() bool { return called.Load() >= 1 }
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			face := &test.MockFaceClient{}
			audio := &test.MockAudioClient{}
			briefing := &test.MockBriefingClient{}

			checkFn := tc.setup(face, audio, briefing)

			conn, cleanup := dialWS(t, http.HandlerFunc(newTestHandler(face, audio, briefing).HandleWebSocket))
			defer cleanup()
			defer func() {
				err := conn.Close()
				assert.NoError(t, err)
			}()

			require.NoError(t, conn.WriteJSON(tc.msg))
			assert.Eventually(t, checkFn, time.Second, 10*time.Millisecond, tc.name)
		})
	}
}

// verifies no service is called for an unrecognised message type.
func TestHandleWebSocketUnknownTypeNoHandlerInvoked(t *testing.T) {
	var syncCalled, processCalled, registerCalled atomic.Int32

	face := &test.MockFaceClient{
		SyncProfileFunc: func(_ *fd.SyncProfileRequest) (*fd.SyncProfileResponse, error) {
			syncCalled.Add(1)
			return &fd.SyncProfileResponse{}, nil
		},
		ProcessVisitorFacesFunc: func(_ *fd.ProcessVisitorFacesRequest) (*fd.ProcessVisitorFacesResponse, error) {
			processCalled.Add(1)
			return &fd.ProcessVisitorFacesResponse{}, nil
		},
		RegisterVisitorFaceFunc: func(_ *fd.RegisterVisitorFaceRequest) (*fd.RegisterVisitorFaceResponse, error) {
			registerCalled.Add(1)
			return &fd.RegisterVisitorFaceResponse{}, nil
		},
	}
	audio := &test.MockAudioClient{}
	briefing := &test.MockBriefingClient{}

	conn, cleanup := dialWS(t, http.HandlerFunc(newTestHandler(face, audio, briefing).HandleWebSocket))
	defer cleanup()
	defer func() {
		err := conn.Close()
		assert.NoError(t, err)
	}()

	require.NoError(t, conn.WriteJSON(handler.Message{Type: "unknown_type", SessionToken: "tok"}))

	assert.Never(t, func() bool {
		return syncCalled.Load() > 0 || processCalled.Load() > 0 ||
			registerCalled.Load() > 0 || audio.SaveCalls > 0 || briefing.GenerateCalls > 0
	}, 200*time.Millisecond, 10*time.Millisecond, "no handler should be invoked for unknown message type")
}

// verifies that gRPC errors are handled gracefully
// and the connection stays open for subsequent messages.
func TestHandleWebSocketErrorPaths(t *testing.T) {
	tests := []struct {
		name  string
		msg   handler.Message
		setup func(*test.MockFaceClient, *test.MockAudioClient) func() bool
	}{
		{
			name: "sync_profile gRPC error does not close connection",
			msg: handler.Message{
				Type:         "sync_profile",
				SessionToken: "tok",
				Frames:       []string{base64.StdEncoding.EncodeToString([]byte("frame"))},
			},
			setup: func(face *test.MockFaceClient, _ *test.MockAudioClient) func() bool {
				var called atomic.Int32
				face.SyncProfileFunc = func(_ *fd.SyncProfileRequest) (*fd.SyncProfileResponse, error) {
					called.Add(1)
					return nil, errors.New("rpc error")
				}
				return func() bool { return called.Load() >= 1 }
			},
		},
		{
			name: "visitor_face gRPC error does not close connection",
			msg: handler.Message{
				Type:         "visitor_face",
				SessionToken: "tok",
				FaceBytes:    base64.StdEncoding.EncodeToString([]byte("face")),
			},
			setup: func(face *test.MockFaceClient, _ *test.MockAudioClient) func() bool {
				var called atomic.Int32
				face.ProcessVisitorFacesFunc = func(_ *fd.ProcessVisitorFacesRequest) (*fd.ProcessVisitorFacesResponse, error) {
					called.Add(1)
					return nil, errors.New("rpc error")
				}
				return func() bool { return called.Load() >= 1 }
			},
		},
		{
			name: "new_visitor_face gRPC error does not close connection",
			msg: handler.Message{
				Type:          "new_visitor_face",
				SessionToken:  "tok",
				VisitorName:   "Alice",
				FaceEmbedding: "0.1,0.2,0.3,0.4",
			},
			setup: func(face *test.MockFaceClient, _ *test.MockAudioClient) func() bool {
				var called atomic.Int32
				face.RegisterVisitorFaceFunc = func(_ *fd.RegisterVisitorFaceRequest) (*fd.RegisterVisitorFaceResponse, error) {
					called.Add(1)
					return nil, errors.New("rpc error")
				}
				return func() bool { return called.Load() >= 1 }
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			face := &test.MockFaceClient{}
			audio := &test.MockAudioClient{}
			briefing := &test.MockBriefingClient{}

			checkFn := tc.setup(face, audio)

			conn, cleanup := dialWS(t, http.HandlerFunc(newTestHandler(face, audio, briefing).HandleWebSocket))
			defer cleanup()
			defer func() {
				err := conn.Close()
				assert.NoError(t, err)
			}()

			require.NoError(t, conn.WriteJSON(tc.msg))

			// verify the error path was hit
			assert.Eventually(t, checkFn, time.Second, 10*time.Millisecond, "expected gRPC call to be made")

			// verify the connection is still open after the error
			require.NoError(t, conn.WriteJSON(handler.Message{Type: "unknown_type", SessionToken: "tok"}),
				"connection must stay open after a handler error")
		})
	}
}

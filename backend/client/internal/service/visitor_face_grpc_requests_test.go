//go:build unit

package service

import (
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fd "mosaic-client.com/gen/face_detection"
	"mosaic-client.com/internal/test"
)

// newSafeConn spins up a WSCapture and wraps its connection in a SafeConn.
func newSafeConn(t *testing.T) (*SafeConn, *test.WSCapture) {
	t.Helper()
	capture := test.NewWSCapture(t)
	return &SafeConn{Conn: capture.Conn}, capture
}

// drainMessages waits briefly for the capture server to receive pending writes.
func drainMessages() { time.Sleep(50 * time.Millisecond) }

// jsonType unmarshals the "type" field from a raw JSON message.
func jsonType(t *testing.T, msg json.RawMessage) string {
	t.Helper()
	var m map[string]json.RawMessage
	assert.NoError(t, json.Unmarshal(msg, &m))
	var typ string
	assert.NoError(t, json.Unmarshal(m["type"], &typ))
	return typ
}

func TestProcessVisitorImage(t *testing.T) {
	t.Run("Returns error on invalid base64 frame", func(t *testing.T) {
		conn, _ := newSafeConn(t)
		client := &test.MockFaceClient{}

		err := ProcessVisitorImage(slog.Default(), "not-valid-base64!!!", "1", conn, client)

		assert.ErrorContains(t, err, "Frame decode error", "invalid base64 should return a decode error")
	})

	t.Run("Returns nil and no WriteJSON when no face detected", func(t *testing.T) {
		conn, capture := newSafeConn(t)
		client := &test.MockFaceClient{
			ProcessVisitorFacesFunc: func(_ *fd.ProcessVisitorFacesRequest) (*fd.ProcessVisitorFacesResponse, error) {
				return &fd.ProcessVisitorFacesResponse{FaceDetected: false}, nil
			},
		}

		frame := base64.StdEncoding.EncodeToString([]byte("frame"))
		err := ProcessVisitorImage(slog.Default(), frame, "1", conn, client)
		drainMessages()

		assert.NoError(t, err, "no face detected should return nil")
		assert.Empty(t, capture.Messages(), "no message should be written to frontend")
	})

	t.Run("Returns nil and no WriteJSON for non-visitor face", func(t *testing.T) {
		conn, capture := newSafeConn(t)
		client := &test.MockFaceClient{
			ProcessVisitorFacesFunc: func(_ *fd.ProcessVisitorFacesRequest) (*fd.ProcessVisitorFacesResponse, error) {
				return &fd.ProcessVisitorFacesResponse{FaceDetected: true, NonVisitorFace: true}, nil
			},
		}

		frame := base64.StdEncoding.EncodeToString([]byte("frame"))
		err := ProcessVisitorImage(slog.Default(), frame, "1", conn, client)
		drainMessages()

		assert.NoError(t, err, "non-visitor face should return nil")
		assert.Empty(t, capture.Messages(), "no message should be written to frontend for profile's own face")
	})

	t.Run("Returns error on ProcessVisitorFaces gRPC error", func(t *testing.T) {
		conn, _ := newSafeConn(t)
		client := &test.MockFaceClient{
			ProcessVisitorFacesFunc: func(_ *fd.ProcessVisitorFacesRequest) (*fd.ProcessVisitorFacesResponse, error) {
				return nil, status.Error(codes.Internal, "internal error")
			},
		}

		frame := base64.StdEncoding.EncodeToString([]byte("frame"))
		err := ProcessVisitorImage(slog.Default(), frame, "1", conn, client)

		assert.ErrorContains(t, err, "ProcessVisitorFace gRPC error", "gRPC error should be propagated")
	})

	t.Run("Writes new_visitor_register for unknown face", func(t *testing.T) {
		t.Cleanup(ClearSeenVisitors)
		conn, capture := newSafeConn(t)
		client := &test.MockFaceClient{
			ProcessVisitorFacesFunc: func(_ *fd.ProcessVisitorFacesRequest) (*fd.ProcessVisitorFacesResponse, error) {
				return &fd.ProcessVisitorFacesResponse{
					FaceDetected: true,
					Faces:        []*fd.FaceResult{{IsKnown: false, FaceEmbedding: []float32{0.1, 0.2}}},
				}, nil
			},
		}

		frame := base64.StdEncoding.EncodeToString([]byte("frame"))
		err := ProcessVisitorImage(slog.Default(), frame, "1", conn, client)
		drainMessages()

		assert.NoError(t, err)
		msgs := capture.Messages()
		assert.Len(t, msgs, 1, "should write exactly one message")
		assert.Equal(t, "new_visitor_register", jsonType(t, msgs[0]))
	})

	t.Run("Writes visitor_briefing_response for known unseen visitor", func(t *testing.T) {
		t.Cleanup(ClearSeenVisitors)
		conn, capture := newSafeConn(t)
		client := &test.MockFaceClient{
			ProcessVisitorFacesFunc: func(_ *fd.ProcessVisitorFacesRequest) (*fd.ProcessVisitorFacesResponse, error) {
				return &fd.ProcessVisitorFacesResponse{
					FaceDetected: true,
					Faces: []*fd.FaceResult{{
						IsKnown: true, VisitorId: 42,
						Name: "Alice", Briefing: "Alice's briefing",
					}},
				}, nil
			},
		}

		frame := base64.StdEncoding.EncodeToString([]byte("frame"))
		err := ProcessVisitorImage(slog.Default(), frame, "1", conn, client)
		drainMessages()

		assert.NoError(t, err)
		msgs := capture.Messages()
		assert.Len(t, msgs, 1, "should write exactly one message")
		assert.Equal(t, "visitor_briefing_response", jsonType(t, msgs[0]))
	})

	t.Run("Does not write visitor_briefing_response for already seen visitor", func(t *testing.T) {
		t.Cleanup(ClearSeenVisitors)
		AddSeenVisitor(99)
		conn, capture := newSafeConn(t)
		client := &test.MockFaceClient{
			ProcessVisitorFacesFunc: func(_ *fd.ProcessVisitorFacesRequest) (*fd.ProcessVisitorFacesResponse, error) {
				return &fd.ProcessVisitorFacesResponse{
					FaceDetected: true,
					Faces:        []*fd.FaceResult{{IsKnown: true, VisitorId: 99}},
				}, nil
			},
		}

		frame := base64.StdEncoding.EncodeToString([]byte("frame"))
		err := ProcessVisitorImage(slog.Default(), frame, "1", conn, client)
		drainMessages()

		assert.NoError(t, err)
		assert.Empty(t, capture.Messages(), "already-seen visitor should not be re-sent to frontend")
	})

	t.Run("Marks visitor as seen after sending briefing", func(t *testing.T) {
		t.Cleanup(ClearSeenVisitors)
		conn, _ := newSafeConn(t)
		client := &test.MockFaceClient{
			ProcessVisitorFacesFunc: func(_ *fd.ProcessVisitorFacesRequest) (*fd.ProcessVisitorFacesResponse, error) {
				return &fd.ProcessVisitorFacesResponse{
					FaceDetected: true,
					Faces:        []*fd.FaceResult{{IsKnown: true, VisitorId: 55}},
				}, nil
			},
		}

		frame := base64.StdEncoding.EncodeToString([]byte("frame"))
		assert.NoError(t, ProcessVisitorImage(slog.Default(), frame, "1", conn, client))

		assert.True(t, HasSeenVisitor(55), "visitor should be marked as seen after sending briefing")
	})
}

func TestRegisterNewVisitorFace(t *testing.T) {
	t.Run("Returns error on RegisterVisitorFace gRPC error", func(t *testing.T) {
		conn, _ := newSafeConn(t)
		client := &test.MockFaceClient{
			RegisterVisitorFaceFunc: func(_ *fd.RegisterVisitorFaceRequest) (*fd.RegisterVisitorFaceResponse, error) {
				return nil, status.Error(codes.Internal, "internal error")
			},
		}

		err := RegisterNewVisitorFace("0.1,0.2", "1", "Alice", conn, client)

		assert.ErrorContains(t, err, "RegisterVisitorFace gRPC error", "gRPC error should be propagated")
	})

	t.Run("Returns nil and no WriteJSON when resp.Success is false", func(t *testing.T) {
		conn, capture := newSafeConn(t)
		client := &test.MockFaceClient{
			RegisterVisitorFaceFunc: func(_ *fd.RegisterVisitorFaceRequest) (*fd.RegisterVisitorFaceResponse, error) {
				return &fd.RegisterVisitorFaceResponse{Success: false}, nil
			},
		}

		err := RegisterNewVisitorFace("0.1,0.2", "1", "Alice", conn, client)
		drainMessages()

		assert.NoError(t, err, "success=false should return nil")
		assert.Empty(t, capture.Messages(), "no message should be written when registration fails")
	})

	t.Run("Writes register_visitor_resp on successful registration", func(t *testing.T) {
		conn, capture := newSafeConn(t)
		client := &test.MockFaceClient{
			RegisterVisitorFaceFunc: func(_ *fd.RegisterVisitorFaceRequest) (*fd.RegisterVisitorFaceResponse, error) {
				return &fd.RegisterVisitorFaceResponse{Success: true, VisitorId: 7}, nil
			},
		}

		err := RegisterNewVisitorFace("0.1,0.2", "1", "Alice", conn, client)
		drainMessages()

		assert.NoError(t, err)
		msgs := capture.Messages()
		assert.Len(t, msgs, 1, "should write exactly one message")
		assert.Equal(t, "register_visitor_resp", jsonType(t, msgs[0]))
	})

	t.Run("Returns error on invalid embedding string", func(t *testing.T) {
		conn, _ := newSafeConn(t)
		client := &test.MockFaceClient{}

		err := RegisterNewVisitorFace("not,a,float", "1", "Alice", conn, client)

		assert.Error(t, err, "invalid embedding should return a parse error")
	})
}

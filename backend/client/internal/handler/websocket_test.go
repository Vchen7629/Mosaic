//go:build integration

package handler_test

import (
	"encoding/base64"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"mosaic-client.com/internal/handler"
	"mosaic-client.com/internal/test"
)

// verifies that each goroutine receives its own copy of the message
// data and not a later-overwritten value from the read loop
func TestVisitorFace_GoroutineCapture(t *testing.T) {
	faceStub := &test.StubFaceClient{}
	h := &handler.WebSocketHandler{
		AudioClient: &test.StubAudioClient{},
		FaceClient:  faceStub,
	}

	conn, cleanup := test.DialWS(t, http.HandlerFunc(h.HandleWebSocket))
	defer cleanup()
	defer conn.Close()

	payloads := [][]byte{
		[]byte("face_payload_one"),
		[]byte("face_payload_two"),
		[]byte("face_payload_three"),
	}

	for _, p := range payloads {
		err := conn.WriteJSON(handler.Message{
			Type:      "visitor_face",
			FaceBytes: base64.StdEncoding.EncodeToString(p),
			ProfileID: "1",
		})
		assert.NoError(t, err)
	}

	// give goroutines time to run and call through to the stub
	time.Sleep(200 * time.Millisecond)

	faceStub.Mu.Lock()
	calls := faceStub.ProcessVisitorCalls
	faceStub.Mu.Unlock()

	assert.Len(t, calls, len(payloads), "expected one gRPC call per msg")

	received := make([][]byte, len(calls))
	for i, c := range calls {
		received[i] = c.FaceBytes
	}

	assert.ElementsMatch(t, payloads, received, "each goroutine must have captured its own message's FaceBytes")
}

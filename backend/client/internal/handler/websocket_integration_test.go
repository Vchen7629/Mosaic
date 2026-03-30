//go:build integration

package handler_test

import (
	"encoding/base64"
	"errors"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"mosaic-client.com/internal/handler"
	"mosaic-client.com/internal/service"
	"mosaic-client.com/internal/test"
)

// verifies that each goroutine receives its own copy of the message
// data and not a later-overwritten value from the read loop
func TestVisitorFace_GoroutineCapture(t *testing.T) {
	faceStub := &test.StubFaceClient{}
	h := &handler.WebSocketHandler{
		Logger:      slog.Default(),
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

// verifies that a "sync_profile" message routes to the face client's SyncProfile RPC
func TestHandleWebSocket_SyncProfile_RouteToFaceService(t *testing.T) {
	faceStub := &test.StubFaceClient{}
	h := &handler.WebSocketHandler{
		Logger:      slog.Default(),
		AudioClient: &test.StubAudioClient{},
		FaceClient:  faceStub,
	}

	conn, cleanup := test.DialWS(t, http.HandlerFunc(h.HandleWebSocket))
	defer cleanup()
	defer conn.Close()

	err := conn.WriteJSON(handler.Message{
		Type:      "sync_profile",
		ProfileID: "1",
		Frames:    []string{base64.StdEncoding.EncodeToString([]byte("frame1"))},
	})
	assert.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	faceStub.Mu.Lock()
	called := faceStub.SyncProfileCalled
	faceStub.Mu.Unlock()

	assert.Equal(t, 1, called, "sync_profile must invoke SyncProfile on the face client")
}

// verifies that a "new_visitor_face" message routes to the face client's RegisterVisitorFace RPC
func TestHandleWebSocket_NewVisitorFace_RouteToRegisterFace(t *testing.T) {
	faceStub := &test.StubFaceClient{}
	h := &handler.WebSocketHandler{
		Logger:      slog.Default(),
		AudioClient: &test.StubAudioClient{},
		FaceClient:  faceStub,
	}

	conn, cleanup := test.DialWS(t, http.HandlerFunc(h.HandleWebSocket))
	defer cleanup()
	defer conn.Close()

	err := conn.WriteJSON(handler.Message{
		Type:          "new_visitor_face",
		ProfileID:     "1",
		VisitorName:   "Alice",
		FaceEmbedding: "0.1,0.2,0.3,0.4",
	})
	assert.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	faceStub.Mu.Lock()
	called := faceStub.RegisterVisitorCalled
	faceStub.Mu.Unlock()

	assert.Equal(t, 1, called, "new_visitor_face must invoke RegisterVisitorFace on the face client")
}

// verifies that a "save_audio_transcript" message routes to the audio client's SaveTranscript RPC
func TestHandleWebSocket_SaveAudioTranscript_RouteToSaveTranscript(t *testing.T) {
	audioStub := &test.StubAudioClient{}
	briefingStub := &test.StubBriefingClient{}
	h := &handler.WebSocketHandler{
		Logger:         slog.Default(),
		AudioClient:    audioStub,
		FaceClient:     &test.StubFaceClient{},
		BriefingClient: briefingStub,
	}

	conn, cleanup := test.DialWS(t, http.HandlerFunc(h.HandleWebSocket))
	defer cleanup()
	defer conn.Close()

	err := conn.WriteJSON(handler.Message{
		Type:          "save_audio_transcript",
		ProfileID:     "1",
		VisitorIDList: []string{"10", "20"},
	})
	assert.NoError(t, err)

	// save_audio_transcript is synchronous in the read loop; a subsequent
	// message will only be dispatched after it returns, so use it as a sentinel.
	err = conn.WriteJSON(handler.Message{
		Type:      "visitor_face",
		FaceBytes: base64.StdEncoding.EncodeToString([]byte("ping")),
		ProfileID: "1",
	})
	assert.NoError(t, err)

	assert.Eventually(t, func() bool {
		audioStub.Mu.Lock()
		defer audioStub.Mu.Unlock()
		return audioStub.SaveTranscriptCalled >= 1
	}, 2*time.Second, 50*time.Millisecond, "save_audio_transcript must invoke SaveTranscript on the audio client")
}

// verifies that Wg.Add(1) and Wg.Done() are paired for every "audio" message,
// so that a subsequent FlushAudio/Wg.Wait() does not block indefinitely.
func TestHandleWebSocket_Audio_WaitGroupAddAndDoneArePaired(t *testing.T) {
	h := &handler.WebSocketHandler{
		Logger:      slog.Default(),
		AudioClient: &test.StubAudioClient{},
		FaceClient:  &test.StubFaceClient{},
	}

	conn, cleanup := test.DialWS(t, http.HandlerFunc(h.HandleWebSocket))
	defer cleanup()
	defer conn.Close()

	// A small 4-byte payload: won't fill the batch buffer, so TranscribeAudio
	// is never called, but Wg.Add(1)/Wg.Done() must still pair correctly.
	audioPayload := base64.StdEncoding.EncodeToString(make([]byte, 4))

	const n = 5
	for range n {
		err := conn.WriteJSON(handler.Message{
			Type:      "audio",
			AudioData: audioPayload,
			ProfileID: "1",
		})
		assert.NoError(t, err)
	}

	done := make(chan struct{})
	go func() {
		service.Wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// all Wg.Done() calls arrived — pass
	case <-time.After(2 * time.Second):
		t.Fatal("Wg.Wait() timed out: Wg.Done() was not called for every audio message")
	}
}

// verifies that when SaveTranscriptWithRetry succeeds, GenerateConversationBriefing is called.
func TestHandleWebSocket_SaveAudioTranscript_Success_CallsBriefing(t *testing.T) {
	audioStub := &test.StubAudioClient{}
	briefingStub := &test.StubBriefingClient{}
	h := &handler.WebSocketHandler{
		Logger:         slog.Default(),
		AudioClient:    audioStub,
		FaceClient:     &test.StubFaceClient{},
		BriefingClient: briefingStub,
	}

	conn, cleanup := test.DialWS(t, http.HandlerFunc(h.HandleWebSocket))
	defer cleanup()
	defer conn.Close()

	err := conn.WriteJSON(handler.Message{
		Type:          "save_audio_transcript",
		ProfileID:     "1",
		VisitorIDList: []string{"10"},
	})
	assert.NoError(t, err)

	// GenerateConversationBriefing runs in a goroutine inside handleSaveAudioTranscript;
	// poll until it arrives or the deadline expires.
	assert.Eventually(t, func() bool {
		briefingStub.Mu.Lock()
		defer briefingStub.Mu.Unlock()
		return briefingStub.GenerateConversationBriefingCalled >= 1
	}, 2*time.Second, 50*time.Millisecond, "GenerateConversationBriefing must be called when SaveTranscript succeeds")
}

// verifies that when SaveTranscriptWithRetry fails, GetSeenVisitors and
// GenerateConversationBriefing are never invoked.
//
// NOTE: SaveTranscriptWithRetry retries 3 times with exponential backoff (1s + 2s + 4s),
// so this test takes ~7 seconds to complete by design.
func TestHandleWebSocket_SaveAudioTranscript_SaveFails_SkipsBriefing(t *testing.T) {
	audioStub := &test.StubAudioClient{SaveTranscriptErr: errors.New("db unavailable")}
	briefingStub := &test.StubBriefingClient{}
	faceStub := &test.StubFaceClient{}
	h := &handler.WebSocketHandler{
		Logger:         slog.Default(),
		AudioClient:    audioStub,
		FaceClient:     faceStub,
		BriefingClient: briefingStub,
	}

	conn, cleanup := test.DialWS(t, http.HandlerFunc(h.HandleWebSocket))
	defer cleanup()
	defer conn.Close()

	err := conn.WriteJSON(handler.Message{
		Type:          "save_audio_transcript",
		ProfileID:     "1",
		VisitorIDList: []string{"10"},
	})
	assert.NoError(t, err)

	// Send a sentinel visitor_face message. Because save_audio_transcript is processed
	// synchronously in the read loop, visitor_face won't be dispatched until
	// handleSaveAudioTranscript returns (after all retries exhaust).
	err = conn.WriteJSON(handler.Message{
		Type:      "visitor_face",
		FaceBytes: base64.StdEncoding.EncodeToString([]byte("sentinel")),
		ProfileID: "1",
	})
	assert.NoError(t, err)

	// Wait for the sentinel to be processed, proving handleSaveAudioTranscript has returned.
	assert.Eventually(t, func() bool {
		faceStub.Mu.Lock()
		defer faceStub.Mu.Unlock()
		return len(faceStub.ProcessVisitorCalls) >= 1
	}, 10*time.Second, 200*time.Millisecond, "sentinel visitor_face was never processed")

	briefingStub.Mu.Lock()
	briefingCalled := briefingStub.GenerateConversationBriefingCalled
	briefingStub.Mu.Unlock()

	assert.Equal(t, 0, briefingCalled, "GenerateConversationBriefing must not be called when SaveTranscript fails")
}

// verifies that an unrecognised message type does not invoke any handler,
// leaving all stub call counts at zero.
func TestHandleWebSocket_UnknownMessageType_NoHandlerInvoked(t *testing.T) {
	audioStub := &test.StubAudioClient{}
	faceStub := &test.StubFaceClient{}
	briefingStub := &test.StubBriefingClient{}
	h := &handler.WebSocketHandler{
		Logger:         slog.Default(),
		AudioClient:    audioStub,
		FaceClient:     faceStub,
		BriefingClient: briefingStub,
	}

	conn, cleanup := test.DialWS(t, http.HandlerFunc(h.HandleWebSocket))
	defer cleanup()
	defer conn.Close()

	err := conn.WriteJSON(handler.Message{
		Type:      "unknown_type",
		ProfileID: "1",
	})
	assert.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	faceStub.Mu.Lock()
	syncCalled := faceStub.SyncProfileCalled
	visitorCalls := len(faceStub.ProcessVisitorCalls)
	registerCalled := faceStub.RegisterVisitorCalled
	faceStub.Mu.Unlock()

	audioStub.Mu.Lock()
	saveCalled := audioStub.SaveTranscriptCalled
	audioStub.Mu.Unlock()

	briefingStub.Mu.Lock()
	briefingCalled := briefingStub.GenerateConversationBriefingCalled
	briefingStub.Mu.Unlock()

	assert.Equal(t, 0, syncCalled, "SyncProfile must not be called for unknown type")
	assert.Equal(t, 0, visitorCalls, "ProcessVisitorFaces must not be called for unknown type")
	assert.Equal(t, 0, registerCalled, "RegisterVisitorFace must not be called for unknown type")
	assert.Equal(t, 0, saveCalled, "SaveTranscript must not be called for unknown type")
	assert.Equal(t, 0, briefingCalled, "GenerateConversationBriefing must not be called for unknown type")
}

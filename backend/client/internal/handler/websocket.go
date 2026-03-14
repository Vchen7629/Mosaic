package handler

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	at "mosaic-client.com/gen/audio_transcription"
	fd "mosaic-client.com/gen/face_detection"
	"mosaic-client.com/internal/service"
)

type Message struct {
	Type string `json:"type"` // either "frame" or "audio"
	Data string `json:"data"` // base64 encoded
}

type WebSocketHandler struct {
	AudioClient at.AudioTranscriptionServiceClient
	FaceClient  fd.FaceDetectionServiceClient
}

var upgrader = websocket.Upgrader{
	HandshakeTimeout: time.Minute * 5, // times out after 5 mins
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins (Tauri apps connect from tauri:// protocol)
		return true
	},
}

func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	log.Printf("[WebSocket] Connection attempt from origin: %s", r.Header.Get("Origin"))

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WebSocket] Upgrade failed: %v", err)
		return
	}

	defer conn.Close()

	patientID := r.URL.Query().Get("patient_id")
	log.Printf("[WebSocket] Connected! Patient ID: %s", patientID)

	for {
		var msg Message

		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err) {
				log.Printf("[WebSocket] Unexpected close: %v", err)
			}
			break
		}

		switch msg.Type {
		case "face_frame":
			go service.ProcessImageFrame(msg.Data, patientID, conn, h.FaceClient)
		case "audio":
			service.Wg.Add(1)
			go func() {
				defer service.Wg.Done()
				service.ProcessAudio(msg.Data, patientID, h.AudioClient)
			}()
		}
	}

	ctx := context.Background()
	service.FlushAudio(ctx, patientID, h.AudioClient)
	err = service.SaveTranscriptWithRetry(ctx, patientID, h.AudioClient)
	if err != nil {
		log.Printf("error saving transcript: %v", err)
	}
}
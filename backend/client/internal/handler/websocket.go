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
	Type 			string `json:"type"`
	FaceBytes 		string `json:"face_bytes,omitempty"` // base64 encoded
	AudioData		string `json:"audio_data,omitempty"`
	FaceEmbedding	string `json:"face_embedding,omitempty"`
	ProfileID		string `json:"profile_id,omitempty"`
	VisitorName		string `json:"visitor_name,omitempty"`
	Frames		  []string `json:"frames,omitempty"`
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

	safe := &service.SafeConn{Conn: conn}

	log.Println("[WebSocket] Connected!")

	var msg Message

	for {
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err) {
				log.Printf("[WebSocket] Unexpected close: %v", err)
			}
			break
		}

		switch msg.Type {
		case "sync_profile":
			go func(m Message) {
				err := service.SyncProfile(m.Frames, safe, h.FaceClient)
				if err != nil {
					log.Printf("[sync_profile] error: %v", err)
				}
			}(msg)
		case "visitor_face":
			go func(m Message) {
				err = service.ProcessVisitorImage(m.FaceBytes, m.ProfileID, safe, h.FaceClient)
				if err != nil {
					log.Printf("[visitor_face] error: %v", err)
				}
			}(msg)
		case "new_visitor_face":
			go func(m Message) {
				err = service.RegisterNewVisitorFace(m.FaceEmbedding, m.ProfileID, m.VisitorName, safe, h.FaceClient)
				if err != nil {
					log.Printf("[new_visitor_face] error: %v", err)
				}
			}(msg)
		case "audio":
			service.Wg.Add(1)
			go func(m Message) {
				defer service.Wg.Done()
				err = service.ProcessAudio(m.AudioData, m.ProfileID, h.AudioClient)
				if err != nil {
					log.Printf("[audio] error: %v", err)
				}
			}(msg)
		}
	}

	if msg.ProfileID != "" {
		ctx := context.Background()
		service.FlushAudio(ctx, msg.ProfileID, h.AudioClient)
		err = service.SaveTranscriptWithRetry(ctx, msg.ProfileID, h.AudioClient)
		if err != nil {
			log.Printf("error saving transcript: %v", err)
		}
	}
}
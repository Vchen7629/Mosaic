package handler

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	at "mosaic-client.com/gen/audio_transcription"
	fd "mosaic-client.com/gen/face_detection"
	cb "mosaic-client.com/gen/conversation_briefing"
	"mosaic-client.com/internal/service"
)

type Message struct {
	Type 			string `json:"type"`
	FaceBytes 		string `json:"face_bytes,omitempty"` // base64 encoded
	AudioData		string `json:"audio_data,omitempty"`
	FaceEmbedding	string `json:"face_embedding,omitempty"`
	ProfileID		string `json:"profile_id,omitempty"`
	VisitorID		string `json:"visitor_id,omitempty"`
	VisitorIDList []string `json:"visitor_id_list,omitempty"`
	VisitorName		string `json:"visitor_name,omitempty"`
	Frames		  []string `json:"frames,omitempty"`
}

type WebSocketHandler struct {
	AudioClient 	at.AudioTranscriptionServiceClient
	FaceClient  	fd.FaceDetectionServiceClient
	BriefingClient 	cb.ConversationBriefingServiceClient
}

var upgrader = websocket.Upgrader{
	HandshakeTimeout: time.Minute * 5, // times out after 5 mins
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins (Tauri apps connect from tauri:// protocol)
		return true
	},
}

// Handles routing websocket messages to the correct gRPC goroutine routes to do processing
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
			go h.handleSyncProfile(msg, safe)
		case "visitor_face":
			go h.handleProcessVisitorFace(msg, safe)
		case "new_visitor_face":
			go h.handleRegisterVisitorFace(msg, safe)
		case "audio":
			service.Wg.Add(1)
			go h.handleAudio(msg)
		case "save_audio_transcript":
			h.handleSaveAudioTranscript(msg)
		}
	}
}

// sends the message to sync profile to do face detection and sync user profile
func (h *WebSocketHandler) handleSyncProfile(m Message, safe *service.SafeConn) {
	err := service.SyncProfile(m.Frames, safe, h.FaceClient)
	if err != nil {
		log.Printf("[sync_profile] error: %v", err)
	}
}

// runs every 2.5 frames, checks the sent face frame against visitors and decides
// whether to register a new face or fetch briefing for an existing face
func (h *WebSocketHandler) handleProcessVisitorFace(m Message, safe *service.SafeConn) {
	err := service.ProcessVisitorImage(m.FaceBytes, m.ProfileID, safe, h.FaceClient)
	if err != nil {
		log.Printf("[visitor_face] error: %v", err)
	}
}

// if the visitor face doesnt match a face in the db, this allows us to register the face
// as a new visitor face with the name
func (h *WebSocketHandler) handleRegisterVisitorFace(m Message, safe *service.SafeConn) {
	err := service.RegisterNewVisitorFace(m.FaceEmbedding, m.ProfileID, m.VisitorName, safe, h.FaceClient)
	if err != nil {
		log.Printf("[new_visitor_face] error: %v", err)
	}
}

// process the audio sent from use mic by transcribing it to text
func (h *WebSocketHandler) handleAudio(m Message) {
	defer service.Wg.Done()
	err := service.ProcessAudio(m.AudioData, m.ProfileID, h.AudioClient)
	if err != nil {
		log.Printf("[audio] error: %v", err)
	}
}

// Saves the transcript to the db and calls a gRPC service to generate briefing
func (h *WebSocketHandler) handleSaveAudioTranscript(m Message) {
	ctx := context.Background()
	service.FlushAudio(ctx, m.ProfileID, h.AudioClient)
	err := service.SaveTranscriptWithRetry(ctx, m.ProfileID, m.VisitorIDList, h.AudioClient)
	if err != nil {
		log.Printf("error saving transcript: %v", err)
	} else {
		visitorIDs := service.GetSeenVisitors()
		go func(m Message) {
			err = service.GenerateConversationBriefing(m.ProfileID, visitorIDs, h.BriefingClient)		
			if err != nil {
				log.Printf("[convo briefing] error: %v", err)
			}
		}(m)
	}
	service.ClearSeenVisitors()
}

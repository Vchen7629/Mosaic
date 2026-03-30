package handler

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	at "mosaic-client.com/gen/audio_transcription"
	cb "mosaic-client.com/gen/conversation_briefing"
	fd "mosaic-client.com/gen/face_detection"
	"mosaic-client.com/internal/service"
)

type Message struct {
	Type          string   `json:"type"`
	FaceBytes     string   `json:"face_bytes,omitempty"` // base64 encoded
	AudioData     string   `json:"audio_data,omitempty"`
	FaceEmbedding string   `json:"face_embedding,omitempty"`
	SessionToken  string   `json:"profile_id,omitempty"`
	VisitorID     string   `json:"visitor_id,omitempty"`
	VisitorIDList []string `json:"visitor_id_list,omitempty"`
	VisitorName   string   `json:"visitor_name,omitempty"`
	Frames        []string `json:"frames,omitempty"`
}

type WebSocketHandler struct {
	Logger         *slog.Logger
	AudioClient    at.AudioTranscriptionServiceClient
	FaceClient     fd.FaceDetectionServiceClient
	BriefingClient cb.ConversationBriefingServiceClient
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
	h.Logger.Debug("[WebSocket] Connection attempt from origin", "args", r.Header.Get("Origin"))

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.Logger.Error("[WebSocket] Upgrade failed", "err", err)
		return
	}

	defer func() {
		err := conn.Close()
		if err != nil {
			h.Logger.Warn("error closing websocket connection", "err", err)
		}
	}()

	safe := &service.SafeConn{Conn: conn}

	h.Logger.Debug("[WebSocket] Connected!")

	var msg Message

	for {
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err) {
				h.Logger.Debug("[WebSocket] Unexpected close", "err", err)
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
		h.Logger.Error("[sync_profile] error", "err", err)
	}
}

// runs every 2.5 frames, checks the sent face frame against visitors and decides
// whether to register a new face or fetch briefing for an existing face
func (h *WebSocketHandler) handleProcessVisitorFace(m Message, safe *service.SafeConn) {
	err := service.ProcessVisitorImage(h.Logger, m.FaceBytes, m.SessionToken, safe, h.FaceClient)
	if err != nil {
		h.Logger.Error("[visitor_face] error", "err", err)
	}
}

// if the visitor face doesnt match a face in the db, this allows us to register the face
// as a new visitor face with the name
func (h *WebSocketHandler) handleRegisterVisitorFace(m Message, safe *service.SafeConn) {
	err := service.RegisterNewVisitorFace(m.FaceEmbedding, m.SessionToken, m.VisitorName, safe, h.FaceClient)
	if err != nil {
		h.Logger.Error("[new_visitor_face] error", "err", err)
	}
}

// process the audio sent from use mic by transcribing it to text
func (h *WebSocketHandler) handleAudio(m Message) {
	defer service.Wg.Done()
	err := service.ProcessAudio(h.Logger, m.AudioData, m.SessionToken, h.AudioClient)
	if err != nil {
		h.Logger.Error("[audio] error", "err", err)
	}
}

// Saves the transcript to the db and calls a gRPC service to generate briefing
func (h *WebSocketHandler) handleSaveAudioTranscript(m Message) {
	ctx := context.Background()
	err := service.FlushAudio(h.Logger, ctx, m.SessionToken, h.AudioClient)
	if err != nil {
		h.Logger.Error("[save_audio_transcript] error flushing audio", "err", err)
	}

	err = service.SaveTranscriptWithRetry(ctx, m.SessionToken, m.VisitorIDList, h.AudioClient)
	if err != nil {
		h.Logger.Error("[save_audio_transcript] error saving transcript", "err", err)
	} else {
		visitorIDs := service.GetSeenVisitors()
		go func(m Message) {
			err = service.GenerateConversationBriefing(m.SessionToken, visitorIDs, h.BriefingClient)
			if err != nil {
				h.Logger.Error("[save_audio_transcript] error generating convo briefing", "err", err)
			}
		}(m)
	}
	service.ClearSeenVisitors()
}

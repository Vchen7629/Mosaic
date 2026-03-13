package handler

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"mosaic-client.com/internal/service"
)

type Message struct {
	Type string `json:"type"` // either "frame" or "audio"
	Data string `json:"data"` // base64 encoded
}

var upgrader = websocket.Upgrader{
	HandshakeTimeout: time.Minute * 5, // times out after 5 mins
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins (Tauri apps connect from tauri:// protocol)
		return true
	},
}

func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
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
			log.Println("Read ws error: ", err)
			break
		}

		switch msg.Type {
		case "frame":
			go service.ProcessImageFrame(msg.Data, patientID, conn)
		case "audio":
			log.Printf("[WebSocket] Received audio message, data length: %d bytes", len(msg.Data))
			go service.ProcessAudio(msg.Data, patientID)
		}
	}
}
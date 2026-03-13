package service

import (
	"encoding/base64"
	"fmt"

	"github.com/gorilla/websocket"
)

func ProcessImageFrame(frameData string, patientID string, conn *websocket.Conn) error {
	// Decode base64
	_, err := base64.StdEncoding.DecodeString(frameData)
	if err != nil {
		return fmt.Errorf("Frame decode error: %w", err)
	}

	// Compress JPEG frame for better performance

	// Send to face detection service via gRPC
	result := "hi"

	// Send result back to frontend?
	conn.WriteJSON(result)
	return nil
}

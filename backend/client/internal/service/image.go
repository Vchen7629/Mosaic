package service

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/gorilla/websocket"
	fd "mosaic-client.com/gen/face_detection"
)

func ProcessImageFrame(
	frameData string, 
	patientID string, 
	conn *websocket.Conn,
	client fd.FaceDetectionServiceClient,
) error {
	ctx := context.Background()
	// Decode base64
	faceBytes, err := base64.StdEncoding.DecodeString(frameData)
	if err != nil {
		return fmt.Errorf("Frame decode error: %w", err)
	}

	//log.Printf("[ProcessFace] Recieved face_data: %s", frameData)

	// Compress JPEG frame for better performance

	// Send to face detection service via gRPC
	resp, err := client.ProcessFaces(ctx, &fd.ProcessFacesRequest{
		FaceBytes: faceBytes,
		PatientId: patientID,
	})
	if err != nil {

	}

	// Send result back to frontend?
	conn.WriteJSON(resp)
	return nil
}

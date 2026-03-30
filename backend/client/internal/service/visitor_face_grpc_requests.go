package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"

	fd "mosaic-client.com/gen/face_detection"
)

type UnknownVisitorResponse struct {
	Type          string    `json:"type"`
	FaceEmbedding []float32 `json:"face_embedding"`
}

type KnownVisitorResponse struct {
	Type        string `json:"type"`
	VisitorName string `json:"visitor_name"`
	Briefing    string `json:"briefing"`
	VisitorID   int32  `json:"visitor_id"`
}

// Process images for potential visitor
func ProcessVisitorImage(
	logger *slog.Logger,
	frameData string,
	sessionToken string,
	conn *SafeConn,
	client fd.FaceDetectionServiceClient,
) error {
	ctx := context.Background()
	// Decode base64
	faceBytes, err := base64.StdEncoding.DecodeString(frameData)
	if err != nil {
		return fmt.Errorf("frame decode error: %w", err)
	}

	//log.Printf("[ProcessFace] Recieved face_data: %s", frameData)

	// Compress JPEG frame for better performance

	// Send to face detection service via gRPC
	resp, err := client.ProcessVisitorFaces(ctx, &fd.ProcessVisitorFacesRequest{
		FaceBytes:    faceBytes,
		SessionToken: sessionToken,
	})
	if err != nil {
		return fmt.Errorf("ProcessVisitorFace gRPC error: %w", err)
	}

	if !resp.FaceDetected {
		logger.Debug("[ProcessVisitorFace] no faces detected")
		return nil
	}

	if resp.NonVisitorFace {
		logger.Debug("[ProcessVisitorFace] Face detected is same as profile face, skipping")
		return nil
	}

	if len(resp.Faces) > 0 {
		processFaceResults(logger, resp.Faces, conn)
	}

	return nil
}

type RegisterVisitorRes struct {
	Type      string `json:"type"`
	VisitorID int32  `json:"visitor_id"`
	Success   bool   `json:"success"`
}

// Register a new visitor face
func RegisterNewVisitorFace(
	faceEmbedding string,
	sessionToken string,
	visitorName string,
	conn *SafeConn,
	client fd.FaceDetectionServiceClient,
) error {
	ctx := context.Background()

	faceEmb, err := ParseEmbedding(faceEmbedding)
	if err != nil {
		return err
	}

	resp, err := client.RegisterVisitorFace(ctx, &fd.RegisterVisitorFaceRequest{
		FaceEmbedding: faceEmb,
		SessionToken:  sessionToken,
		VisitorName:   visitorName,
	})

	if err != nil {
		return fmt.Errorf("RegisterVisitorFace gRPC error: %w", err)
	}

	// Todo: Implement retry with exponential backoff
	if !resp.Success {
		return nil
	}

	err = conn.WriteJSON(RegisterVisitorRes{Type: "register_visitor_resp", VisitorID: resp.VisitorId, Success: resp.Success})
	if err != nil {
		return fmt.Errorf("registerVisitorFace error writing to frontend: %w", err)
	}
	return nil
}

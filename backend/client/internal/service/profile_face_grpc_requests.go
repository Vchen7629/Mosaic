package service

import (
	"context"
	"encoding/base64"
	"fmt"

	fd "mosaic-client.com/gen/face_detection"
)

type ProfileSyncRes struct {
	Type         string `json:"type"`
	SessionToken string `json:"session_token"`
}

// Process an array of face frames for profile sync
// matches against existing profiles or registers a new one
func SyncProfile(
	frames []string,
	conn *SafeConn,
	client fd.FaceDetectionServiceClient,
) error {
	ctx := context.Background()

	faceBytes := make([][]byte, 0, len(frames))
	for _, frameData := range frames {
		// Decode base64
		decoded, err := base64.StdEncoding.DecodeString(frameData)
		if err != nil {
			return fmt.Errorf("frame decode error: %w", err)
		}

		faceBytes = append(faceBytes, decoded)
	}

	resp, err := client.SyncProfile(ctx, &fd.SyncProfileRequest{FaceBytes: faceBytes})
	if err != nil {
		return fmt.Errorf("sync gRPC error: %w", err)
	}

	if !resp.FaceDetected {
		return nil
	}

	if !resp.NewFace {
		err = conn.WriteJSON(ProfileSyncRes{Type: "profile_face_response", SessionToken: resp.SessionToken})
		if err != nil {
			return fmt.Errorf("error sending face same as profile face res to frontend: %w", err)
		}
		return nil
	}

	regResp, err := client.RegisterProfileFace(ctx, &fd.RegisterProfileFaceRequest{
		FaceEmbedding: resp.FaceEmbedding,
	})
	if err != nil {
		return fmt.Errorf("RegisterProfileFace gRPC error: %w", err)
	}

	err = conn.WriteJSON(ProfileSyncRes{Type: "profile_face_response", SessionToken: regResp.SessionToken})
	if err != nil {
		return fmt.Errorf("syncProfile error writing to frontend: %w", err)
	}
	return nil
}

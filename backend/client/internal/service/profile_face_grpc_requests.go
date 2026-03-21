package service

import (
	"context"
	"encoding/base64"
	"fmt"

	fd "mosaic-client.com/gen/face_detection"
)


type ProfileSyncRes struct {
	Type 			string `json:"type"`
	ProfileId		int32 `json:"profile_id"`
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
		return fmt.Errorf("Sync gRPC error: %w", err)
	}

	if !resp.FaceDetected {
		return nil
	}

	if !resp.NewFace {
		conn.WriteJSON(ProfileSyncRes{Type: "profile_face_response", ProfileId: resp.ProfileId})
		return nil
	}

	regResp, err := client.RegisterProfileFace(ctx, &fd.RegisterProfileFaceRequest{
		FaceEmbedding: resp.FaceEmbedding,
	})
	if err != nil {
		return fmt.Errorf("RegisterProfileFace gRPC error: %w", err)
	}

	conn.WriteJSON(ProfileSyncRes{Type: "profile_face_response", ProfileId: regResp.ProfileId})
	return nil
}

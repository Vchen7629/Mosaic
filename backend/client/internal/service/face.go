package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"strconv"
	fd "mosaic-client.com/gen/face_detection"
)

type UnknownVisitorResponse struct {
	Type			string `json:"type"`
	FaceEmbedding 	[]float32 `json:"face_embedding"`
}

type KnownVisitorResponse struct {
	Type			string `json:"type"`
	VisitorName 	string `json:"visitor_name"`
	Briefing		string `json:"briefing"`
}

// Process images for potential visitor
func ProcessVisitorImage(
	frameData string, 
	patientID string, 
	conn *SafeConn,
	client fd.FaceDetectionServiceClient,
) error {
	ctx := context.Background()
	// Decode base64
	faceBytes, err := base64.StdEncoding.DecodeString(frameData)
	if err != nil {
		return fmt.Errorf("Frame decode error: %w", err)
	}

	id64, err := strconv.ParseInt(patientID, 10, 32)
	if err != nil {
		return fmt.Errorf("Error converting string to int64: %w", err)
	}

	//log.Printf("[ProcessFace] Recieved face_data: %s", frameData)

	// Compress JPEG frame for better performance

	// Send to face detection service via gRPC
	resp, err := client.ProcessVisitorFaces(ctx, &fd.ProcessVisitorFacesRequest{
		FaceBytes: faceBytes,
		PatientId: int32(id64),
	})
	if err != nil {
		return fmt.Errorf("ProcessVisitorFace gRPC error: %w", err)
	}

	if !resp.FaceDetected {
		log.Println("ProcessVisitorFace no faces detected")
		return nil
	}

	if len(resp.Faces) > 0 {
		for _, faceData := range resp.Faces {
			if !faceData.IsKnown {
				conn.WriteJSON(UnknownVisitorResponse{
					Type: "new_visitor_register",
					FaceEmbedding: faceData.FaceEmbedding,
				})
			} else {
				conn.WriteJSON(KnownVisitorResponse{
					Type: "existing_visitor_response",
					VisitorName: faceData.Name,
					Briefing: faceData.Briefing,
				})
			}
		}
	}

	return nil
}

// Register a new visitor face
func RegisterNewVisitorFace(
	faceEmbedding string, 
	patientID string, 
	visitorName string,
	conn *SafeConn,
	client fd.FaceDetectionServiceClient,
) error {
	ctx := context.Background()

	id64, err := strconv.ParseInt(patientID, 10, 32)
	if err != nil {
		return fmt.Errorf("Error converting string to int64: %w", err)
	}

	faceEmb, err := ParseEmbedding(faceEmbedding)
	if err != nil {
		return err
	}
	// Todo: create db functions and handler function to save new visitor face embeddings to db
	// on face_detection_test service
	// Send to face detection service via gRPC
	resp, err := client.RegisterVisitorFace(ctx, &fd.RegisterVisitorFaceRequest{
		FaceEmbedding: faceEmb,
		PatientId: int32(id64),
		VisitorName: visitorName,
	})

	if err != nil {
		return fmt.Errorf("RegisterVisitorFace gRPC error: %w", err)
	}

	// Todo: Implement retry with exponential backoff
	if !resp.Success {
		return nil
	}

	// Could add a return value so frontend can display success
	// banner?
	return nil
}

type ProfileFaceRes struct {
	Type 			string `json:"type"`
	FaceDetected	bool `json:"face_detected"`
	Success			bool `json:"success"`
	PatientID		int32 `json:"patient_id"`
}

// Process images for loading user patient profile
// handles registration internally if it detects a new face
func ProcessProfileImage(
	frameData string, 
	conn *SafeConn,
	client fd.FaceDetectionServiceClient,
) error {
	ctx := context.Background()
	// Decode base64
	faceBytes, err := base64.StdEncoding.DecodeString(frameData)
	if err != nil {
		return fmt.Errorf("Frame decode error: %w", err)
	}

	resp, err := client.ProcessProfileFace(ctx, &fd.ProcessProfileFaceRequest{FaceBytes: faceBytes})
	if err != nil {
		return fmt.Errorf("ProcessProfileFace gRPC error: %w", err)
	}

	if !resp.FaceDetected {
		conn.WriteJSON(ProfileFaceRes{Type: "profile_face_response", FaceDetected: false})
		return nil
	}

	patientID := resp.PatientId

	// face isnt a registered profile
	if resp.NewFace {
		regResp, err := client.RegisterProfileFace(ctx, &fd.RegisterProfileFaceRequest{FaceEmbedding: resp.FaceEmbedding})
		if err != nil {
			return fmt.Errorf("RegisterProfileFace gRPC error: %w", err)
		}
		patientID = regResp.PatientId
		conn.WriteJSON(ProfileFaceRes{
			Type:			"profile_face_response",
			FaceDetected: 	true,
			Success: 		regResp.Success,
			PatientID: 		patientID,
		})
		return nil
	}

	conn.WriteJSON(ProfileFaceRes{
		Type:			"profile_face_response",
		FaceDetected: 	true,
		Success: 		resp.Success,
		PatientID: 		patientID,
	})
	return nil
}

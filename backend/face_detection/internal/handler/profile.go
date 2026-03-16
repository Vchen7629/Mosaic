package handler

import (
	"context"
	"fmt"
	"log"

	"github.com/Kagami/go-face"
	fd "mosaic-face-detection.com/gen"
	"mosaic-face-detection.com/internal/db"
	"mosaic-face-detection.com/internal/service"
)

// Handler to process faces for syncing user profile
func (s *FaceDetectionServer) SyncProfile(
	ctx context.Context, 
	req *fd.SyncProfileRequest,
) (*fd.SyncProfileResponse, error) {	
	var allEmbeddings []face.Descriptor
	for _, frameBytes := range req.FaceBytes {
		embeddings, err := service.GenerateFaceEmbeddings(s.rec, frameBytes)
		if err != nil {
			return nil, err
		}	
		allEmbeddings = append(allEmbeddings, embeddings...)
	}

	// return early if no faces in frame
	if len(allEmbeddings) == 0 {
		log.Println("GRPC profile no faces detected in any frame")
		return &fd.SyncProfileResponse{FaceDetected: false}, nil
	}

	var knownProfileFaceEmbs []service.Faces
	err := db.RetryWithBackoff(ctx, db.DefaultRetryConfig(), func() error {
		var err error
		knownProfileFaceEmbs, err = s.pool.FetchAllProfileFaceEmb()
		return err
	})
	if err != nil {
		log.Printf("GRPC user profile error fetching from db: %v", err)
		return &fd.SyncProfileResponse{Success: false}, err
	}

	matchingProfileID, matched := service.CompareProfileFaces(s.rec, allEmbeddings, knownProfileFaceEmbs)

	if !matched {
		log.Println("GRPC profile no faces matched!, returning embeddings for registration")
		faceEmbeddings := make([]*fd.FaceEmbedding, len(allEmbeddings))
		for i, emb := range allEmbeddings {
			faceEmbeddings[i] = &fd.FaceEmbedding{FaceEmbedding: emb[:]}
		}
		return &fd.SyncProfileResponse{
			FaceDetected: true,
			Success: true,
			NewFace: true,
			FaceEmbedding: faceEmbeddings,
		}, nil
	}

	return &fd.SyncProfileResponse{
		FaceDetected: true,
		PatientId: matchingProfileID,
		Success: true,
	}, nil
}


// Handler for when the face embedding doesnt 
// match with an existing embedding for users
func (s*FaceDetectionServer) RegisterProfileFace(
	ctx context.Context,
	req *fd.RegisterProfileFaceRequest,
) (*fd.RegisterProfileFaceResponse, error) {
	if len(req.FaceEmbedding) == 0 {
		return nil, fmt.Errorf("no face embeddings provided")
	}

	embeddings := make([]face.Descriptor, len(req.FaceEmbedding))
	for i, e := range req.FaceEmbedding {
		err := service.ValidateEmbeddingSlice(e.FaceEmbedding)
		if err != nil {
			return nil, err
		}
		copy(embeddings[i][:], e.FaceEmbedding)
	}

	var profileID *int32
	err := db.RetryWithBackoff(ctx, db.DefaultRetryConfig(), func() error {
		var err error
		profileID, err = s.pool.AddNewFaceForUser(embeddings)
		return err
	})
	if err != nil {
		return &fd.RegisterProfileFaceResponse{Success: false}, nil
	}

	return &fd.RegisterProfileFaceResponse{PatientId: *profileID, Success: true}, nil
}

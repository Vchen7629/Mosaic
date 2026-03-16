package handler

import (
	"context"
	"log"

	"github.com/Kagami/go-face"
	fd "mosaic-face-detection.com/gen"
	"mosaic-face-detection.com/internal/db"
	"mosaic-face-detection.com/internal/service"
)

// Handler to process faces for visitors
func (s *FaceDetectionServer) ProcessVisitorFaces(
	ctx context.Context, 
	req *fd.ProcessVisitorFacesRequest,
) (*fd.ProcessVisitorFacesResponse, error) {	
	embeddings, err := service.GenerateFaceEmbeddings(s.rec, req.FaceBytes)
	if err != nil {
		return nil, err
	}

	// return early if no faces in frame
	if len(embeddings) == 0 {
		return &fd.ProcessVisitorFacesResponse{FaceDetected: false}, nil
	}

	var knownVisitors []service.Faces
	err = db.RetryWithBackoff(ctx, db.DefaultRetryConfig(), func() error {
		var err error
		knownVisitors, err = s.pool.FetchAllVisitorFaceEmbForPatient(req.PatientId)
		return err
	})
	if err != nil {
		return &fd.ProcessVisitorFacesResponse{ Success: false }, err
	}

	matchingFaceRes := service.CompareVisitorFaces(s.rec, embeddings, knownVisitors)

	faceResults := make([]*fd.FaceResult, len(embeddings))
	for i, visitorID := range matchingFaceRes {
		if visitorID == -1 { // non matching face case
			faceResults[i] = &fd.FaceResult{
				IsKnown: false,
				FaceEmbedding: embeddings[i][:], // [128]float32 to []float32
			}
		} else {
			var briefing string
			err := db.RetryWithBackoff(ctx, db.DefaultRetryConfig(), func() error {
				var err error
				briefing, err = s.pool.FetchVisitorBriefing(req.PatientId, visitorID)
				return err
			})
			if err != nil {
				log.Printf("Failed to fetch briefing for visitor %d: %v", visitorID, err)
				briefing = "" // continuing with empty briefing instead of failing entire req
			}
			
			faceResults[i] = &fd.FaceResult{
				IsKnown: true,
				Briefing: briefing,
			}
		}
	}

	return &fd.ProcessVisitorFacesResponse{
		FaceDetected: true,
		Faces: faceResults,
		Success: true,
	}, nil
}

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

	log.Printf("GRPC generated %d embeddings from %d frames!", len(allEmbeddings), len(req.FaceBytes))

	// return early if no faces in frame
	if len(allEmbeddings) == 0 {
		log.Println("GRPC profile no faces detected in any frame")
		return &fd.SyncProfileResponse{FaceDetected: false}, nil
	}

	var knownProfileFaceEmbs []service.Faces
	err := db.RetryWithBackoff(ctx, db.DefaultRetryConfig(), func() error {
		var err error
		knownProfileFaceEmbs, err = s.pool.FetchAllProfileFaceEmb()
		log.Println("GRPC Called db once to fetch all embeddings")
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
// match with an existing embedding for a visitor
func (s*FaceDetectionServer) RegisterVisitorFace(
	ctx context.Context,
	req *fd.RegisterVisitorFaceRequest,
) (*fd.RegisterVisitorFaceResponse, error) {
	err := service.ValidateEmbeddingSlice(req.FaceEmbedding)
	if err != nil {
		return nil, err
	}

	// converting []float32 to face.Descriptor [128]float32
	var embedding face.Descriptor
	copy(embedding[:], req.FaceEmbedding)

	err = db.RetryWithBackoff(ctx, db.DefaultRetryConfig(), func() error {
		err := s.pool.AddNewFaceForVisitor(req.PatientId, req.VisitorName, embedding)
		return err
	})
	if err != nil {
		return &fd.RegisterVisitorFaceResponse{Success: false}, nil
	}

	return &fd.RegisterVisitorFaceResponse{Success: true}, nil
}

// Handler for when the face embedding doesnt 
// match with an existing embedding for users
func (s*FaceDetectionServer) RegisterProfileFace(
	ctx context.Context,
	req *fd.RegisterProfileFaceRequest,
) (*fd.RegisterProfileFaceResponse, error) {
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

package handler

import (
	"context"
	"log"

	"github.com/Kagami/go-face"
	fd "mosaic-face-detection.com/gen"
	"mosaic-face-detection.com/internal/db"
	"mosaic-face-detection.com/internal/service"
)

// Handler to process face pipeline that
// does entire processing pipeline
func (s *FaceDetectionServer) ProcessFaces(
	ctx context.Context, 
	req *fd.ProcessFacesRequest,
) (*fd.ProcessFacesResponse, error) {	
	embeddings, err := service.GenerateFaceEmbeddings(s.rec, req.FaceBytes)
	if err != nil {
		return nil, err
	}

	// return early if no faces in frame
	if len(embeddings) == 0 {
		return &fd.ProcessFacesResponse{FaceDetected: false}, nil
	}

	var knownVisitors []service.KnownVisitor
	err = db.RetryWithBackoff(ctx, db.DefaultRetryConfig(), func() error {
		var err error
		knownVisitors, err = s.pool.FetchAllVisitorFaceEmbForPatient(req.PatientId)
		return err
	})
	if err != nil {
		return &fd.ProcessFacesResponse{ Success: false }, err
	}

	matchingFaceRes := service.CompareFaces(s.rec, embeddings, knownVisitors)

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

	return &fd.ProcessFacesResponse{
		FaceDetected: true,
		Faces: faceResults,
		Success: true,
	}, nil
}

// Handler for when the face embedding doesnt 
// match with an existing embedding
func (s*FaceDetectionServer) RegisterFace(
	ctx context.Context,
	req *fd.RegisterFaceRequest,
) (*fd.RegisterFaceResponse, error) {
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
		return &fd.RegisterFaceResponse{Success: false}, nil
	}

	return &fd.RegisterFaceResponse{Success: true}, nil
}
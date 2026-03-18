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
	rec := s.recPool.Acquire() // acquiring one instance of the rec model from pool
	defer s.recPool.Release(rec)

	embeddings, err := service.GenerateFaceEmbeddings(rec, req.FaceBytes)
	if err != nil || len(embeddings) == 0 {
		return &fd.ProcessVisitorFacesResponse{FaceDetected: false}, nil
	}

	var knownVisitors []service.VisitorFaces
	if req.ProfileId > 0 {
		err = db.RetryWithBackoff(ctx, db.DefaultRetryConfig(), func() error {
			var err error
			knownVisitors, err = s.pool.FetchAllVisitorData(req.ProfileId)
			return err
		})
		if err != nil {
			return &fd.ProcessVisitorFacesResponse{Success: false}, err
		}
	}

	matchingFaceRes := service.CompareVisitorFaces(rec, embeddings, knownVisitors)

	faceResults := make([]*fd.FaceResult, len(embeddings))
	for i, match := range matchingFaceRes {
		if match.ID == -1 { // non matching face case
			faceResults[i] = &fd.FaceResult{
				IsKnown:       false,
				FaceEmbedding: embeddings[i][:], // [128]float32 to []float32
			}
		} else {
			var briefing string
			err := db.RetryWithBackoff(ctx, db.DefaultRetryConfig(), func() error {
				var err error
				briefing, err = s.pool.FetchVisitorBriefing(req.ProfileId, match.ID)
				return err
			})
			if err != nil {
				log.Printf("Failed to fetch briefing for visitor %d: %v", match.ID, err)
				briefing = "" // continuing with empty briefing instead of failing entire req
			}

			faceResults[i] = &fd.FaceResult{
				IsKnown:  	true,
				Briefing: 	briefing,
				VisitorId: 	match.ID,
				Name: 		match.Name,
			}
		}
	}

	return &fd.ProcessVisitorFacesResponse{
		FaceDetected: true,
		Faces:        faceResults,
		Success:      true,
	}, nil
}

// Handler for when the face embedding doesnt
// match with an existing embedding for a visitor
func (s *FaceDetectionServer) RegisterVisitorFace(
	ctx context.Context,
	req *fd.RegisterVisitorFaceRequest,
) (*fd.RegisterVisitorFaceResponse, error) {
	err := service.ValidateEmbeddingSlice(req.FaceEmbedding)
	if err != nil {
		return nil, err
	}

	log.Printf(
		"Recieved one face embedding of size %d with name %s patientID %d", 
		len(req.FaceEmbedding), req.VisitorName, req.ProfileId)
	// converting []float32 to face.Descriptor [128]float32
	var embedding face.Descriptor
	copy(embedding[:], req.FaceEmbedding)

	var visitorID *int32
	err = db.RetryWithBackoff(ctx, db.DefaultRetryConfig(), func() error {
		visitorID, err = s.pool.AddNewFaceForVisitor(req.ProfileId, req.VisitorName, embedding)
		return err
	})
	if err != nil {
		return &fd.RegisterVisitorFaceResponse{Success: false}, nil
	}

	return &fd.RegisterVisitorFaceResponse{Success: true, VisitorId: *visitorID}, nil
}

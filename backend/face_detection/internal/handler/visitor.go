package handler

import (
	"context"

	"github.com/Kagami/go-face"
	fd "mosaic-face-detection.com/gen"
	"mosaic-face-detection.com/internal/service"
)

// Handler to process faces for visitors
func (s *FaceDetectionServer) ProcessVisitorFaces(
	ctx context.Context,
	req *fd.ProcessVisitorFacesRequest,
) (*fd.ProcessVisitorFacesResponse, error) {
	rec := s.recPool.Acquire() // acquiring one instance of the rec model from pool
	defer s.recPool.Release(rec)

	s.logger.Debug(
		"Processing visitor faces while recording", 
		"profile_id", req.ProfileId,
		"face_byte_size", len(req.FaceBytes),
	)

	embeddings, err := service.GenerateFaceEmbeddings(rec, req.FaceBytes)
	if err != nil {
		s.logger.Error("error generating face embeddings", "err", err)
		return &fd.ProcessVisitorFacesResponse{FaceDetected: false}, nil
	}

	if len(embeddings) == 0 {
		s.logger.Debug("embeddings generated has size 0")
		return &fd.ProcessVisitorFacesResponse{FaceDetected: false}, nil
	}

	currentProfileEmbs, err := retryDB(s.logger, ctx, func() ([]service.ProfileFaces, error) {
		return s.pool.FetchProfileFaceEmbForID(req.ProfileId)
	})
	if err != nil {
		s.logger.Error("error fetching profile face embedding", "err", err)
		return &fd.ProcessVisitorFacesResponse{Success: false}, err
	}

	_, matched := service.CompareProfileFaces(rec, []face.Descriptor{embeddings[0]}, currentProfileEmbs)
	if matched {
		s.logger.Debug("process visitor face, matched face")
		return &fd.ProcessVisitorFacesResponse{FaceDetected: true, NonVisitorFace: true}, nil
	}

	var knownVisitors []service.VisitorFaces
	if req.ProfileId > 0 {
		knownVisitors, err = retryDB(s.logger, ctx, func() ([]service.VisitorFaces, error) {
			return s.pool.FetchAllVisitorData(req.ProfileId)
		})
		if err != nil {
			s.logger.Error("error fetching visitor data", "err", err)
			return &fd.ProcessVisitorFacesResponse{Success: false}, err
		}
	}

	matchingFaceRes := service.CompareVisitorFaces(rec, embeddings, knownVisitors)

	faceResults := make([]*fd.FaceResult, len(embeddings))
	for i, match := range matchingFaceRes {
		if match.ID == -1 { // non matching face case
			s.logger.Debug(
				"visitor face doesnt match with a visitor in the db for profile", 
				"profile_id", req.ProfileId,
				"visitor_name", match.Name,
			)
			faceResults[i] = &fd.FaceResult{
				IsKnown:       false,
				FaceEmbedding: embeddings[i][:], // [128]float32 to []float32
			}
		} else {
			briefing, err := retryDB(s.logger, ctx, func() (string, error) {
				return s.pool.FetchVisitorBriefing(req.ProfileId, match.ID)
			})
			if err != nil {
				s.logger.Error(
					"Failed to fetch briefing for visitor", 
					"profile_id", req.ProfileId,
					"visitor_id", match.ID,
					"err", err,
				)
				briefing = "" // continuing with empty briefing instead of failing entire req
			}

			faceResults[i] = &fd.FaceResult{
				IsKnown:   true,
				Briefing:  briefing,
				VisitorId: match.ID,
				Name:      match.Name,
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

	s.logger.Debug(
		"Recieved one face embedding of size", 
		"name", req.VisitorName, 
		"profile_id", req.ProfileId, 
		"embedding_len", len(req.FaceEmbedding),
	)
	// converting []float32 to face.Descriptor [128]float32
	var embedding face.Descriptor
	copy(embedding[:], req.FaceEmbedding)

	visitorID, err := retryDB(s.logger, ctx, func() (*int32, error) {
		return s.pool.AddNewFaceForVisitor(req.ProfileId, req.VisitorName, embedding)
	})
	if err != nil {
		s.logger.Error("error adding new face for visitor", "profile_id", req.ProfileId, "err", err)
		return &fd.RegisterVisitorFaceResponse{Success: false}, nil
	}

	return &fd.RegisterVisitorFaceResponse{Success: true, VisitorId: *visitorID}, nil
}

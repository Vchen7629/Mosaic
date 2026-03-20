package handler

import (
	"context"
	"time"

	"github.com/Kagami/go-face"
	fd "mosaic-face-detection.com/gen"
	"mosaic-face-detection.com/internal/observability"
	"mosaic-face-detection.com/internal/service"
	"mosaic-face-detection.com/internal/db"
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
	
	embGenStart := time.Now()
	embeddings, err := service.GenerateFaceEmbeddings(rec, req.FaceBytes)
	observability.EmbeddingGenerationDuration.Observe(float64(time.Since(embGenStart).Milliseconds()))
	if err != nil {
		observability.ErrorsTotal.WithLabelValues("generate_embeddings").Inc()
		s.logger.Error("error generating face embeddings", "err", err)
		return &fd.ProcessVisitorFacesResponse{FaceDetected: false}, nil
	}

	if len(embeddings) == 0 {
		s.logger.Debug("embeddings generated has size 0")
		return &fd.ProcessVisitorFacesResponse{FaceDetected: false}, nil
	}

	profileFetchStart := time.Now()
	currentProfileEmbs, err := db.RetryDB(s.logger, ctx, func() ([]service.ProfileFaces, error) {
		return s.pool.FetchProfileFaceEmbForID(req.ProfileId)
	})
	observability.ProfileEmbFetchDuration.Observe(float64(time.Since(profileFetchStart).Milliseconds()))
	if err != nil {
		observability.ErrorsTotal.WithLabelValues("fetch_profile_embeddings").Inc()
		s.logger.Error("error fetching profile face embedding", "err", err)
		return &fd.ProcessVisitorFacesResponse{Success: false}, err
	}

	profileCmbStart := time.Now()
	_, matched := service.CompareProfileFaces(rec, []face.Descriptor{embeddings[0]}, currentProfileEmbs)
	observability.ProfileComparisonDuration.Observe(float64(time.Since(profileCmbStart).Milliseconds()))
	if matched {
		s.logger.Debug("process visitor face, matched face")
		return &fd.ProcessVisitorFacesResponse{FaceDetected: true, NonVisitorFace: true}, nil
	}

	var knownVisitors []service.VisitorFaces
	if req.ProfileId > 0 {
		visitorFetchStart := time.Now()
		knownVisitors, err = db.RetryDB(s.logger, ctx, func() ([]service.VisitorFaces, error) {
			return s.pool.FetchAllVisitorData(req.ProfileId)
		})
		observability.VisitorEmbFetchDuration.Observe(float64(time.Since(visitorFetchStart).Milliseconds()))
		if err != nil {
			observability.ErrorsTotal.WithLabelValues("fetch_visitor_data").Inc()
			s.logger.Error("error fetching visitor data", "err", err)
			return &fd.ProcessVisitorFacesResponse{Success: false}, err
		}
	}

	visitorCmpStart := time.Now()
	matchingFaceRes := service.CompareVisitorFaces(rec, embeddings, knownVisitors)
	observability.VisitorComparisonDuration.Observe(float64(time.Since(visitorCmpStart).Milliseconds()))

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
			briefingFetchStart := time.Now()
			briefing, err := db.RetryDB(s.logger, ctx, func() (string, error) {
				return s.pool.FetchVisitorBriefing(req.ProfileId, match.ID)
			})
			observability.BriefingFetchDuration.Observe(float64(time.Since(briefingFetchStart).Milliseconds()))
			if err != nil {
				observability.ErrorsTotal.WithLabelValues("fetch_briefing").Inc()
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

	visitorRegisterStart := time.Now()
	visitorID, err := db.RetryDB(s.logger, ctx, func() (*int32, error) {
		return s.pool.AddNewFaceForVisitor(req.ProfileId, req.VisitorName, embedding)
	})
	observability.VisitorRegisterDuration.Observe(float64(time.Since(visitorRegisterStart).Milliseconds()))
	if err != nil {
		observability.ErrorsTotal.WithLabelValues("register_visitor").Inc()
		s.logger.Error("error adding new face for visitor", "profile_id", req.ProfileId, "err", err)
		return &fd.RegisterVisitorFaceResponse{Success: false}, nil
	}

	return &fd.RegisterVisitorFaceResponse{Success: true, VisitorId: *visitorID}, nil
}

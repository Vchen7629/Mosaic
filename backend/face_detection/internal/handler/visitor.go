package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/Kagami/go-face"
	fd "mosaic-face-detection.com/gen"
	"mosaic-face-detection.com/internal/cache"
	"mosaic-face-detection.com/internal/db"
	"mosaic-face-detection.com/internal/observability"
	"mosaic-face-detection.com/internal/service"
)

// Handler to process faces for visitors
func (s *FaceDetectionServer) ProcessVisitorFaces(
	ctx context.Context,
	req *fd.ProcessVisitorFacesRequest,
) (*fd.ProcessVisitorFacesResponse, error) {
	rec := s.recPool.Acquire() // acquiring one instance of the rec model from pool
	defer s.recPool.Release(rec)

	s.logger.Debug("Processing visitor faces while recording", "session_token", req.SessionToken, "face_byte_size", len(req.FaceBytes))

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

	profileID, err := cache.FetchProfileIDFromCache(ctx, s.cache, req.SessionToken)
	if err != nil {
		profileIDPtr, dbErr := db.RetryDB(s.logger, ctx, func() (*int32, error) {
			return s.pool.FetchProfileIDWithSession(req.SessionToken)
		})
		if dbErr != nil {
			observability.ErrorsTotal.WithLabelValues("fetch_profile_id_with_session").Inc()
			s.logger.Error("session token not found", "session_token", req.SessionToken)
			return &fd.ProcessVisitorFacesResponse{FaceDetected: false}, fmt.Errorf("invalid session token")
		}
		profileID = profileIDPtr
	}

	currentProfileEmbs, err := cache.FetchProfileFaceEmbForIDWithCache(
		ctx, *profileID, s.cache, s.logger,
		func() ([]service.ProfileFaces, error) {
			return db.RetryDB(s.logger, ctx, func() ([]service.ProfileFaces, error) {
				return s.pool.FetchProfileFaceEmbForID(*profileID)
			})
		},
	)
	if err != nil {
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
	if *profileID > 0 {
		visitorFetchStart := time.Now()
		knownVisitors, err = cache.FetchAllVisitorDataWithCache(
			ctx, *profileID, s.cache, s.logger,
			func() ([]service.VisitorFaces, error) {
				return db.RetryDB(s.logger, ctx, func() ([]service.VisitorFaces, error) {
					return s.pool.FetchAllVisitorData(*profileID)
				})
			},
		)
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
		faceResults[i] = s.buildFaceResult(ctx, *profileID, match, embeddings[i])
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
		"session_token", req.SessionToken,
		"embedding_len", len(req.FaceEmbedding),
	)
	// converting []float32 to face.Descriptor [128]float32
	var embedding face.Descriptor
	copy(embedding[:], req.FaceEmbedding)

	profileID, err := cache.FetchProfileIDFromCache(ctx, s.cache, req.SessionToken)
	if err != nil {
		profileIDPtr, dbErr := db.RetryDB(s.logger, ctx, func() (*int32, error) {
			return s.pool.FetchProfileIDWithSession(req.SessionToken)
		})
		if dbErr != nil {
			observability.ErrorsTotal.WithLabelValues("fetch_profile_id_with_session").Inc()
			s.logger.Error("session token not found", "session_token", req.SessionToken)
			return &fd.RegisterVisitorFaceResponse{Success: false}, fmt.Errorf("invalid session token")
		}
		profileID = profileIDPtr
	}

	visitorRegisterStart := time.Now()
	visitorID, err := db.RetryDB(s.logger, ctx, func() (*int32, error) {
		return s.pool.AddNewFaceForVisitor(*profileID, req.VisitorName, embedding)
	})
	observability.VisitorRegisterDuration.Observe(float64(time.Since(visitorRegisterStart).Milliseconds()))
	if err != nil {
		observability.ErrorsTotal.WithLabelValues("register_visitor").Inc()
		s.logger.Error("error adding new face for visitor", "profile_id", profileID, "err", err)
		return &fd.RegisterVisitorFaceResponse{Success: false}, nil
	}

	newEntry := service.VisitorFaces{ID: *visitorID, Name: req.VisitorName, Embedding: embedding}
	cache.AppendToVisitorDataCache(ctx, *profileID, s.cache, newEntry)

	return &fd.RegisterVisitorFaceResponse{Success: true, VisitorId: *visitorID}, nil
}

// Builds the face result gRPC response with briefing text, visitor id and name
func (s *FaceDetectionServer) buildFaceResult(
	ctx context.Context,
	profileID int32,
	match service.VisitorMatch,
	embedding face.Descriptor,
) *fd.FaceResult {
	if match.ID == -1 { // non matching face case
		s.logger.Debug(
			"visitor face doesnt match with a visitor in the db for profile",
			"profile_id", profileID,
			"visitor_name", match.Name,
		)
		return &fd.FaceResult{
			IsKnown:       false,
			FaceEmbedding: embedding[:], // [128]float32 to []float32
		}
	}
	briefingFetchStart := time.Now()
	briefing, err := db.RetryDB(s.logger, ctx, func() (string, error) {
		return s.pool.FetchVisitorBriefing(profileID, match.ID)
	})
	observability.BriefingFetchDuration.Observe(float64(time.Since(briefingFetchStart).Milliseconds()))
	if err != nil {
		observability.ErrorsTotal.WithLabelValues("fetch_briefing").Inc()
		s.logger.Error("Failed to fetch briefing for visitor", "profile_id", profileID, "visitor_id", match.ID, "err", err)
		briefing = "" // continuing with empty briefing instead of failing entire req
	}

	return &fd.FaceResult{IsKnown: true, Briefing: briefing, VisitorId: match.ID, Name: match.Name}
}

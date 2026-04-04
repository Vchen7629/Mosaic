package handler

import (
	"context"
	"crypto/rand"
	"time"

	"github.com/Kagami/go-face"
	fd "mosaic-face-detection.com/gen"
	"mosaic-face-detection.com/internal/cache"
	"mosaic-face-detection.com/internal/db"
	"mosaic-face-detection.com/internal/observability"
	"mosaic-face-detection.com/internal/service"
)

// Handler to process faces for syncing user profile
func (s *FaceDetectionServer) SyncProfile(
	ctx context.Context,
	req *fd.SyncProfileRequest,
) (*fd.SyncProfileResponse, error) {
	rec, err := s.recPool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer s.recPool.Release(rec)

	s.logger.Debug("Syncing Profile using face bytes", "face_byte_size", len(req.FaceBytes))

	var allEmbeddings []face.Descriptor
	for _, frameBytes := range req.FaceBytes {
		embGenStart := time.Now()
		embeddings, err := service.GenerateFaceEmbeddings(rec, frameBytes)
		observability.EmbeddingGenerationDuration.Observe(float64(time.Since(embGenStart).Milliseconds()))
		if err != nil {
			observability.ErrorsTotal.WithLabelValues("generate_embeddings").Inc()
			s.logger.Error("failed to generate face embeddings using the profile face", "err", err)
			return nil, err
		}
		allEmbeddings = append(allEmbeddings, embeddings...)
	}

	// return early if no faces in frame
	if len(allEmbeddings) == 0 {
		s.logger.Debug("SyncProfile no faces detected in any frame")
		return &fd.SyncProfileResponse{FaceDetected: false}, nil
	}

	fetchProfileEmbStart := time.Now()
	knownProfileFaceEmbs, err := db.RetryDB(s.logger, ctx, func() ([]service.ProfileFaces, error) {
		return s.pool.FetchAllProfileFaceEmb()
	})
	observability.ProfileEmbFetchDuration.Observe(float64(time.Since(fetchProfileEmbStart).Milliseconds()))
	if err != nil {
		observability.ErrorsTotal.WithLabelValues("fetch_profile_embeddings").Inc()
		s.logger.Error("failed to fetch face embeddings for all profiles from db", "err", err)
		return &fd.SyncProfileResponse{Success: false}, err
	}

	profileCmpStart := time.Now()
	matchingProfileID, matched := service.CompareProfileFaces(rec, allEmbeddings, knownProfileFaceEmbs)
	observability.ProfileComparisonDuration.Observe(float64(time.Since(profileCmpStart).Milliseconds()))

	if !matched {
		s.logger.Debug("GRPC profile no faces matched!, registering new face embeddings in database")

		profileRegisterStart := time.Now()
		profileID, err := db.RetryDB(s.logger, ctx, func() (*int32, error) {
			return s.pool.AddNewFaceForProfile(allEmbeddings)
		})
		observability.ProfileRegisterDuration.Observe(float64(time.Since(profileRegisterStart).Milliseconds()))
		if err != nil {
			observability.ErrorsTotal.WithLabelValues("register_profile").Inc()
			s.logger.Error("failed to register new profile with face", "err", err)
			return &fd.SyncProfileResponse{Success: false}, err
		}

		sessionToken := rand.Text()

		err = cache.CreateNewProfileSession(ctx, *profileID, s.cache, sessionToken)
		if err != nil {
			s.logger.Warn("failed to store new session in cache", "err", err)
		}

		_, err = db.RetryDB(s.logger, ctx, func() (struct{}, error) {
			return struct{}{}, s.pool.UpsertSession(*profileID, sessionToken)
		})
		if err != nil {
			s.logger.Error("failed to store session in db", "err", err)
			return nil, err
		}

		return &fd.SyncProfileResponse{
			FaceDetected:  true,
			Success:       true,
			SessionToken: 	sessionToken,
		}, nil
	}

	sessionToken := rand.Text()

	err = cache.CreateNewProfileSession(ctx, matchingProfileID, s.cache, sessionToken)
	if err != nil {
		s.logger.Warn("failed to store session in cache", "err", err)
	}

	_, err = db.RetryDB(s.logger, ctx, func() (struct{}, error) {
		return struct{}{}, s.pool.UpsertSession(matchingProfileID, sessionToken)
	})
	if err != nil {
		s.logger.Error("failed to store session in db", "err", err)
		return nil, err
	}

	return &fd.SyncProfileResponse{
		FaceDetected: true,
		SessionToken: sessionToken,
		Success:      true,
	}, nil
}
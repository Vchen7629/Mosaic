package handler

import (
	"context"
	"fmt"

	"github.com/Kagami/go-face"
	fd "mosaic-face-detection.com/gen"
	"mosaic-face-detection.com/internal/service"
)

// Handler to process faces for syncing user profile
func (s *FaceDetectionServer) SyncProfile(
	ctx context.Context,
	req *fd.SyncProfileRequest,
) (*fd.SyncProfileResponse, error) {
	rec := s.recPool.Acquire() // acquiring one instance of the rec model from pool
	defer s.recPool.Release(rec)

	s.logger.Debug("Syncing Profile using face bytes", "face_byte_size", len(req.FaceBytes))


	var allEmbeddings []face.Descriptor
	for _, frameBytes := range req.FaceBytes {
		embeddings, err := service.GenerateFaceEmbeddings(rec, frameBytes)
		if err != nil {
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

	knownProfileFaceEmbs, err := retryDB(s.logger, ctx, func() ([]service.ProfileFaces, error) {
		return s.pool.FetchAllProfileFaceEmb()
	})
	if err != nil {
		s.logger.Error("failed to fetch face embeddings for all profiles from db", "err", err)
		return &fd.SyncProfileResponse{Success: false}, err
	}

	matchingProfileID, matched := service.CompareProfileFaces(rec, allEmbeddings, knownProfileFaceEmbs)

	if !matched {
		s.logger.Debug("GRPC profile no faces matched!, returning embeddings for registration")
		faceEmbeddings := make([]*fd.FaceEmbedding, len(allEmbeddings))
		for i, emb := range allEmbeddings {
			faceEmbeddings[i] = &fd.FaceEmbedding{FaceEmbedding: emb[:]}
		}
		return &fd.SyncProfileResponse{
			FaceDetected:  true,
			Success:       true,
			NewFace:       true,
			FaceEmbedding: faceEmbeddings,
		}, nil
	}

	return &fd.SyncProfileResponse{
		FaceDetected: true,
		ProfileId:    matchingProfileID,
		Success:      true,
	}, nil
}

// Handler for when the face embedding doesnt
// match with an existing embedding for users
func (s *FaceDetectionServer) RegisterProfileFace(
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

	profileID, err := retryDB(s.logger, ctx, func() (*int32, error) {
		return s.pool.AddNewFaceForUser(embeddings)
	})
	if err != nil {
		s.logger.Error("failed to add register new profile with face", "err", err)
		return &fd.RegisterProfileFaceResponse{Success: false}, nil
	}

	return &fd.RegisterProfileFaceResponse{ProfileId: *profileID, Success: true}, nil
}

package handler

import (
	"log/slog"

	"github.com/valkey-io/valkey-go"
	fd "mosaic-face-detection.com/gen"
	"mosaic-face-detection.com/internal/db"
	"mosaic-face-detection.com/internal/service"
)

type FaceDetectionServer struct {
	fd.UnimplementedFaceDetectionServiceServer
	logger  *slog.Logger
	recPool *service.RecognizerPool
	cache   valkey.Client
	pool    *db.DBPool
}

func NewFaceDetectionServer(
	logger *slog.Logger,
	recPool *service.RecognizerPool,
	cache valkey.Client,
	dbPool *db.DBPool,
) *FaceDetectionServer {
	return &FaceDetectionServer{
		logger:  logger,
		recPool: recPool,
		cache:   cache,
		pool:    dbPool,
	}
}

package handler

import (
	"log/slog"

	fd "mosaic-face-detection.com/gen"
	"mosaic-face-detection.com/internal/db"
	"mosaic-face-detection.com/internal/service"
)

type FaceDetectionServer struct {
	fd.UnimplementedFaceDetectionServiceServer
	logger 	*slog.Logger
	recPool *service.RecognizerPool
	pool    *db.DBPool
}

func NewFaceDetectionServer(
	logger *slog.Logger,
	recPool *service.RecognizerPool,
	dbPool *db.DBPool,
) *FaceDetectionServer {
	return &FaceDetectionServer{
		logger: logger,
		recPool: recPool,
		pool:    dbPool,
	}
}

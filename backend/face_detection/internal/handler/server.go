package handler

import (
	fd "mosaic-face-detection.com/gen"
	"mosaic-face-detection.com/internal/db"
	"mosaic-face-detection.com/internal/service"
)

type FaceDetectionServer struct {
	fd.UnimplementedFaceDetectionServiceServer
	recPool *service.RecognizerPool
	pool    *db.DBPool
}

func NewFaceDetectionServer(
	recPool *service.RecognizerPool,
	dbPool *db.DBPool,
) *FaceDetectionServer {
	return &FaceDetectionServer{
		recPool: recPool,
		pool:    dbPool,
	}
}

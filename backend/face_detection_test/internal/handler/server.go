package handler

import (
	"github.com/Kagami/go-face"
	fd "mosaic-face-detection.com/gen"
	"mosaic-face-detection.com/internal/db"
)

type FaceDetectionServer struct {
	fd.UnimplementedFaceDetectionServiceServer
	rec *face.Recognizer
	pool   *db.DBPool

}

func NewFaceDetectionServer(
	rec *face.Recognizer, 
	dbPool *db.DBPool,
) *FaceDetectionServer {
	return &FaceDetectionServer{
		rec: rec,
		pool: dbPool,
	}
}
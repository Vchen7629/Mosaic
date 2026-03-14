package service

import (
	"log"
	"path/filepath"

	"github.com/Kagami/go-face"
)

var modelsDir = filepath.Join("models")

// initializes the face detection model
func InitializeFaceDetector() (*face.recognizer, error){
	rec, err := face.NewRecognizer(modelsDir)
	if err != nil {
		return nil, log.Fatalf("Can't init face recognizer: %v", err)
	}

	return rec, nil
}

func GenerateFaceEmbeddings(faceBytes []byte) {
	
}
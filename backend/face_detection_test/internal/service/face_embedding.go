package service

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/Kagami/go-face"
)

var ModelsDir = filepath.Join("models")

// initializes the face detection model
func InitializeFaceDetector() (*face.Recognizer, error) {
	rec, err := face.NewRecognizer(ModelsDir)
	if err != nil {
		log.Fatalf("can't init face recognizer: %v", err)
	}

	return rec, nil
}

// takes raw image bytes, runs face detection, and returns
// the 128-d  descriptor for the first detected face
func GenerateFaceEmbeddings(
	rec *face.Recognizer, 
	faceBytes []byte,
) ([]face.Descriptor, error) {
	faces, err := rec.Recognize(faceBytes)
	if err != nil {
		return nil, fmt.Errorf("recognition failed: %w", err)
	}
	
	embeddings := make([]face.Descriptor, len(faces))
	for i, f := range faces {
		embeddings[i] = f.Descriptor
	}

	return embeddings, nil
}

type Faces struct {
	ID			int32
	Embedding 	face.Descriptor
}
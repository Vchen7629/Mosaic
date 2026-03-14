package service

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/Kagami/go-face"
)

var modelsDir = filepath.Join("models")

// initializes the face detection model
func InitializeFaceDetector() (*face.Recognizer, error) {
	rec, err := face.NewRecognizer(modelsDir)
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

// visitor from database
type KnownVisitor struct {
	ID			int32
	Embedding 	face.Descriptor
}

// Compare face embeddings of visitors
// saved in the database to the new face
// embeddings sent by the frontend
func CompareFaces(
	rec *face.Recognizer,
	embeddings []face.Descriptor,
	knownVisitors []KnownVisitor,
) map[int]int32 { // index of detected_face -> visitor_id (-1 if unknown)
	knownEmbeddings := make([]face.Descriptor, len(knownVisitors))
	knownIDs := make([]int32, len(knownVisitors))

 	for i, v := range knownVisitors {
		knownEmbeddings[i] = v.Embedding
		knownIDs[i] = v.ID
	}
	rec.SetSamples(knownEmbeddings, knownIDs)

	results := make(map[int]int32, len(embeddings))
	for i, embedding := range embeddings {
		results[i] = int32(rec.ClassifyThreshold(embedding, 0.6))
	}

	// returns dict with key: index of detected face in frame (0, 1, 2,..)
	// and value: visitor_id from db if matched, or -1 if unknown
	// example: face 0 and 2 are known: {0: 5, 1: -1, 2: 3}
	return results
}
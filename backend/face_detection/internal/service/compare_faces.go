package service

import (
	"github.com/Kagami/go-face"
)

// Compare face embeddings of visitors
// saved in the database to the new face
// embeddings sent by the frontend
func CompareVisitorFaces(
	rec *face.Recognizer,
	embeddings []face.Descriptor,
	knownVisitors []Faces,
) map[int]int32 { // index of detected_face -> visitor_id (-1 if unknown)
	knownEmbeddings := make([]face.Descriptor, len(knownVisitors))
	knownIDs := make([]int32, len(knownVisitors))

	for i, v := range knownVisitors {
		knownEmbeddings[i] = v.Embedding
		knownIDs[i] = v.ID
	}

	results := make(map[int]int32, len(embeddings))

	if len(knownVisitors) == 0 {
		for i := range embeddings {
			results[i] = -1
		}
		return results
	}

	rec.SetSamples(knownEmbeddings, knownIDs)

	for i, embedding := range embeddings {
		results[i] = int32(rec.ClassifyThreshold(embedding, 0.6))
	}

	// returns dict with key: index of detected face in frame (0, 1, 2,..)
	// and value: visitor_id from db if matched, or -1 if unknown
	// example: face 0 and 2 are known: {0: 5, 1: -1, 2: 3}
	return results
}

// Compares the client embedding against embeddings of all
// profiles fetched from the database
func CompareProfileFaces(
	rec *face.Recognizer,
	embedding []face.Descriptor,
	profileEmbeddings []Faces,
) (int32, bool) {
	if len(profileEmbeddings) == 0 || len(embedding) == 0 {
		return 0, false
	}

	knownEmbeddings := make([]face.Descriptor, len(profileEmbeddings))
	knownIDs := make([]int32, len(profileEmbeddings))
	for i, p := range profileEmbeddings {
		knownEmbeddings[i] = p.Embedding
		knownIDs[i] = p.ID
	}

	rec.SetSamples(knownEmbeddings, knownIDs)
	for _, emb := range embedding {
		matchedID := int32(rec.ClassifyThreshold(emb, 0.6))
		if matchedID != -1 {
			return matchedID, true
		}
	}

	return 0, false
}

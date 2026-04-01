package service

import (
	"github.com/Kagami/go-face"
)

type VisitorMatch struct {
	ID   int32
	Name string
}

// Compare face embeddings of visitors
// saved in the database to the new face
// embeddings sent by the frontend
func CompareVisitorFaces(
	rec *face.Recognizer,
	embeddings []face.Descriptor,
	knownVisitors []VisitorFaces,
) map[int]VisitorMatch { // index of detected_face -> VisitorMatch (-1 if unknown)
	knownEmbeddings := make([]face.Descriptor, len(knownVisitors))
	knownIDs := make([]int32, len(knownVisitors))
	nameByID := make(map[int32]string, len(knownVisitors))

	for i, v := range knownVisitors {
		knownEmbeddings[i] = v.Embedding
		knownIDs[i] = v.ID
		nameByID[v.ID] = v.Name
	}

	results := make(map[int]VisitorMatch, len(embeddings))

	if len(knownVisitors) == 0 {
		for i := range embeddings {
			results[i] = VisitorMatch{ID: -1}
		}
		return results
	}

	rec.SetSamples(knownEmbeddings, knownIDs)
	defer rec.SetSamples([]face.Descriptor{}, []int32{})

	for i, embedding := range embeddings {
		matchedID := int32(rec.ClassifyThreshold(embedding, 0.6))
		results[i] = VisitorMatch{ID: matchedID, Name: nameByID[matchedID]}
	}

	// returns dict with key: index of detected face in frame (0, 1, 2,..)
	// and value: VisitorMatch with visitor_id and name from db if matched,
	// or ID = -1 if unknown
	// example: face 0 and 2 are known: {0: 5, 1: -1, 2: 3}
	return results
}

// Compares the client embedding against embeddings of all
// profiles fetched from the database
func CompareProfileFaces(
	rec *face.Recognizer,
	embedding []face.Descriptor,
	profileEmbeddings []ProfileFaces,
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
	defer rec.SetSamples([]face.Descriptor{}, []int32{})
	for _, emb := range embedding {
		matchedID := int32(rec.ClassifyThreshold(emb, 0.6))
		if matchedID != -1 {
			return matchedID, true
		}
	}

	return 0, false
}

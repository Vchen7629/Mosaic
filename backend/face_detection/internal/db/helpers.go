package db

import (
	"fmt"

	"github.com/jackc/pgx/v4"
	"github.com/pgvector/pgvector-go"
	"mosaic-face-detection.com/internal/service"
)

// helper function that adds the face embeddings to a list
func scanFaceEmbeddings(rows pgx.Rows, errContext string) ([]service.Faces, error) {
	var result []service.Faces
	for rows.Next() {
		var f service.Faces
		var embVector pgvector.Vector

		if err := rows.Scan(&f.ID, &embVector); err != nil {
			return nil, fmt.Errorf("error fetching %s emb: %v", errContext, err)
		}

		copy(f.Embedding[:], embVector.Slice())
		result = append(result, f)
	}
	return result, nil
}

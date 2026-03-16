package test

import (
	"context"
	"testing"

	"github.com/Kagami/go-face"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pgvector/pgvector-go"
	"github.com/stretchr/testify/assert"
)


func CheckVisitorEmbeddings(
	t *testing.T, 
	pool *pgxpool.Pool, 
	patientID int32,
	visitor_name string,
) face.Descriptor {
	ctx := context.Background()

	query := `
		SELECT face_embedding
		FROM visitor_face_embeddings
		WHERE profile_id = $1
		AND visitor_name = $2
	`
	var embeddingRes pgvector.Vector
	err := pool.QueryRow(ctx, query, patientID, visitor_name).Scan(&embeddingRes)

	assert.Nil(t, err)

	// convert pgvector back to face.Descriptor since thats what we're comparing
	var embeddingFromDB face.Descriptor
	copy(embeddingFromDB[:], embeddingRes.Slice())

	return embeddingFromDB
}

// Create a visitor row in the database and return id of the newly created row
func SeedVisitor(
	t *testing.T, 
	pool *pgxpool.Pool, 
	patientID int32,
	visitorName string,
	faceEmbedding []float32,
) int32 {
	t.Helper()

	ctx := context.Background()
	var visitorID int32

	err := pool.QueryRow(ctx,
		`INSERT INTO visitor_face_embeddings 
		(profile_id, visitor_name, face_embedding) 
		VALUES ($1, $2, $3) 
		RETURNING id`,
		patientID, visitorName, pgvector.NewVector(faceEmbedding),
	).Scan(&visitorID)

	if err != nil {
		t.Fatalf("Failed to seed visitor: %v", err)
	}

	return visitorID
}
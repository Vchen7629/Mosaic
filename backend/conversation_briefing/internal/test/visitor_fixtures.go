package test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pgvector/pgvector-go"
	"github.com/stretchr/testify/assert"
)

// AddNewVisitor generates an embedding from imgBytes, seeds a profile and a visitor
// under that profile, and returns (profileID, visitorID).
func AddNewVisitor(
	t *testing.T,
	imgBytes []byte,
	testDB *TestDBContainer,
) (int32, int32) {
	t.Helper()
	embeddings := MakeEmbedding(0.2, 128)
	assert.Len(t, embeddings, 1)

	profileID := SeedProfile(t, testDB.Pool, embeddings)
	visitorID := SeedVisitor(t, testDB.Pool, profileID, "test_visitor", embeddings)

	return profileID, visitorID
}

// Create a visitor row in the database and return id of the newly created row
func SeedVisitor(
	t *testing.T, 
	pool *pgxpool.Pool, 
	profileID int32,
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
		profileID, visitorName, pgvector.NewVector(faceEmbedding),
	).Scan(&visitorID)

	if err != nil {
		t.Fatalf("Failed to seed visitor: %v", err)
	}

	return visitorID
}

package test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pgvector/pgvector-go"
	"github.com/stretchr/testify/assert"
)

func AddNewProfile(
	t *testing.T, 
	imgBytes []byte,
	testDB *TestDBContainer,
) int32 {
	embeddings := MakeEmbedding(0.2, 128)
	assert.Len(t, embeddings, 1)

	expectedID := SeedProfile(t, testDB.Pool, embeddings)

	return expectedID
}

// Create a profile row and its face embedding, return the profile id
func SeedProfile(t *testing.T, pool *pgxpool.Pool, faceEmbedding []float32) int32 {
	t.Helper()

	ctx := context.Background()
	var profileID int32

	err := pool.QueryRow(
		ctx,
		`INSERT INTO profiles DEFAULT VALUES RETURNING id`,
	).Scan(&profileID)
	if err != nil {
		t.Fatalf("Failed to seed profile: %v", err)
	}

	_, err = pool.Exec(
		ctx,
		`INSERT INTO profile_face_embeddings (profile_id, face_embedding) VALUES ($1, $2)`,
		profileID, pgvector.NewVector(faceEmbedding),
	)
	if err != nil {
		t.Fatalf("Failed to seed profile face embedding: %v", err)
	}

	return profileID
}

// creates float32 array embeddings
func MakeEmbedding(value float32, size int) []float32 {
	emb := make([]float32, size)
	for i := range emb {
		emb[i] = value
	}
	return emb
}
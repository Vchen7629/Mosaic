package test

import (
	"context"
	"testing"

	"github.com/Kagami/go-face"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pgvector/pgvector-go"
	"github.com/stretchr/testify/assert"
	"mosaic-face-detection.com/internal/service"
)

func AddNewProfile(
	t *testing.T, 
	rec *face.Recognizer, 
	imgBytes []byte,
	testDB *TestDBContainer,
) int32 {
	embeddings, err := service.GenerateFaceEmbeddings(rec, imgBytes)
	assert.NoError(t, err)
	assert.Len(t, embeddings, 1)

	embFloat32 := make([]float32, len(embeddings[0]))
	for i, v := range embeddings[0] {
		embFloat32[i] = v
	}
	expectedID := SeedProfile(t, testDB.Pool, embFloat32)

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


func CheckProfileEmbeddings(
	t *testing.T,
	pool *pgxpool.Pool,
	profileID int32,
) face.Descriptor {
	ctx := context.Background()

	query := `SELECT face_embedding FROM profile_face_embeddings WHERE profile_id = $1`
	var embeddingRes pgvector.Vector
	err := pool.QueryRow(ctx, query, profileID).Scan(&embeddingRes)

	assert.Nil(t, err)

	// convert pgvector back to face.Descriptor since thats what we're comparing
	var embeddingFromDB face.Descriptor
	copy(embeddingFromDB[:], embeddingRes.Slice())

	return embeddingFromDB
}

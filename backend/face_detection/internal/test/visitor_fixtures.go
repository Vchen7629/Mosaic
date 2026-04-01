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

// AddNewVisitor generates an embedding from imgBytes, seeds a profile and a visitor
// under that profile, and returns (profileID, visitorID).
func AddNewVisitor(
	t *testing.T,
	recPool *service.RecognizerPool,
	imgBytes []byte,
	testDB *TestDBContainer,
) (int32, int32) {
	t.Helper()

	rec, _ := recPool.Acquire(context.Background())
	defer recPool.Release(rec)

	embeddings, err := service.GenerateFaceEmbeddings(rec, imgBytes)
	assert.NoError(t, err)
	assert.Len(t, embeddings, 1)

	embFloat32 := make([]float32, len(embeddings[0]))
	copy(embFloat32, embeddings[0][:])

	profileID := SeedProfile(t, testDB.Pool, embFloat32)
	visitorID := SeedVisitor(t, testDB.Pool, profileID, "test_visitor", embFloat32)

	return profileID, visitorID
}

func CheckVisitorEmbeddings(
	t *testing.T,
	pool *pgxpool.Pool,
	profileID int32,
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
	err := pool.QueryRow(ctx, query, profileID, visitor_name).Scan(&embeddingRes)

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

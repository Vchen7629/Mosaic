package test

import (
	"context"
	"testing"
	"time"

	"github.com/Kagami/go-face"
	"github.com/jackc/pgx/v4/pgxpool"
	pgvector "github.com/pgvector/pgvector-go"
	"github.com/stretchr/testify/assert"
)

type ProductSourceConfig struct {                                                                        
	ProductID         int                                                                                  
	Platform          string                                                                               
	PlatformProductID string                                                                               
	URL               string                                                                               
} 

type PriceSnapshotConfig struct {
	ProductSourceID int
	Price			float64
	Currency 		string
	InStock			bool
	CheckedAt		*time.Time
}

// Create a profile row and its face embedding, return the profile id
func SeedPatient(t *testing.T, pool *pgxpool.Pool, faceEmbedding []float32) int32 {
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
		(patient_id, visitor_name, face_embedding) 
		VALUES ($1, $2, $3) 
		RETURNING id`,
		patientID, visitorName, pgvector.NewVector(faceEmbedding),
	).Scan(&visitorID)

	if err != nil {
		t.Fatalf("Failed to seed visitor: %v", err)
	}

	return visitorID
}
// Create a product and return the product ID
func SeedBriefing(
	t *testing.T, 
	pool *pgxpool.Pool, 
	patientID, visitorID int32,
	briefingText string,
) {
	t.Helper()

	ctx := context.Background()

	_, err := pool.Exec(ctx,
		`INSERT INTO briefings
		(patient_id, visitor_id, briefing_text)
		VALUES ($1, $2, $3)`,
		patientID, visitorID, briefingText,
	)

	if err != nil {
		t.Fatalf("Failed to seed briefing: %v", err)
	}
}

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
		WHERE patient_id = $1
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

func CheckUserEmbeddings(
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


// creates float32 array embeddings
func MakeEmbedding(value float32, size int) []float32 {
	emb := make([]float32, size)
	for i := range emb {
		emb[i] = value
	}
	return emb
}
package test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
)
// Create a product and return the product ID
func SeedBriefing(
	t *testing.T, 
	pool *pgxpool.Pool, 
	profileID, visitorID int32,
	briefingText string,
) {
	t.Helper()

	ctx := context.Background()

	_, err := pool.Exec(ctx,
		`INSERT INTO briefings
		(profile_id, visitor_id, briefing_text)
		VALUES ($1, $2, $3)`,
		profileID, visitorID, briefingText,
	)

	if err != nil {
		t.Fatalf("Failed to seed briefing: %v", err)
	}
}

// creates float32 array embeddings
func MakeEmbedding(value float32, size int) []float32 {
	emb := make([]float32, size)
	for i := range emb {
		emb[i] = value
	}
	return emb
}
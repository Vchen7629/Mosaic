package test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
)

// Read back the briefing_text for a given profile/visitor from the DB
func FetchBriefing(t *testing.T, pool *pgxpool.Pool, profileID int32, visitorID int32) (string, bool) {
	t.Helper()

	var text string
	err := pool.QueryRow(
		context.Background(),
		`SELECT briefing_text FROM briefings WHERE profile_id = $1 AND visitor_id = $2`,
		profileID, visitorID,
	).Scan(&text)
	if err != nil {
		return "", false
	}
	return text, true
}

// Create a profile row and its face embedding, return the profile id
func SeedConvos(
	t *testing.T, 
	pool *pgxpool.Pool,
	profileID int32, 
	visitorID int32,
	convo_text string,
) {
	t.Helper()

	_, err := pool.Exec(
		context.Background(),
		`INSERT INTO conversation_records 
		(profile_id, visitor_id, convo_text) 
		VALUES ($1, $2, $3)`,
		profileID, visitorID, convo_text)

	if err != nil {
		t.Fatalf("Failed to seed convo: %v", err)
	}
}
package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/Kagami/go-face"
	"github.com/jackc/pgx/v4"
	pgvector "github.com/pgvector/pgvector-go"
	"mosaic-face-detection.com/internal/observability"
	"mosaic-face-detection.com/internal/service"
)

// fetch all the visitor data like id, name, and  embeddings for a profile using profileID
func (db *DBPool) FetchAllVisitorData(profileID int32) ([]service.VisitorFaces, error) {
	if profileID <= 0 {
		return nil, errors.New("profileID must be positive")
	}
	ctx := context.Background()
	observability.DBReadsTotal.WithLabelValues("fetch_visitor_data").Inc()

	rows, err := db.pool.Query(ctx, `
		SELECT id, visitor_name, face_embedding
		FROM visitor_face_embeddings
		WHERE profile_id = $1
		AND visitor_name != 'Unknown'
	`, profileID)

	if err != nil {
		return nil, fmt.Errorf("error fetching from db for profileID: %v", err)
	}

	var visitorFaces []service.VisitorFaces

	for rows.Next() {
		var visitor service.VisitorFaces
		var embVector pgvector.Vector

		err := rows.Scan(&visitor.ID, &visitor.Name, &embVector)
		if err != nil {
			return nil, fmt.Errorf("error fetching visitor emb: %v", err)
		}

		copy(visitor.Embedding[:], embVector.Slice())
		visitorFaces = append(visitorFaces, visitor)
	}

	db.logger.Debug(
		"Fetched all visitor data for a specific profile from db",
		"profile_id", profileID,
		"visitor_face_count", len(visitorFaces),
	)

	return visitorFaces, nil
}

// fetch briefing for the visitor for the patient
func (db *DBPool) FetchVisitorBriefing(profileID, visitorID int32) (string, error) {
	if profileID <= 0 {
		return "", errors.New("profileID must be positive")
	}
	if visitorID <= 0 {
		return "", errors.New("visitorID must be positive")
	}

	ctx := context.Background()
	observability.DBReadsTotal.WithLabelValues("fetch_briefing").Inc()

	var briefing string

	query := `
		SELECT briefing_text
		FROM briefings
		WHERE profile_id = $1
		AND visitor_id = $2
	`

	err := db.pool.QueryRow(ctx, query, profileID, visitorID).Scan(&briefing)
	if errors.Is(err, pgx.ErrNoRows) {
		db.logger.Warn(
			"No visitor briefings fetched for this visitor",
			"profile_id", profileID,
			"visitor_id", visitorID,
		)
		return "", nil
	}

	if err != nil {
		return "", fmt.Errorf("error fetching briefing: %w", err)
	}

	db.logger.Debug(
		"Fetched visitor briefing for the visitor for profile",
		"profile_id", profileID,
		"visitor_id", visitorID,
	)

	return briefing, nil
}

// Add a new visitor for a patient with their name and face_embedding
// and returns the id (visitor_id) of the newly created row
func (db *DBPool) AddNewFaceForVisitor(
	profileID int32, name string, embedding face.Descriptor,
) (*int32, error) {
	if profileID <= 0 {
		return nil, errors.New("profileID must be positive")
	}
	if name == "" {
		return nil, errors.New("name must be a non empty string")
	}

	observability.DBReadsTotal.WithLabelValues("add_visitor").Inc()

	err := service.ValidateEmbedding(embedding)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	embeddingVector := pgvector.NewVector(embedding[:])

	var visitor_id int32

	query := `
		INSERT INTO visitor_face_embeddings
		(profile_id, visitor_name, face_embedding)
		VALUES ($1, $2, $3)
		ON CONFLICT (profile_id, visitor_name) DO UPDATE
			SET visitor_name = EXCLUDED.visitor_name,
				face_embedding = EXCLUDED.face_embedding
		RETURNING id
	`

	err = db.pool.QueryRow(ctx, query, profileID, name, embeddingVector).Scan(&visitor_id)
	if err != nil {
		return nil, fmt.Errorf("error adding new visitor: %w", err)
	}

	db.logger.Debug(
		"created new visitor with face embedding",
		"profile_id", profileID,
		"visitor_name", name,
		"new_visitor_id", visitor_id,
	)

	return &visitor_id, nil
}

package db

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/Kagami/go-face"
	"github.com/jackc/pgx/v4"
	pgvector "github.com/pgvector/pgvector-go"
	"mosaic-face-detection.com/internal/service"
)

// fetch all user profile embeddings saved in the database
func (db *DBPool) FetchProfileFaceEmbForID(profileID int32) ([]service.ProfileFaces, error) {
	if profileID <= 0 {
		return nil, errors.New("profileID must be positive")
	}
	ctx := context.Background()

	rows, err := db.pool.Query(ctx, `
		SELECT profile_id, face_embedding FROM profile_face_embeddings
		WHERE profile_id = $1
	`, profileID)
	if err != nil {
		return nil, fmt.Errorf("error fetching profile face embeddings from db: %v", err)
	}

	var result []service.ProfileFaces
	for rows.Next() {
		var f service.ProfileFaces
		var embVector pgvector.Vector

		if err := rows.Scan(&f.ID, &embVector); err != nil {
			return nil, fmt.Errorf("error fetching profile emb: %v", err)
		}

		copy(f.Embedding[:], embVector.Slice())
		result = append(result, f)
	}
	return result, nil
}

// fetch all user profile embeddings saved in the database
func (db *DBPool) FetchAllProfileFaceEmb() ([]service.ProfileFaces, error) {
	ctx := context.Background()

	rows, err := db.pool.Query(ctx, `
		SELECT profile_id, face_embedding FROM profile_face_embeddings
	`)
	if err != nil {
		return nil, fmt.Errorf("error fetching profile face embeddings from db: %v", err)
	}

	var result []service.ProfileFaces
	for rows.Next() {
		var f service.ProfileFaces
		var embVector pgvector.Vector

		if err := rows.Scan(&f.ID, &embVector); err != nil {
			return nil, fmt.Errorf("error fetching profile emb: %v", err)
		}

		copy(f.Embedding[:], embVector.Slice())
		result = append(result, f)
	}
	return result, nil
}

// fetch all the visitor data like id, name, and  embeddings for a profile using profileID
func (db *DBPool) FetchAllVisitorData(profileID int32) ([]service.VisitorFaces, error) {
	if profileID <= 0 {
		return nil, errors.New("profileID must be positive")
	}
	ctx := context.Background()

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

		if err := rows.Scan(&visitor.ID, &visitor.Name, &embVector); err != nil {
			return nil, fmt.Errorf("error fetching visitor emb: %v", err)
		}

		copy(visitor.Embedding[:], embVector.Slice())
		visitorFaces = append(visitorFaces, visitor)
	}

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
	var briefing string

	query := `
		SELECT briefing_text
		FROM briefings
		WHERE profile_id = $1
		AND visitor_id = $2
	`

	err := db.pool.QueryRow(ctx, query, profileID, visitorID).Scan(&briefing)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}

	if err != nil {
		return "", fmt.Errorf("error fetching briefing: %w", err)
	}

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

	log.Printf("db, created new row with id %d", visitor_id)

	return &visitor_id, nil
}

// Add a new visitor for a user with their name and face_embedding
func (db *DBPool) AddNewFaceForUser(embeddings []face.Descriptor) (*int32, error) {
	for _, emb := range embeddings {
		err := service.ValidateEmbedding(emb)
		if err != nil {
			return nil, err
		}
	}

	ctx := context.Background()

	var profileID int32

	err := WithTransaction(ctx, db.pool, func(tx pgx.Tx) error {
		newProfileQuery := `INSERT INTO profiles DEFAULT VALUES RETURNING id`

		err := tx.QueryRow(ctx, newProfileQuery).Scan(&profileID)
		if err != nil {
			return fmt.Errorf("error adding new profile: %w", err)
		}

		// adding this to handle a edge case where the user never confirms
		// visitor name for unknown faces and saves the conversation recording
		addUnknownVisitorQuery := `
			INSERT INTO visitor_face_embeddings 
			(profile_id, visitor_name, face_embedding)
			VALUES ($1, 'Unknown', $2)`

		zeroEmbedding := make([]float32, 128) // zero-initialized by default
		zeroVector := pgvector.NewVector(zeroEmbedding)

		_, err = tx.Exec(ctx, addUnknownVisitorQuery, profileID, zeroVector)
		if err != nil {
			return fmt.Errorf("error adding unknown visitor for profile: %w", err)
		}

		insertEmbQuery := `
			INSERT INTO profile_face_embeddings
			(profile_id, face_embedding)
			VALUES ($1, $2)`

		for _, emb := range embeddings {
			embeddingVector := pgvector.NewVector(emb[:])
			_, err = tx.Exec(ctx, insertEmbQuery, profileID, embeddingVector)
			if err != nil {
				return fmt.Errorf("error adding new embeddings: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &profileID, nil
}

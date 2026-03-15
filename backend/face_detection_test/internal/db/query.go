package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/Kagami/go-face"
	pgvector "github.com/pgvector/pgvector-go"
	"mosaic-face-detection.com/internal/service"
)

// fetch all user profile embeddings saved in the database
func (db *DBPool) FetchAllProfileFaceEmb() ([]service.Faces, error) {
	ctx := context.Background()

	rows, err := db.pool.Query(ctx, `SELECT id, face_embedding FROM patient`)
	if err != nil {
		return nil, fmt.Errorf("error fetching profile face embeddings from db: %v", err)
	}

	return scanFaceEmbeddings(rows, "profile")
}

// fetch all the visitor id embeddings for a patient using patientID
func (db *DBPool) FetchAllVisitorFaceEmbForPatient(
	patientID int32,
) ([]service.Faces, error) {
	if patientID <= 0 {
		return nil, errors.New("patientID must be positive")
	} 
	ctx := context.Background()

	rows, err := db.pool.Query(ctx, `
		SELECT id, face_embedding
		FROM visitor_face_embeddings
		WHERE patient_id = $1
	`, patientID)

	if err != nil {
		return nil, fmt.Errorf("error fetching from db for patientID: %v", err)
	}

	return scanFaceEmbeddings(rows, "visitor")
}

// fetch briefing for the visitor for the patient
func (db *DBPool) FetchVisitorBriefing(patientID, visitorID int32) (string, error) {
	if patientID <= 0 {
		return "", errors.New("patientID must be positive")
	} 
	if visitorID <= 0 {
		return "", errors.New("visitorID must be positive")
	}

	ctx := context.Background()
	var briefing string

	query := `
		SELECT briefing_text
		FROM briefings
		WHERE patient_id = $1
		AND visitor_id = $2
	`

	err := db.pool.QueryRow(ctx, query, patientID, visitorID).Scan(&briefing)
	if err != nil {
		return "", fmt.Errorf("error fetching briefing: %w", err)
	}

	return briefing, nil
}

// Add a new visitor for a patient with their name and face_embedding
func (db *DBPool) AddNewFaceForVisitor(
	patientID int32, name string, embedding face.Descriptor,
) error {
	if patientID <= 0 {
		return errors.New("patientID must be positive")
	} 
	if name == "" {
		return errors.New("name must be a non empty string")
	}

	err := service.ValidateEmbedding(embedding)
	if err != nil {
		return err
	}
	
	ctx := context.Background()

	embeddingVector := pgvector.NewVector(embedding[:])

	query := `
		INSERT INTO visitor_face_embeddings
		(patient_id, visitor_name, face_embedding)
		VALUES ($1, $2, $3)
		ON CONFLICT (patient_id, visitor_name) DO UPDATE
			SET visitor_name = EXCLUDED.visitor_name,
				face_embedding = EXCLUDED.face_embedding
	`

	_, err = db.pool.Exec(ctx, query, patientID, name, embeddingVector)
	if err != nil {
		return fmt.Errorf("error adding new visitor: %w", err)
	}

	return nil
}

// Add a new visitor for a user with their name and face_embedding
func (db *DBPool) AddNewFaceForUser(embedding face.Descriptor) (*int32, error) {
	err := service.ValidateEmbedding(embedding)
	if err != nil {
		return nil, err
	}
	
	ctx := context.Background()

	var patientID int32

	embeddingVector := pgvector.NewVector(embedding[:])

	query := `INSERT INTO patient (face_embedding) VALUES ($1) RETURNING id`

	err = db.pool.QueryRow(ctx, query, embeddingVector).Scan(&patientID)
	if err != nil {
		return nil, fmt.Errorf("error adding new user: %w", err)
	}

	return &patientID, nil
}
package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/Kagami/go-face"
	pgvector "github.com/pgvector/pgvector-go"
	"mosaic-face-detection.com/internal/service"
)

// fetch all the visitor id embeddings for a patient using patientID
func (db *DBPool) FetchAllVisitorFaceEmbForPatient(
	patientID int32,
) ([]service.KnownVisitor, error) {
	if patientID <= 0 {
		return nil, errors.New("patientID must be positive")
	} 
	ctx := context.Background()
	var visitorEmbList []service.KnownVisitor

	query := `
		SELECT id, face_embedding
		FROM visitor_face_embeddings
		WHERE patient_id = $1
	`

	rows, err := db.pool.Query(ctx, query, patientID)
	if err != nil {
		return nil, fmt.Errorf("error fetching from db for patientID: %v", err)
	}

	for rows.Next() {
		var visitorEmb service.KnownVisitor
		var embVector pgvector.Vector

		err := rows.Scan(
			&visitorEmb.ID,
			&embVector,
		)

		if err != nil {
			return nil, fmt.Errorf("Error fetching visitor emb: %v", err)
		}

		// Convert pgvector to face.Descriptor ([128]float32)
		copy(visitorEmb.Embedding[:], embVector.Slice())
		visitorEmbList = append(visitorEmbList, visitorEmb)
	}

	return visitorEmbList, nil
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
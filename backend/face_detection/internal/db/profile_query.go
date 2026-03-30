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

// fetch all user profile embeddings saved in the database
func (db *DBPool) FetchProfileFaceEmbForID(profileID int32) ([]service.ProfileFaces, error) {
	if profileID <= 0 {
		return nil, errors.New("profileID must be positive")
	}
	ctx := context.Background()
	observability.DBReadsTotal.WithLabelValues("fetch_profile_embeddings").Inc()

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
	db.logger.Debug("Fetched profile face embeddings for profile from db", "profile_id", profileID)

	return result, nil
}

// fetch all user profile embeddings saved in the database
func (db *DBPool) FetchAllProfileFaceEmb() ([]service.ProfileFaces, error) {
	ctx := context.Background()
	observability.DBReadsTotal.WithLabelValues("fetch_all_profile_embeddings").Inc()

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

	db.logger.Debug("Fetched all profile face embeddings from db", "count", len(result))

	return result, nil
}

// Add a new visitor for a user with their name and face_embedding
func (db *DBPool) AddNewFaceForProfile(embeddings []face.Descriptor) (*int32, error) {
	observability.DBReadsTotal.WithLabelValues("register_profile").Inc()

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

	db.logger.Debug("Created new profile for user", "new_profile_id", profileID)

	return &profileID, nil
}

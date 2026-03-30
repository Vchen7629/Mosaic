package db

import (
	"context"
	"errors"
	"fmt"
)

func (db *DBPool) UpsertSession(profileID int32, sessionToken string) error {
	if profileID <= 0 {
		return errors.New("profileID must be positive")
	}

	if sessionToken == "" {
		return errors.New("missing sessionToken param")
	}

	_, err := db.pool.Exec(context.Background(), `
		INSERT INTO sessions (profile_id, session_token)
		VALUES ($1, $2)
		ON CONFLICT (profile_id) DO UPDATE
			SET session_token = EXCLUDED.session_token
	`, profileID, sessionToken)
	if err != nil {
		return fmt.Errorf("error upserting session: %w", err)
	}

	return nil
}

func (db *DBPool) FetchProfileIDWithSession(sessionToken string) (*int32, error) {
	if sessionToken == "" {
		return nil, errors.New("missing sessionToken param")
	}

	var profileID int32

	err := db.pool.QueryRow(context.Background(), `
		SELECT profile_id FROM sessions WHERE session_token = $1
	`, sessionToken).Scan(&profileID)
	if err != nil {
		return nil, fmt.Errorf("error fetching profile_id for session_token: %w", err)
	}

	return &profileID, nil
}

func (db *DBPool) FetchSessionWithProfileID(profileID int32) (*string, error) {
	if profileID <= 0 {
		return nil, errors.New("profileID must be positive")
	}

	var sessionToken string

	err := db.pool.QueryRow(context.Background(), `
		SELECT session_token FROM sessions WHERE profile_id = $1
	`, profileID).Scan(&sessionToken)
	if err != nil {
		return nil, fmt.Errorf("error fetching session_token for profile_id: %w", err)
	}

	return &sessionToken, nil
}

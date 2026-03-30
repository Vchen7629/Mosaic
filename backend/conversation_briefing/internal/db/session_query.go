package db

import (
	"context"
	"errors"
	"fmt"
)

func (db *DBPool) FetchProfileIDWithSession(ctx context.Context, sessionToken string) (*int32, error) {
	if sessionToken == "" {
		return nil, errors.New("missing sessionToken param")
	}

	var profileID int32

	err := db.pool.QueryRow(ctx, `
		SELECT profile_id FROM sessions WHERE session_token = $1
	`, sessionToken).Scan(&profileID)
	if err != nil {
		return nil, fmt.Errorf("error fetching profile_id for session_token: %w", err)
	}

	return &profileID, nil
}

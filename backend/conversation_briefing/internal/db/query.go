package db

import (
	"context"
	"errors"
	"fmt"
)

type Conversations struct {
	ProfileID int32
	VisitorID int32
	ConvoList []string
}

// fetch 5 most recent conversations from the db for the profile
func (db *DBPool) FetchRecentConversations(profileID, visitorID int32) ([]Conversations, error) {
	if profileID <= 0 {
		return nil, errors.New("profileID must be positive")
	}
	if visitorID <= 0 {
		return nil, errors.New("visitorID must be positive")
	}

	ctx := context.Background()

	rows, err := db.pool.Query(ctx, `
		SELECT convo_text FROM conversation_records
		WHERE profile_id = $1
		AND visitor_id = ANY($2)
		ORDER BY visitor_id, created_at DESC
	`, profileID, visitorID)
	if err != nil {
		return nil, fmt.Errorf("error fetching convo text from db: %v", err)
	}
	defer rows.Close()

	// grouping convo by visitorID
	convoMap := make(map[int32]*Conversations)
	for rows.Next() {
		var visitorID int32
		var convo string

		err := rows.Scan(&visitorID, &convo)
		if err != nil {
			return nil, fmt.Errorf("error fetching conversation: %v", err)
		}

		_, ok := convoMap[visitorID]
		if !ok {
			convoMap[visitorID] = &Conversations{
				ProfileID: profileID,
				VisitorID: visitorID,
			}
		}

		if len(convoMap[visitorID].ConvoList) < 5 {
			convoMap[visitorID].ConvoList = append(convoMap[visitorID].ConvoList, convo)
		}
	}

	result := make([]Conversations, 0, len(convoMap))
	for _, convo := range convoMap {
		result = append(result, *convo)
	}

	return result, nil
}

// fetch all user profile embeddings saved in the database
func (db *DBPool) InsertBriefing(profileID, visitorID int32, briefing string) error {
	ctx := context.Background()

	_, err := db.pool.Exec(ctx, `
		INSERT INTO briefings (profile_id, visitor_id, briefing_text) 
		VALUES ($1, $2, $3)
		ON CONFLICT (profile_id, visitor_id) DO UPDATE
			SET convo_text = EXCLUDED.convo_text,
	`, profileID, visitorID, briefing)
	if err != nil {
		return fmt.Errorf("error inserting new briefing: %v", err)
	} 

	return nil
}

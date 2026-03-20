package db

import (
	"context"
	"errors"
	"fmt"

	"mosaic-conversation-briefing.com/internal/observability"
)

type Conversations struct {
	ProfileID int32
	VisitorID int32
	ConvoList []string
}

// fetch 5 most recent conversations from the db for the profile
func (db *DBPool) FetchRecentConversations(profileID int32, visitorIDs []int32) ([]Conversations, error) {
	ctx := context.Background()
	observability.DBReadsTotal.WithLabelValues("fetch_conversations").Inc()
	observability.CacheMissesTotal.WithLabelValues("fetch_conversations").Inc()

	rows, err := db.pool.Query(ctx, `
		SELECT visitor_id, convo_text FROM conversation_records
		WHERE profile_id = $1
		AND visitor_id = ANY($2)
		ORDER BY visitor_id, created_at DESC
	`, profileID, visitorIDs)
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

	db.logger.Debug(
		"Fetched recent conversations", 
		"profile_id", profileID, 
		"visitor_ids", visitorIDs, 
		"convo_count", len(convoMap),
	)

	result := make([]Conversations, 0, len(convoMap))
	for _, convo := range convoMap {
		result = append(result, *convo)
	}

	return result, nil
}

// Upsert briefing for all visitorIDs individually
// Not atomic currently since rather have some succeed with new briefing rather than all for nothing
func (db *DBPool) InsertBriefing(profileID int32, briefings map[int32]string) error {
	ctx := context.Background()
	observability.DBWritesTotal.WithLabelValues("insert_briefings").Inc()
	observability.CacheMissesTotal.WithLabelValues("insert_briefings").Inc()

	query := `INSERT INTO briefings (profile_id, visitor_id, briefing_text)
			VALUES ($1, $2, $3)
			ON CONFLICT (profile_id, visitor_id) DO UPDATE
				SET briefing_text = EXCLUDED.briefing_text`

	failCount := 0
	for visitorID, briefingText := range briefings {
		_, err := db.pool.Exec(ctx, query, profileID, visitorID, briefingText)
		if err != nil {
			// not failing if some visitor ID fails so succeeded visitors will get updated briefing
			db.logger.Warn("briefing failed to save", "visitor", visitorID, "err", err)
			failCount++
		}
	}

	if failCount == len(briefings) {
		return errors.New("all briefing inserts failed")
	}

	return nil
}

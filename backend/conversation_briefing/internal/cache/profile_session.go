package cache

import (
	"context"
	"errors"
	"fmt"

	"github.com/valkey-io/valkey-go"
	"mosaic-conversation-briefing.com/internal/observability"
)

// adds a new profile session to the valkey cache
func CreateNewProfileSession(
	ctx context.Context,
	profileID int32,
	cacheClient valkey.Client,
	sessionToken string,
) error {
	if cacheClient == nil {
		return errors.New("no cache client provided")
	}

	key := fmt.Sprintf("session:%s", sessionToken)
	value := fmt.Sprintf("%d", profileID)

	err := cacheClient.Do(ctx, cacheClient.B().Set().Key(key).Value(value).ExSeconds(43200).Build()).Error()
	if err != nil {
		observability.ErrorsTotal.WithLabelValues("create_new_profile_session").Inc()
		return fmt.Errorf("error storing session in cache: %w", err)
	}
	return nil
}

func FetchProfileIDFromCache(
	ctx context.Context,
	cacheClient valkey.Client,
	sessionToken string,
) (*int32, error) {
	if cacheClient == nil {
		return nil, errors.New("no cache client provided")
	}
	key := fmt.Sprintf("session:%s", sessionToken)

	r := cacheClient.Do(ctx, cacheClient.B().Get().Key(key).Build())

	val, err := r.AsInt64()
	if err != nil {
		observability.ErrorsTotal.WithLabelValues("fetch_profile_id_from_cache").Inc()
		return nil, err
	}

	result := int32(val)

	return &result, nil
}

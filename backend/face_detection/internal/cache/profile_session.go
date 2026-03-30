package cache

import (
	"context"
	"errors"
	"fmt"

	"github.com/valkey-io/valkey-go"
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
		return fmt.Errorf("error storing session in cache: %w", err)
	}
	return nil
}

func FetchProfileIDFromCache(
	ctx context.Context,
	cacheClient valkey.Client,
	sessionToken string,
) (int32, error) {
	key := fmt.Sprintf("session:%s", sessionToken)

	result := cacheClient.Do(ctx, cacheClient.B().Get().Key(key).Build())
	
	val, err := result.AsInt64()
	if err != nil {
		return 0, err
	}

	return int32(val), nil
}
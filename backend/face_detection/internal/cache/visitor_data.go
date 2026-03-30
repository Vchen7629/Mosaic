package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/valkey-io/valkey-go"
	"mosaic-face-detection.com/internal/observability"
	"mosaic-face-detection.com/internal/service"
)

func AppendToVisitorDataCache(
	ctx context.Context,
	profileID int32,
	cacheClient valkey.Client,
	newEntry service.VisitorFaces,
) {
	if cacheClient == nil {
		return
	}

	key := fmt.Sprintf("fetch_all_visitor_data:%d", profileID)

	cacheResult := cacheClient.Do(ctx, cacheClient.B().Get().Key(key).Build())
	cacheBytes, err := cacheResult.AsBytes()
	if err != nil {
		return
	}

	var visitors []service.VisitorFaces

	jsonErr := json.Unmarshal(cacheBytes, &visitors)
	if jsonErr != nil {
		return
	}

	visitors = append(visitors, newEntry)

	updated, jsonErr := json.Marshal(visitors)
	if jsonErr != nil {
		return
	}

	cacheClient.Do(ctx, cacheClient.B().Set().Key(key).Value(string(updated)).ExSeconds(43200).Build())
}

func FetchAllVisitorDataWithCache(
	ctx context.Context,
	profileID int32,
	cacheClient valkey.Client,
	logger *slog.Logger,
	fetchAllVisitorData func() ([]service.VisitorFaces, error),
) ([]service.VisitorFaces, error) {
	key := fmt.Sprintf("fetch_all_visitor_data:%d", profileID)

	if cacheClient == nil {
		return fetchAllVisitorData()
	}

	cacheResult := cacheClient.Do(ctx, cacheClient.B().Get().Key(key).Build())

	cacheBytes, err := cacheResult.AsBytes()
	if err == nil {
		var visitors []service.VisitorFaces

		jsonErr := json.Unmarshal(cacheBytes, &visitors)
		if jsonErr == nil {
			observability.CacheHitsTotal.WithLabelValues("fetch_all_visitor_data").Inc()
			return visitors, nil
		}
	} else if !valkey.IsValkeyNil(err) {
		logger.Warn("cache get error", "key", key, "err", err)
	}

	observability.CacheMissesTotal.WithLabelValues("fetch_all_visitor_data").Inc()

	visitors, err := fetchAllVisitorData()
	if err != nil {
		return nil, err
	}

	cacheBytes, jsonErr := json.Marshal(visitors)
	if jsonErr == nil {
		cacheClient.Do(ctx, cacheClient.B().Set().Key(key).Value(string(cacheBytes)).ExSeconds(43200).Build())
	}

	return visitors, nil
}

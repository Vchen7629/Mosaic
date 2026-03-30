package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/valkey-io/valkey-go"
	"mosaic-face-detection.com/internal/observability"
	"mosaic-face-detection.com/internal/service"
)

func FetchProfileFaceEmbForIDWithCache(
	ctx context.Context,
	profileID int32,
	cacheClient valkey.Client,
	logger *slog.Logger,
	fetchProfileFaceEmb func() ([]service.ProfileFaces, error),
) ([]service.ProfileFaces, error) {
	key := fmt.Sprintf("fetch_profile_face_emb_for_id:%d", profileID)

	if cacheClient == nil {
		return fetchProfileFaceEmb()
	}

	cacheResult := cacheClient.Do(ctx, cacheClient.B().Get().Key(key).Build())

	cacheBytes, err := cacheResult.AsBytes()
	if err == nil {
		var embs []service.ProfileFaces

		jsonErr := json.Unmarshal(cacheBytes, &embs)
		if jsonErr == nil {
			observability.CacheHitsTotal.WithLabelValues("fetch_profile_face_emb_for_id").Inc()
			return embs, nil
		}
	} else if !valkey.IsValkeyNil(err) {
		logger.Warn("cache get error", "key", key, "err", err)
	}

	observability.CacheMissesTotal.WithLabelValues("fetch_profile_face_emb_for_id").Inc()

	profileFetchStart := time.Now()
	currentProfileEmbs, err := fetchProfileFaceEmb()
	observability.ProfileEmbFetchDuration.Observe(float64(time.Since(profileFetchStart).Milliseconds()))
	if err != nil {
		observability.ErrorsTotal.WithLabelValues("fetch_profile_embeddings").Inc()
		return nil, err
	}

	cacheBytes, jsonErr := json.Marshal(currentProfileEmbs)
	if jsonErr == nil {
		// 12 hrs
		cacheClient.Do(ctx, cacheClient.B().Set().Key(key).Value(string(cacheBytes)).ExSeconds(43200).Build())
	}

	return currentProfileEmbs, nil
}

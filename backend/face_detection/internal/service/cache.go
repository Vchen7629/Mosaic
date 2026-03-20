package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/valkey-io/valkey-go"
	"mosaic-face-detection.com/internal/observability"
)

func FetchProfileFaceEmbForIDWithCache(
	ctx context.Context,
	profileID int32,
	cacheClient valkey.Client,
	logger *slog.Logger,
	fetchProfileFaceEmb func() ([]ProfileFaces, error),
) ([]ProfileFaces, error) {
	key := fmt.Sprintf("fetch_profile_face_emb_for_id:%d", profileID)

	cacheResult := cacheClient.Do(ctx, cacheClient.B().Get().Key(key).Build())

	cacheBytes, err := cacheResult.AsBytes()
	if err == nil {
		var embs []ProfileFaces

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

func FetchAllVisitorDataWithCache(
	ctx context.Context,
	profileID int32,
	cacheClient valkey.Client,
	logger *slog.Logger,
	fetchAllVisitorData func() ([]VisitorFaces, error),
) ([]VisitorFaces, error) {
	key := fmt.Sprintf("fetch_all_visitor_data:%d", profileID)

	cacheResult := cacheClient.Do(ctx, cacheClient.B().Get().Key(key).Build())

	cacheBytes, err := cacheResult.AsBytes()
	if err == nil {
		var visitors []VisitorFaces

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

func AppendToVisitorDataCache(
	ctx context.Context,
	profileID int32,
	cacheClient valkey.Client,
	newEntry VisitorFaces,
) {
	key := fmt.Sprintf("fetch_all_visitor_data:%d", profileID)

	cacheResult := cacheClient.Do(ctx, cacheClient.B().Get().Key(key).Build())
	cacheBytes, err := cacheResult.AsBytes()
	if err != nil {
		return
	}

	var visitors []VisitorFaces

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

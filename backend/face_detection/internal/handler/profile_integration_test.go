//go:build integration

package handler_test

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	fd "mosaic-face-detection.com/gen"
	"mosaic-face-detection.com/internal/cache"
	"mosaic-face-detection.com/internal/db"
	"mosaic-face-detection.com/internal/handler"
	"mosaic-face-detection.com/internal/service"
	"mosaic-face-detection.com/internal/test"
)

var testDB *test.TestDBContainer
var testCache *test.TestCacheContainer

const testModelsDir = "../models"
const testImagesDir = "../test/images"

func TestMain(m *testing.M) {
	var dbcleanup func()
	testDB, dbcleanup = test.SetupTestDatabaseForTestMain()
	defer dbcleanup()

	var cacheCleanup func()
	testCache, cacheCleanup = test.SetupTestCacheForTestMain()
	defer cacheCleanup()

	m.Run()
}

func TestSyncProfile(t *testing.T) {
	recPool, err := service.NewRecognizerPool(testModelsDir, 5)
	assert.NoError(t, err)

	pool := testDB.Pool
	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(jsonHandler).With("service", "face_detection")
	dbPool := db.NewDBPool(pool, logger)
	cacheClient := testCache.Client
	server := handler.NewFaceDetectionServer(logger, recPool, cacheClient, dbPool)

	t.Run("Corrupt image bytes returns error from GenerateFaceEmbeddings", func(t *testing.T) {
		res, err := server.SyncProfile(context.Background(), &fd.SyncProfileRequest{
			FaceBytes: [][]byte{[]byte("not a valid image")},
		})

		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("DB pool failure after face detected returns error from FetchAllProfileFaceEmb", func(t *testing.T) {
		test.FlushCache(t, cacheClient)

		imgBytes, err := os.ReadFile(filepath.Join(testImagesDir, "bona.jpg"))
		assert.NoError(t, err)

		closedDB := test.SetupTestDatabase(t)
		closedDB.Pool.Close()
		closedDBPool := db.NewDBPool(closedDB.Pool, logger)
		closedServer := handler.NewFaceDetectionServer(logger, recPool, cacheClient, closedDBPool)

		res, err := closedServer.SyncProfile(context.Background(), &fd.SyncProfileRequest{
			FaceBytes: [][]byte{imgBytes},
		})

		assert.Error(t, err)
		assert.False(t, res.Success)
	})

	t.Run("No face bytes input should return face detected = false", func(t *testing.T) {
		test.FlushCache(t, cacheClient)

		res, err := server.SyncProfile(context.Background(), &fd.SyncProfileRequest{
			FaceBytes: [][]byte{},
		})

		assert.NoError(t, err)
		assert.False(t, res.FaceDetected)
	})

	t.Run("Face detected but no profiles in db registers new profile and returns session token", func(t *testing.T) {
		test.CleanupTables(t, pool)
		test.FlushCache(t, cacheClient)

		imgBytes, err := os.ReadFile(filepath.Join(testImagesDir, "bona.jpg"))
		assert.NoError(t, err)

		res, err := server.SyncProfile(context.Background(), &fd.SyncProfileRequest{
			FaceBytes: [][]byte{imgBytes},
		})

		assert.NoError(t, err)
		assert.True(t, res.FaceDetected)
		assert.True(t, res.Success)
		assert.NotEmpty(t, res.SessionToken)
	})

	t.Run("No face match registers new profile and stores session in db and cache", func(t *testing.T) {
		test.CleanupTables(t, pool)
		test.FlushCache(t, cacheClient)

		imgBytes, err := os.ReadFile(filepath.Join(testImagesDir, "bona.jpg"))
		assert.NoError(t, err)

		res, err := server.SyncProfile(context.Background(), &fd.SyncProfileRequest{
			FaceBytes: [][]byte{imgBytes},
		})

		assert.NoError(t, err)
		assert.True(t, res.FaceDetected)
		assert.True(t, res.Success)
		assert.NotEmpty(t, res.SessionToken)

		// verify session stored in db under the new profile (id=1 since tables were cleaned)
		newProfileID := int32(1)
		sessionInDB, dbErr := dbPool.FetchSessionWithProfileID(newProfileID)
		assert.NoError(t, dbErr)
		assert.Equal(t, res.SessionToken, *sessionInDB)

		// verify session stored in cache
		profileIDFromCache, cacheErr := cache.FetchProfileIDFromCache(context.Background(), cacheClient, res.SessionToken)
		assert.NoError(t, cacheErr)
		assert.Equal(t, newProfileID, *profileIDFromCache)
	})

	t.Run("No face match with existing profiles creates a separate new profile", func(t *testing.T) {
		test.CleanupTables(t, pool)
		test.FlushCache(t, cacheClient)

		imgBytes, err1 := os.ReadFile(filepath.Join(testImagesDir, "bona.jpg"))
		noMatchImgBytes, err2 := os.ReadFile(filepath.Join(testImagesDir, "man1.jpg"))
		assert.NoError(t, err1)
		assert.NoError(t, err2)

		existingProfileID := test.AddNewProfile(t, recPool, noMatchImgBytes, testDB)

		res, err := server.SyncProfile(context.Background(), &fd.SyncProfileRequest{
			FaceBytes: [][]byte{imgBytes},
		})

		assert.NoError(t, err)
		assert.True(t, res.FaceDetected)
		assert.True(t, res.Success)
		assert.NotEmpty(t, res.SessionToken)

		// session should belong to the newly created profile, not the existing one
		sessionInDB, dbErr := dbPool.FetchSessionWithProfileID(existingProfileID + 1)
		assert.NoError(t, dbErr)
		assert.Equal(t, res.SessionToken, *sessionInDB)
	})

	t.Run("Face matching a profile in db should return a session token", func(t *testing.T) {
		test.CleanupTables(t, pool)
		test.FlushCache(t, cacheClient)

		imgBytes, err := os.ReadFile(filepath.Join(testImagesDir, "bona.jpg"))
		assert.NoError(t, err)

		_ = test.AddNewProfile(t, recPool, imgBytes, testDB)

		res, err := server.SyncProfile(context.Background(), &fd.SyncProfileRequest{
			FaceBytes: [][]byte{imgBytes},
		})

		assert.NoError(t, err)
		assert.True(t, res.FaceDetected)
		assert.True(t, res.Success)
		assert.NotEmpty(t, res.SessionToken)
	})

	t.Run("Face match stores session in both cache and db", func(t *testing.T) {
		test.CleanupTables(t, pool)
		test.FlushCache(t, cacheClient)

		imgBytes, err := os.ReadFile(filepath.Join(testImagesDir, "bona.jpg"))
		assert.NoError(t, err)

		profileID := test.AddNewProfile(t, recPool, imgBytes, testDB)

		res, err := server.SyncProfile(context.Background(), &fd.SyncProfileRequest{
			FaceBytes: [][]byte{imgBytes},
		})

		assert.NoError(t, err)
		assert.NotEmpty(t, res.SessionToken)

		// verify session persisted in db
		sessionInDB, dbErr := dbPool.FetchSessionWithProfileID(profileID)
		assert.NoError(t, dbErr)
		assert.Equal(t, res.SessionToken, *sessionInDB)

		// verify session persisted in cache
		profileIDFromCache, cacheErr := cache.FetchProfileIDFromCache(context.Background(), cacheClient, res.SessionToken)
		assert.NoError(t, cacheErr)
		assert.Equal(t, profileID, *profileIDFromCache)
	})

	t.Run("Multiple frames of same face aggregates embeddings and still matches profile", func(t *testing.T) {
		test.CleanupTables(t, pool)
		test.FlushCache(t, cacheClient)

		imgBytes1, err1 := os.ReadFile(filepath.Join(testImagesDir, "bona.jpg"))
		imgBytes2, err2 := os.ReadFile(filepath.Join(testImagesDir, "bona2.jpg"))
		assert.NoError(t, err1)
		assert.NoError(t, err2)

		_ = test.AddNewProfile(t, recPool, imgBytes1, testDB)

		// Pass two different images of the same person as separate frames to verify aggregation
		res, err := server.SyncProfile(context.Background(), &fd.SyncProfileRequest{
			FaceBytes: [][]byte{imgBytes1, imgBytes2},
		})

		assert.NoError(t, err)
		assert.True(t, res.FaceDetected)
		assert.True(t, res.Success)
		assert.NotEmpty(t, res.SessionToken, "should return a session token for the matching profile")
	})
}

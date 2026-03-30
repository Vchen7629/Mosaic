//go:build integration

package handler_test

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/Kagami/go-face"
	"github.com/stretchr/testify/assert"
	fd "mosaic-face-detection.com/gen"
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

	t.Run("No face bytes input should return face detected = false", func(t *testing.T) {
		test.FlushCache(t, cacheClient)

		res, err := server.SyncProfile(context.Background(), &fd.SyncProfileRequest{
			FaceBytes: [][]byte{},
		})

		assert.NoError(t, err)
		assert.False(t, res.FaceDetected)
	})

	t.Run("Face detected but no profiles in db should return correct vals", func(t *testing.T) {
		test.CleanupTables(t, pool)
		test.FlushCache(t, cacheClient)

		imgBytes, err := os.ReadFile(filepath.Join(testImagesDir, "bona.jpg"))
		assert.NoError(t, err)

		res, err := server.SyncProfile(context.Background(), &fd.SyncProfileRequest{
			FaceBytes: [][]byte{imgBytes},
		})

		assert.NoError(t, err)
		assert.True(t, res.FaceDetected)
		assert.True(t, res.NewFace)
		assert.True(t, res.Success)
		assert.NotEmpty(t, res.FaceEmbedding)
		assert.Len(t, res.FaceEmbedding[0].FaceEmbedding, 128, "should be a 128 dim emb")
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
		assert.False(t, res.NewFace)
		assert.NotEmpty(t, res.SessionToken)
	})

	t.Run("Face detected but no matches with existing profile should return emb", func(t *testing.T) {
		test.CleanupTables(t, pool)
		test.FlushCache(t, cacheClient)

		imgBytes, err1 := os.ReadFile(filepath.Join(testImagesDir, "bona.jpg"))
		noMatchImgBytes, err2 := os.ReadFile(filepath.Join(testImagesDir, "man1.jpg"))

		assert.NoError(t, err1)
		assert.NoError(t, err2)

		_ = test.AddNewProfile(t, recPool, noMatchImgBytes, testDB)

		res, err := server.SyncProfile(context.Background(), &fd.SyncProfileRequest{
			FaceBytes: [][]byte{imgBytes},
		})

		assert.NoError(t, err)
		assert.True(t, res.FaceDetected)
		assert.NotEmpty(t, res.FaceEmbedding)
		assert.Len(t, res.FaceEmbedding[0].FaceEmbedding, 128, "should be a 128 dim emb")
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
		assert.False(t, res.NewFace, "should match existing profile, not be a new face")
		assert.NotEmpty(t, res.SessionToken, "should return a session token for the matching profile")
	})
}

func TestRegisterProfileFace(t *testing.T) {
	recPool, err := service.NewRecognizerPool(testModelsDir, 5)
	assert.NoError(t, err)

	pool := testDB.Pool
	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(jsonHandler).With("service", "face_detection")
	dbPool := db.NewDBPool(pool, logger)
	cacheClient := testCache.Client
	server := handler.NewFaceDetectionServer(logger, recPool, cacheClient, dbPool)

	t.Run("One valid embedding should be saved to db properly", func(t *testing.T) {
		test.CleanupTables(t, pool)
		test.FlushCache(t, cacheClient)

		validEmbedding := test.MakeEmbedding(0.5, 128)
		var embedding face.Descriptor
		copy(embedding[:], validEmbedding)

		res, err := server.RegisterProfileFace(context.Background(), &fd.RegisterProfileFaceRequest{
			FaceEmbedding: []*fd.FaceEmbedding{
				{FaceEmbedding: embedding[:]},
			},
		})

		dbEmb := test.CheckProfileEmbeddings(t, pool, int32(1))

		assert.NoError(t, err)
		assert.True(t, res.Success)
		assert.EqualValues(t, dbEmb, embedding)
		assert.NotEmpty(t, res.SessionToken, "should return a session token")
	})

	t.Run("Multiple valid embedding should be saved to db properly", func(t *testing.T) {
		test.CleanupTables(t, pool)
		test.FlushCache(t, cacheClient)

		embeddings := make([]face.Descriptor, 3)
		for i, val := range []float32{0.1, 0.2, 0.3} {
			copy(embeddings[i][:], test.MakeEmbedding(val, 128))
		}

		fdEmbeddings := make([]*fd.FaceEmbedding, len(embeddings))
		for i, e := range embeddings {
			fdEmbeddings[i] = &fd.FaceEmbedding{FaceEmbedding: e[:]}
		}

		res, err := server.RegisterProfileFace(context.Background(), &fd.RegisterProfileFaceRequest{
			FaceEmbedding: fdEmbeddings,
		})

		assert.NoError(t, err)
		assert.True(t, res.Success)

		dbEmbs := test.CheckAllProfileEmbeddings(t, pool, int32(1))
		assert.Len(t, dbEmbs, len(embeddings), "all embeddings should be saved")
		assert.EqualValues(t, embeddings, dbEmbs)
		assert.NotEmpty(t, res.SessionToken, "should return a session token")
	})

	t.Run("Return error on invalid embedding length", func(t *testing.T) {
		test.CleanupTables(t, pool)
		test.FlushCache(t, cacheClient)

		invalidEmbedding := test.MakeEmbedding(0.5, 256)

		_, err := server.RegisterProfileFace(context.Background(), &fd.RegisterProfileFaceRequest{
			FaceEmbedding: []*fd.FaceEmbedding{
				{FaceEmbedding: invalidEmbedding},
			},
		})

		assert.Error(t, err)
	})

	t.Run("Empty embeddings slice should return error", func(t *testing.T) {
		test.CleanupTables(t, pool)
		test.FlushCache(t, cacheClient)

		_, err := server.RegisterProfileFace(context.Background(), &fd.RegisterProfileFaceRequest{
			FaceEmbedding: []*fd.FaceEmbedding{},
		})

		assert.Error(t, err)
	})
}

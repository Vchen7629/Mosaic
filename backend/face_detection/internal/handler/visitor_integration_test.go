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

func TestProcessVisitorFaces(t *testing.T) {
	recPool, err := service.NewRecognizerPool(testModelsDir, 5)
	assert.NoError(t, err)
	defer recPool.Close()

	rec := recPool.Acquire()
	defer recPool.Release(rec)

	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(jsonHandler).With("service", "face_detection")
	pool := testDB.Pool
	dbPool := db.NewDBPool(pool, logger)
	cacheClient := testCache.Client
	server := handler.NewFaceDetectionServer(logger, recPool, cacheClient, dbPool)

	t.Run("No face bytes input should return face detected = false", func(t *testing.T) {
		test.FlushCache(t, cacheClient)

		res, err := server.ProcessVisitorFaces(context.Background(), &fd.ProcessVisitorFacesRequest{
			FaceBytes: []byte{},
		})

		assert.NoError(t, err)
		assert.False(t, res.FaceDetected)
	})

	t.Run("Trying to process a face that matches no visitors in the db should return isKnown false and face embedding", func(t *testing.T) {
		test.CleanupTables(t, pool)
		test.FlushCache(t, cacheClient)
		service.ClearSessions()

		imgBytes, err := os.ReadFile(filepath.Join(testImagesDir, "bona.jpg"))
		assert.NoError(t, err)

		profileFaceEmb := test.GetEmbedding(t, rec, "man1.jpg")
		profileID := test.SeedProfile(t, pool, profileFaceEmb[:])
		sessionToken := test.SeedSession(t, profileID)

		res, err := server.ProcessVisitorFaces(context.Background(), &fd.ProcessVisitorFacesRequest{
			FaceBytes:    imgBytes,
			SessionToken: sessionToken,
		})

		assert.NoError(t, err)
		assert.True(t, res.FaceDetected)
		assert.True(t, res.Success)
		assert.NotEmpty(t, res.Faces)
		assert.False(t, res.Faces[0].IsKnown)
		assert.Len(t, res.Faces[0].FaceEmbedding, 128, "should be a 128 dim emb")
	})

	t.Run("Face matching a visitor in db and not the current profile should return isKnown true with briefing", func(t *testing.T) {
		test.CleanupTables(t, pool)
		test.FlushCache(t, cacheClient)
		service.ClearSessions()

		imgBytes, err := os.ReadFile(filepath.Join(testImagesDir, "bona.jpg"))
		assert.NoError(t, err)

		matchingFaceEmb := test.GetEmbedding(t, rec, "bona2.jpg")
		nonMatchingFaceEmb := test.GetEmbedding(t, rec, "man1.jpg")
		reqProfileID := test.SeedProfile(t, pool, nonMatchingFaceEmb[:])
		visitorID := test.SeedVisitor(t, pool, reqProfileID, "visitor1", matchingFaceEmb[:])
		sessionToken := test.SeedSession(t, reqProfileID)

		res, err := server.ProcessVisitorFaces(context.Background(), &fd.ProcessVisitorFacesRequest{
			FaceBytes:    imgBytes,
			SessionToken: sessionToken,
		})

		assert.NoError(t, err)
		assert.True(t, res.FaceDetected)
		assert.True(t, res.Success)
		assert.Equal(t, visitorID, res.Faces[0].VisitorId, "the visitor id for the matching face in db and the req should match")
		assert.Equal(t, "visitor1", res.Faces[0].Name, "the name should match")
		assert.Equal(t, "", res.Faces[0].Briefing, "briefing should be just empty since the visitor was just created")
	})

	t.Run("Visitor Face matching currently synced profile face should return the NonVisitorFace = true and face_detected = true", func(t *testing.T) {
		test.CleanupTables(t, pool)
		test.FlushCache(t, cacheClient)
		service.ClearSessions()

		imgBytes, err := os.ReadFile(filepath.Join(testImagesDir, "bona.jpg"))
		assert.NoError(t, err)

		profileID := test.AddNewProfile(t, recPool, imgBytes, testDB)
		sessionToken := test.SeedSession(t, profileID)

		res, err := server.ProcessVisitorFaces(context.Background(), &fd.ProcessVisitorFacesRequest{
			FaceBytes:    imgBytes,
			SessionToken: sessionToken,
		})

		assert.NoError(t, err)
		assert.True(t, res.NonVisitorFace)
		assert.True(t, res.FaceDetected)
	})
}

func TestRegisterVisitorFace(t *testing.T) {
	recPool, err := service.NewRecognizerPool(testModelsDir, 5)
	assert.NoError(t, err)

	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(jsonHandler).With("service", "face_detection")
	pool := testDB.Pool
	dbPool := db.NewDBPool(pool, logger)
	cacheClient := testCache.Client
	server := handler.NewFaceDetectionServer(logger, recPool, cacheClient, dbPool)

	t.Run("One valid embedding should be saved to db properly", func(t *testing.T) {
		test.CleanupTables(t, pool)
		test.FlushCache(t, cacheClient)
		service.ClearSessions()

		validEmbedding := test.MakeEmbedding(0.5, 128)
		var embedding face.Descriptor
		copy(embedding[:], validEmbedding)

		profileID := test.SeedProfile(t, pool, validEmbedding)
		sessionToken := test.SeedSession(t, profileID)

		res, err := server.RegisterVisitorFace(context.Background(), &fd.RegisterVisitorFaceRequest{
			FaceEmbedding: embedding[:],
			SessionToken:  sessionToken,
			VisitorName:   "test_visitor",
		})

		dbEmb := test.CheckVisitorEmbeddings(t, pool, profileID, "test_visitor")

		assert.NoError(t, err)
		assert.True(t, res.Success)
		assert.EqualValues(t, dbEmb, embedding)
		assert.Equal(t, int32(1), res.VisitorId, "should also return the visitor id")
	})

	t.Run("Return error on invalid embedding length", func(t *testing.T) {
		invalidEmbedding := test.MakeEmbedding(0.5, 256)

		_, err := server.RegisterVisitorFace(context.Background(), &fd.RegisterVisitorFaceRequest{
			FaceEmbedding: invalidEmbedding,
			SessionToken:  "any-token",
			VisitorName:   "test_visitor",
		})

		assert.Error(t, err)
	})

	t.Run("Empty embeddings slice should return error", func(t *testing.T) {
		_, err := server.RegisterVisitorFace(context.Background(), &fd.RegisterVisitorFaceRequest{
			FaceEmbedding: []float32{},
			SessionToken:  "any-token",
			VisitorName:   "test_visitor",
		})

		assert.Error(t, err)
	})
}

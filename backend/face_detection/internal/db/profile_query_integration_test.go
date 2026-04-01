//go:build integration

package db_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/Kagami/go-face"
	"github.com/stretchr/testify/assert"
	"mosaic-face-detection.com/internal/db"
	"mosaic-face-detection.com/internal/test"
)

func TestFetchAllProfileFaceEmb(t *testing.T) {
	pool := testDB.Pool
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(handler).With("service", "face_detection")
	dbPool := db.NewDBPool(pool, logger)

	t.Run("returns empty list when fetching from an empty db", func(t *testing.T) {
		test.CleanupTables(t, pool)

		profileEmbList, err := dbPool.FetchAllProfileFaceEmb()

		assert.Nil(t, err)
		assert.Empty(t, profileEmbList)
	})

	t.Run("returns the list containing id and emb", func(t *testing.T) {
		test.CleanupTables(t, pool)

		patientEmbs := []float32{0.1, 0.3, 0.5}
		profileIDs := make([]int32, 3)
		for i, val := range patientEmbs {
			profileIDs[i] = test.SeedProfile(t, pool, test.MakeEmbedding(val, 128))
		}

		profileEmbList, err := dbPool.FetchAllProfileFaceEmb()

		assert.Nil(t, err)
		assert.Equal(t, 3, len(profileEmbList))
		for i, expected := range patientEmbs {
			assert.Equal(t, profileIDs[i], profileEmbList[i].ID)
			expectedEmb := [128]float32{}
			for j := range expectedEmb {
				expectedEmb[j] = expected
			}
			assert.EqualValues(t, expectedEmb, profileEmbList[i].Embedding)
		}
	})
}

func TestFetchProfileFaceEmbForID(t *testing.T) {
	pool := testDB.Pool
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(handler).With("service", "face_detection")
	dbPool := db.NewDBPool(pool, logger)

	t.Run("returns empty list when profile has no embeddings", func(t *testing.T) {
		test.CleanupTables(t, pool)

		// seed a profile without embeddings via direct insert
		var profileID int32
		err := pool.QueryRow(
			context.Background(),
			`INSERT INTO profiles DEFAULT VALUES RETURNING id`,
		).Scan(&profileID)
		assert.Nil(t, err)

		result, err := dbPool.FetchProfileFaceEmbForID(profileID)

		assert.Nil(t, err)
		assert.Empty(t, result)
	})

	t.Run("returns embeddings for a seeded profile", func(t *testing.T) {
		test.CleanupTables(t, pool)

		embedding := test.MakeEmbedding(0.5, 128)
		profileID := test.SeedProfile(t, pool, embedding)

		result, err := dbPool.FetchProfileFaceEmbForID(profileID)

		assert.Nil(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, profileID, result[0].ID)

		expectedEmb := [128]float32{}
		for i := range expectedEmb {
			expectedEmb[i] = 0.5
		}
		assert.EqualValues(t, expectedEmb, result[0].Embedding)
	})

	t.Run("returns only embeddings for the requested profile", func(t *testing.T) {
		test.CleanupTables(t, pool)

		profileID1 := test.SeedProfile(t, pool, test.MakeEmbedding(0.1, 128))
		_ = test.SeedProfile(t, pool, test.MakeEmbedding(0.9, 128))

		result, err := dbPool.FetchProfileFaceEmbForID(profileID1)

		assert.Nil(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, profileID1, result[0].ID)
	})
}

func TestAddNewFaceForProfile(t *testing.T) {
	pool := testDB.Pool
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(handler).With("service", "face_detection")
	dbPool := db.NewDBPool(pool, logger)

	t.Run("returns error and nil id for invalid embedding", func(t *testing.T) {
		test.CleanupTables(t, pool)

		zerosEmbedding := test.MakeEmbedding(0, 128)
		var invalidEmbedding face.Descriptor
		copy(invalidEmbedding[:], zerosEmbedding)

		id, err := dbPool.AddNewFaceForProfile([]face.Descriptor{invalidEmbedding})
		assert.Equal(t, "embedding cannot be all zeros", err.Error())
		assert.Nil(t, id)
	})

	t.Run("successfully adds multiple face embedding for profile", func(t *testing.T) {
		test.CleanupTables(t, pool)

		embeddings := make([]face.Descriptor, 3)
		for i, val := range []float32{0.1, 0.2, 0.3} {
			copy(embeddings[i][:], test.MakeEmbedding(val, 128))
		}

		id, err := dbPool.AddNewFaceForProfile(embeddings)
		assert.Nil(t, err)
		assert.NotNil(t, id)

		embeddingFromDB := test.CheckProfileEmbeddings(t, pool, *id)

		assert.EqualValues(t, embeddings[0], embeddingFromDB)
	})
}

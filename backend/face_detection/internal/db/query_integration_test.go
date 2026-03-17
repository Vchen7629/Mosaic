//go:build integration

package db_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/Kagami/go-face"
	"github.com/stretchr/testify/assert"
	"mosaic-face-detection.com/internal/db"
	"mosaic-face-detection.com/internal/test"
)

var testDB *test.TestDBContainer

// Setup and teardown database for all tests in this package
func TestMain(m *testing.M) {
	var cleanup func()
	testDB, cleanup = test.SetupTestDatabaseForTestMain()
	defer cleanup()

	// Run all tests
	code := m.Run()

	os.Exit(code)
}

func TestFetchAllVisitorFaceEmbForPatient(t *testing.T) {
	pool := testDB.Pool
	dbPool := db.NewDBPool(pool)
	t.Run("returns error for negative profileID", func(t *testing.T) {
		test.CleanupTables(t, pool)

		profileID := test.SeedProfile(t, pool, test.MakeEmbedding(0.1, 128))
		_ = test.SeedVisitor(t, pool, profileID, "testvisitor", test.MakeEmbedding(0.1, 128))

		visitorEmbList, err := dbPool.FetchAllVisitorFaceEmbForProfile(-1)

		assert.Nil(t, visitorEmbList, "emb list returns nil")
		assert.Equal(t, "profileID must be positive", err.Error())
	})

	t.Run("returns list of visitor embeddings", func(t *testing.T) {
		test.CleanupTables(t, pool)

		profileID := test.SeedProfile(t, pool, test.MakeEmbedding(0.1, 128))

		visitorEmbs := []float32{0.1, 0.2, 0.3}
		visitorIDs := make([]int32, 3)
		for i, val := range visitorEmbs {
			visitorIDs[i] = test.SeedVisitor(
				t, pool, profileID, fmt.Sprintf("visitor%d", i+1), test.MakeEmbedding(val, 128),
			)
		}

		visitorEmbList, err := dbPool.FetchAllVisitorFaceEmbForProfile(profileID)

		assert.Nil(t, err)
		assert.Equal(t, 3, len(visitorEmbList), "should return 3 visitor embedding")
		for i, expected := range visitorEmbs {
			assert.Equal(t, visitorIDs[i], visitorEmbList[i].ID)
			expectedEmb := [128]float32{}
			for j := range expectedEmb {
				expectedEmb[j] = expected
			}
			assert.EqualValues(t, expectedEmb, visitorEmbList[i].Embedding)
		}
	})

	t.Run("returns error when trying to fetch for a patient that doesnt exist", func(t *testing.T) {
		test.CleanupTables(t, pool)
		embedding := test.MakeEmbedding(0.1, 128)

		profileID := test.SeedProfile(t, pool, embedding)
		_ = test.SeedVisitor(t, pool, profileID, "visitor", embedding)

		visitorEmbList, err := dbPool.FetchAllVisitorFaceEmbForProfile(23)

		assert.Nil(t, err)
		assert.Equal(t, 0, len(visitorEmbList))
	})
}

func TestFetchAllProfileFaceEmb(t *testing.T) {
	pool := testDB.Pool
	dbPool := db.NewDBPool(pool)
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

func TestFetchVisitorBriefing(t *testing.T) {
	pool := testDB.Pool
	dbPool := db.NewDBPool(pool)
	embedding := test.MakeEmbedding(0.1, 128)

	t.Run("returns error for negative profileID and visitorID", func(t *testing.T) {
		test.CleanupTables(t, pool)
		testBriefing := "this is a test"

		profileID := test.SeedProfile(t, pool, embedding)
		visitorID := test.SeedVisitor(t, pool, profileID, "testvisitor", embedding)
		test.SeedBriefing(t, pool, profileID, visitorID, testBriefing)

		briefing, err := dbPool.FetchVisitorBriefing(-1, visitorID)

		assert.Equal(t, "", briefing, "briefing returns empty str on error")
		assert.Equal(t, "profileID must be positive", err.Error())

		briefing, err = dbPool.FetchVisitorBriefing(profileID, -1)

		assert.Equal(t, "", briefing, "briefing returns empty str on error")
		assert.Equal(t, "visitorID must be positive", err.Error())
	})

	t.Run("returns the briefing for the correct visitor", func(t *testing.T) {
		test.CleanupTables(t, pool)

		testBriefing1 := "this is a test briefing 1"
		testBriefing2 := "this is a test briefing 2"

		profileID := test.SeedProfile(t, pool, embedding)
		visitorID1 := test.SeedVisitor(t, pool, profileID, "testvisitor1", embedding)
		visitorID2 := test.SeedVisitor(t, pool, profileID, "testvisitor2", embedding)

		test.SeedBriefing(t, pool, profileID, visitorID1, testBriefing1)
		test.SeedBriefing(t, pool, profileID, visitorID2, testBriefing2)

		briefing1, err1 := dbPool.FetchVisitorBriefing(profileID, visitorID1)
		briefing2, err2 := dbPool.FetchVisitorBriefing(profileID, visitorID2)

		assert.Equal(t, testBriefing1, briefing1, "briefing returned for correct visitor")
		assert.Nil(t, err1)

		assert.Equal(t, testBriefing2, briefing2, "briefing returned for correct visitor")
		assert.Nil(t, err2)
	})

	t.Run("returns error when trying to fetch briefing for nonexistant user", func(t *testing.T) {
		test.CleanupTables(t, pool)

		testBriefing := "this is a idk"

		profileID := test.SeedProfile(t, pool, embedding)
		visitorID := test.SeedVisitor(t, pool, profileID, "testvisitor", embedding)
		test.SeedBriefing(t, pool, profileID, visitorID, testBriefing)

		briefing, err1 := dbPool.FetchVisitorBriefing(profileID, 69)
		briefing, err2 := dbPool.FetchVisitorBriefing(69, visitorID)

		assert.Equal(t, "", briefing, "briefing returns empty str")
		assert.Equal(t, "error fetching briefing: no rows in result set", err1.Error())
		assert.Equal(t, "error fetching briefing: no rows in result set", err2.Error())
	})
}

func TestAddNewFaceForVisitor(t *testing.T) {
	pool := testDB.Pool
	dbPool := db.NewDBPool(pool)

	t.Run("returns error for invalid profileID, name, and embedding", func(t *testing.T) {
		test.CleanupTables(t, pool)
		validEmbedding := test.MakeEmbedding(0.1, 128)

		profileID := test.SeedProfile(t, pool, validEmbedding)
		var embedding face.Descriptor
		copy(embedding[:], validEmbedding)

		visitorID, err := dbPool.AddNewFaceForVisitor(-1, "valid name", embedding)
		assert.Equal(t, "profileID must be positive", err.Error())
		assert.Nil(t, visitorID)

		visitorID, err = dbPool.AddNewFaceForVisitor(profileID, "", embedding)
		assert.Equal(t, "name must be a non empty string", err.Error())
		assert.Nil(t, visitorID)

		zerosEmbedding := test.MakeEmbedding(0, 128)
		var invalidEmbedding face.Descriptor
		copy(invalidEmbedding[:], zerosEmbedding)

		visitorID, err = dbPool.AddNewFaceForVisitor(profileID, "valid name", invalidEmbedding)
		assert.Equal(t, "embedding cannot be all zeros", err.Error())
		assert.Nil(t, visitorID)
	})

	t.Run("successfully creates the new face embedding for visitor", func(t *testing.T) {
		test.CleanupTables(t, pool)

		validEmbedding := test.MakeEmbedding(0.1, 128)

		profileID := test.SeedProfile(t, pool, validEmbedding)
		var embedding face.Descriptor
		copy(embedding[:], validEmbedding)

		visitorID, err := dbPool.AddNewFaceForVisitor(profileID, "valid name", embedding)
		assert.Nil(t, err)

		embeddingFromDB := test.CheckVisitorEmbeddings(t, pool, profileID, "valid name")

		assert.EqualValues(t, embedding, embeddingFromDB)
		assert.Equal(t, int32(1), *visitorID, "first visitorID should have id 1")
	})

	t.Run("duplicate visitor embedding call for same patient doesnt error", func(t *testing.T) {
		test.CleanupTables(t, pool)

		validEmbedding1 := test.MakeEmbedding(0.1, 128)

		profileID := test.SeedProfile(t, pool, validEmbedding1)
		var embedding1 face.Descriptor
		copy(embedding1[:], validEmbedding1)

		visitorID, err := dbPool.AddNewFaceForVisitor(profileID, "valid name", embedding1)
		assert.Nil(t, err)
		assert.Equal(t, int32(1), *visitorID, "first visitorID should have id 1")

		embeddingFromDB := test.CheckVisitorEmbeddings(t, pool, profileID, "valid name")

		assert.EqualValues(t, embedding1, embeddingFromDB)

		// upsert value
		validEmbedding2 := test.MakeEmbedding(0.5, 128)
		var embedding2 face.Descriptor
		copy(embedding2[:], validEmbedding2)

		visitorID2, err := dbPool.AddNewFaceForVisitor(profileID, "valid name", embedding2)
		assert.Nil(t, err)

		upsertEmbedding := test.CheckVisitorEmbeddings(t, pool, profileID, "valid name")

		assert.EqualValues(t, embedding2, upsertEmbedding, "should update the embedding to new one")
		assert.EqualValues(t, visitorID2, visitorID, "Id should be the same in upsert")
	})
}

func TestAddNewFaceForUser(t *testing.T) {
	pool := testDB.Pool
	dbPool := db.NewDBPool(pool)

	t.Run("returns error and nil id for invalid embedding", func(t *testing.T) {
		test.CleanupTables(t, pool)

		zerosEmbedding := test.MakeEmbedding(0, 128)
		var invalidEmbedding face.Descriptor
		copy(invalidEmbedding[:], zerosEmbedding)

		id, err := dbPool.AddNewFaceForUser([]face.Descriptor{invalidEmbedding})
		assert.Equal(t, "embedding cannot be all zeros", err.Error())
		assert.Nil(t, id)
	})

	t.Run("successfully adds multiple face embedding for profile and creates unknown user placehold", func(t *testing.T) {
		test.CleanupTables(t, pool)

		embeddings := make([]face.Descriptor, 3)
		for i, val := range []float32{0.1, 0.2, 0.3} {
			copy(embeddings[i][:], test.MakeEmbedding(val, 128))
		}

		id, err := dbPool.AddNewFaceForUser(embeddings)
		assert.Nil(t, err)
		assert.NotNil(t, id)

		zerosEmbedding := test.MakeEmbedding(0, 128)
		var unknownVisEmbedding face.Descriptor
		copy(unknownVisEmbedding[:], zerosEmbedding)

		embeddingFromDB := test.CheckProfileEmbeddings(t, pool, *id)
		unknownVisEmbFromDB := test.CheckVisitorEmbeddings(t, pool, *id, "Unknown")

		assert.EqualValues(t, embeddings[0], embeddingFromDB)
		assert.EqualValues(t, unknownVisEmbFromDB, unknownVisEmbedding)
	})
}

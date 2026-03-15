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
	t.Run("returns error for negative patientID", func(t *testing.T) {
		test.CleanupTables(t, pool)

		patientID := test.SeedPatient(t, pool, test.MakeEmbedding(0.1, 128))
		_ = test.SeedVisitor(t, pool, patientID, "testvisitor", test.MakeEmbedding(0.1, 128))

		visitorEmbList, err := dbPool.FetchAllVisitorFaceEmbForPatient(-1)

		assert.Nil(t, visitorEmbList, "emb list returns nil")
		assert.Equal(t, "patientID must be positive", err.Error())
	})

	t.Run("returns list of visitor embeddings", func(t *testing.T) {
		test.CleanupTables(t, pool)

		patientID := test.SeedPatient(t, pool, test.MakeEmbedding(0.1, 128))

		visitorEmbs := []float32{0.1, 0.2, 0.3}
		visitorIDs := make([]int32, 3)
		for i, val := range visitorEmbs {
			visitorIDs[i] = test.SeedVisitor(
				t, pool, patientID, fmt.Sprintf("visitor%d", i+1), test.MakeEmbedding(val, 128),
			)
		}

		visitorEmbList, err := dbPool.FetchAllVisitorFaceEmbForPatient(patientID)

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

		patientID := test.SeedPatient(t, pool, embedding)
		_ = test.SeedVisitor(t, pool, patientID, "visitor", embedding)

		visitorEmbList, err := dbPool.FetchAllVisitorFaceEmbForPatient(23)

		assert.Nil(t, err)
		assert.Equal(t, 0, len(visitorEmbList))
	})
}

func TestFetchVisitorBriefing(t *testing.T) {
	pool := testDB.Pool
	dbPool := db.NewDBPool(pool)
	embedding := test.MakeEmbedding(0.1, 128)

	t.Run("returns error for negative patientID and visitorID", func(t *testing.T) {
		test.CleanupTables(t, pool)
		testBriefing := "this is a test"

		patientID := test.SeedPatient(t, pool, embedding)
		visitorID := test.SeedVisitor(t, pool, patientID, "testvisitor", embedding)
		test.SeedBriefing(t, pool, patientID, visitorID, testBriefing)

		briefing, err := dbPool.FetchVisitorBriefing(-1, visitorID)

		assert.Equal(t, "", briefing, "briefing returns empty str on error")
		assert.Equal(t, "patientID must be positive", err.Error())

		briefing, err = dbPool.FetchVisitorBriefing(patientID, -1)

		assert.Equal(t, "", briefing, "briefing returns empty str on error")
		assert.Equal(t, "visitorID must be positive", err.Error())
	})

	t.Run("returns the briefing for the correct visitor", func(t *testing.T) {
		test.CleanupTables(t, pool)
		
		testBriefing1 := "this is a test briefing 1"
		testBriefing2 := "this is a test briefing 2"

		patientID := test.SeedPatient(t, pool, embedding)
		visitorID1 := test.SeedVisitor(t, pool, patientID, "testvisitor1", embedding)
		visitorID2 := test.SeedVisitor(t, pool, patientID, "testvisitor2", embedding)

		test.SeedBriefing(t, pool, patientID, visitorID1, testBriefing1)
		test.SeedBriefing(t, pool, patientID, visitorID2, testBriefing2)

		briefing1, err1 := dbPool.FetchVisitorBriefing(patientID, visitorID1)
		briefing2, err2 := dbPool.FetchVisitorBriefing(patientID, visitorID2)

		assert.Equal(t, testBriefing1, briefing1, "briefing returned for correct visitor")
		assert.Nil(t, err1)

		assert.Equal(t, testBriefing2, briefing2, "briefing returned for correct visitor")
		assert.Nil(t, err2)
	})

	t.Run("returns error when trying to fetch briefing for nonexistant user", func(t *testing.T) {
		test.CleanupTables(t, pool)
		
		testBriefing := "this is a idk"

		patientID := test.SeedPatient(t, pool, embedding)
		visitorID := test.SeedVisitor(t, pool, patientID, "testvisitor", embedding)
		test.SeedBriefing(t, pool, patientID, visitorID, testBriefing)

		briefing, err1 := dbPool.FetchVisitorBriefing(patientID, 69)
		briefing, err2 := dbPool.FetchVisitorBriefing(69, visitorID)

		assert.Equal(t, "", briefing, "briefing returns empty str")
		assert.Equal(t, "error fetching briefing: no rows in result set", err1.Error())
		assert.Equal(t, "error fetching briefing: no rows in result set", err2.Error())
	})
}

func TestAddNewFaceForVisitor(t *testing.T) {
	pool := testDB.Pool
	dbPool := db.NewDBPool(pool)

	t.Run("returns error for invalid patientID, name, and embedding", func(t *testing.T) {
		test.CleanupTables(t, pool)
		validEmbedding := test.MakeEmbedding(0.1, 128)

		patientID := test.SeedPatient(t, pool, validEmbedding)
		var embedding face.Descriptor
		copy(embedding[:], validEmbedding)

		err := dbPool.AddNewFaceForVisitor(-1, "valid name", embedding)
		assert.Equal(t, "patientID must be positive", err.Error())

		err = dbPool.AddNewFaceForVisitor(patientID, "", embedding)
		assert.Equal(t, "name must be a non empty string", err.Error())

		zerosEmbedding := test.MakeEmbedding(0, 128)
		var invalidEmbedding face.Descriptor
		copy(invalidEmbedding[:], zerosEmbedding)

		err = dbPool.AddNewFaceForVisitor(patientID, "valid name", invalidEmbedding)
		assert.Equal(t, "embedding cannot be all zeros", err.Error())
	})

	t.Run("successfully creates the new face embedding for visitor", func(t *testing.T) {
		test.CleanupTables(t, pool)

		validEmbedding := test.MakeEmbedding(0.1, 128)

		patientID := test.SeedPatient(t, pool, validEmbedding)
		var embedding face.Descriptor
		copy(embedding[:], validEmbedding)

		err := dbPool.AddNewFaceForVisitor(patientID, "valid name", embedding)
		assert.Nil(t, err)

		embeddingFromDB := test.CheckVisitorEmbeddings(t, pool, patientID, "valid name")

		assert.EqualValues(t, embedding, embeddingFromDB)
	})

	t.Run("duplicate visitor embedding call for same patient doesnt error", func(t *testing.T) {
		test.CleanupTables(t, pool)

		validEmbedding1 := test.MakeEmbedding(0.1, 128)

		patientID := test.SeedPatient(t, pool, validEmbedding1)
		var embedding1 face.Descriptor
		copy(embedding1[:], validEmbedding1)

		err := dbPool.AddNewFaceForVisitor(patientID, "valid name", embedding1)
		assert.Nil(t, err)

		embeddingFromDB := test.CheckVisitorEmbeddings(t, pool, patientID, "valid name")

		assert.EqualValues(t, embedding1, embeddingFromDB)

		// upsert value
		validEmbedding2 := test.MakeEmbedding(0.5, 128)
		var embedding2 face.Descriptor
		copy(embedding2[:], validEmbedding2)

		err = dbPool.AddNewFaceForVisitor(patientID, "valid name", embedding2)
		assert.Nil(t, err)

		upsertEmbedding := test.CheckVisitorEmbeddings(t, pool, patientID, "valid name")

		assert.EqualValues(t, embedding2, upsertEmbedding, "should update the embedding to new one")
	})
}	
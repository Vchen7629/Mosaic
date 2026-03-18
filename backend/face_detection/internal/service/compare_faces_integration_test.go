//go:build integration

package service_test

import (
	"testing"

	"github.com/Kagami/go-face"
	"github.com/stretchr/testify/assert"
	"mosaic-face-detection.com/internal/service"
)

func TestCompareVisitorFaces(t *testing.T) {
	recPool, err := service.NewRecognizerPool(testModelsDir, 5)
	assert.NoError(t, err)
	defer recPool.Close()

	rec := recPool.Acquire()
	defer recPool.Release(rec)

	// Generate the embeddings for all photos upfront so i can pass them into my tests
	bonaEmb := getEmbedding(t, rec, "bona.jpg")
	bona2Emb := getEmbedding(t, rec, "bona2.jpg")
	bona3Emb := getEmbedding(t, rec, "bona3.jpg")
	man1Emb := getEmbedding(t, rec, "man1.jpg")
	eunseo1 := getEmbedding(t, rec, "eunseo1.jpg")

	t.Run("matches same person across diff photos", func(t *testing.T) {
		knownVisitors := []service.VisitorFaces{
			{ID: 1, Name: "Bona", Embedding: bonaEmb},
		}

		res2 := service.CompareVisitorFaces(rec, []face.Descriptor{bona2Emb}, knownVisitors)
		res3 := service.CompareVisitorFaces(rec, []face.Descriptor{bona3Emb}, knownVisitors)

		assert.Equal(t, int32(1), res2[0].ID, "bona2 should match bona's visitorID")
		assert.Equal(t, "Bona", res2[0].Name, "name should match")
		assert.Equal(t, int32(1), res3[0].ID, "bona3 should match bona's visitorID")
		assert.Equal(t, "Bona", res3[0].Name, "name should match")
	})

	t.Run("shouldnt match different people", func(t *testing.T) {
		knownVisitors := []service.VisitorFaces{
			{ID: 1, Name: "Bona", Embedding: bonaEmb},
		}

		man1Res := service.CompareVisitorFaces(rec, []face.Descriptor{man1Emb}, knownVisitors)

		assert.Equal(t, int32(-1), man1Res[0].ID, "man1 shouldn't match bona")
		assert.Equal(t, "", man1Res[0].Name, "unknown faces should have empty face")
	})

	t.Run("identifies correct visitor among multiple known visitors", func(t *testing.T) {
		knownVisitor := []service.VisitorFaces{
			{ID: 1, Name: "Bona", Embedding: bonaEmb},
			{ID: 2, Name: "Man1", Embedding: man1Emb},
			{ID: 3, Name: "Eunseo", Embedding: eunseo1},
		}

		// check if each person gets their own id back
		bonaRes := service.CompareVisitorFaces(rec, []face.Descriptor{bona2Emb}, knownVisitor)
		man1Res := service.CompareVisitorFaces(rec, []face.Descriptor{man1Emb}, knownVisitor)
		eunseoRes := service.CompareVisitorFaces(rec, []face.Descriptor{eunseo1}, knownVisitor)

		assert.Equal(t, int32(1), bonaRes[0].ID)
		assert.Equal(t, "Bona", bonaRes[0].Name, "name should match")
		assert.Equal(t, int32(2), man1Res[0].ID)
		assert.Equal(t, "Man1", man1Res[0].Name, "name should match")
		assert.Equal(t, int32(3), eunseoRes[0].ID)
		assert.Equal(t, "Eunseo", eunseoRes[0].Name, "name should match")
	})

	t.Run("returns -1 for all faces when no known visitors", func(t *testing.T) {
		res := service.CompareVisitorFaces(
			rec, []face.Descriptor{bonaEmb, man1Emb}, []service.VisitorFaces{},
		)

		assert.Equal(t, int32(-1), res[0].ID)
		assert.Equal(t, "", res[0].Name)
		assert.Equal(t, int32(-1), res[1].ID)
		assert.Equal(t, "", res[1].Name)
	})

	t.Run("returns empty map when no embeddings passed", func(t *testing.T) {
		knownVisitors := []service.VisitorFaces{{ID: 1, Name: "Bona", Embedding: bonaEmb}}

		res := service.CompareVisitorFaces(rec, []face.Descriptor{}, knownVisitors)

		assert.Empty(t, res)
	})

	t.Run("map keys are face indicies", func(t *testing.T) {
		knownVisitors := []service.VisitorFaces{{ID: 1, Name:"Bona", Embedding: bonaEmb}}
		embeddings := []face.Descriptor{bonaEmb, man1Emb, eunseo1}

		res := service.CompareVisitorFaces(rec, embeddings, knownVisitors)

		assert.Len(t, res, 3)
		for i := range embeddings {
			_, exists := res[i]
			assert.True(t, exists, "result should have key for face index %d", i)
		}
	})
}

func TestCompareProfileFaces(t *testing.T) {
	recPool, err := service.NewRecognizerPool(testModelsDir, 5)
	assert.NoError(t, err)
	defer recPool.Close()

	rec := recPool.Acquire()
	defer recPool.Release(rec)

	bonaEmb := getEmbedding(t, rec, "bona.jpg")
	bona2Emb := getEmbedding(t, rec, "bona2.jpg")
	bona3Emb := getEmbedding(t, rec, "bona3.jpg")
	man1Emb := getEmbedding(t, rec, "man1.jpg")
	eunseo1 := getEmbedding(t, rec, "eunseo1.jpg")

	t.Run("matches same person across diff photos", func(t *testing.T) {
		profiles := []service.ProfileFaces{
			{ID: 1, Embedding: bonaEmb},
		}

		id2, matched2 := service.CompareProfileFaces(rec, []face.Descriptor{bona2Emb}, profiles)
		id3, matched3 := service.CompareProfileFaces(rec, []face.Descriptor{bona3Emb}, profiles)

		assert.True(t, matched2, "bona2 should match a profile")
		assert.Equal(t, int32(1), id2, "bona2 should match bona's profileID")
		assert.True(t, matched3, "bona3 should match a profile")
		assert.Equal(t, int32(1), id3, "bona3 should match bona's profileID")
	})

	t.Run("returns false when face doesnt match any profile", func(t *testing.T) {
		profiles := []service.ProfileFaces{
			{ID: 1, Embedding: bonaEmb},
		}

		id, matched := service.CompareProfileFaces(rec, []face.Descriptor{man1Emb}, profiles)

		assert.False(t, matched, "man1 should not match bona's profile")
		assert.Equal(t, int32(0), id)
	})

	t.Run("returns false when profile embeddings list is empty", func(t *testing.T) {
		id, matched := service.CompareProfileFaces(rec, []face.Descriptor{bonaEmb}, []service.ProfileFaces{})

		assert.False(t, matched)
		assert.Equal(t, int32(0), id)
	})

	t.Run("returns false when embedding list is empty", func(t *testing.T) {
		profiles := []service.ProfileFaces{
			{ID: 1, Embedding: bonaEmb},
		}

		id, matched := service.CompareProfileFaces(rec, []face.Descriptor{}, profiles)

		assert.False(t, matched)
		assert.Equal(t, int32(0), id)
	})

	t.Run("only uses first embedding when multiple are passed", func(t *testing.T) {
		profiles := []service.ProfileFaces{
			{ID: 1, Embedding: bonaEmb},
		}

		// bona2 matches, man1 does not — only bona2 (index 0) should be evaluated
		id, matched := service.CompareProfileFaces(rec, []face.Descriptor{bona2Emb, man1Emb}, profiles)

		assert.True(t, matched)
		assert.Equal(t, int32(1), id)
	})

	t.Run("identifies correct profile among multiple profiles", func(t *testing.T) {
		profiles := []service.ProfileFaces{
			{ID: 1, Embedding: bonaEmb},
			{ID: 2, Embedding: man1Emb},
			{ID: 3, Embedding: eunseo1},
		}

		bonaID, bonaMat := service.CompareProfileFaces(rec, []face.Descriptor{bona2Emb}, profiles)
		man1ID, man1Mat := service.CompareProfileFaces(rec, []face.Descriptor{man1Emb}, profiles)
		eunseoID, eunseoMat := service.CompareProfileFaces(rec, []face.Descriptor{eunseo1}, profiles)

		assert.True(t, bonaMat)
		assert.Equal(t, int32(1), bonaID)
		assert.True(t, man1Mat)
		assert.Equal(t, int32(2), man1ID)
		assert.True(t, eunseoMat)
		assert.Equal(t, int32(3), eunseoID)
	})

	t.Run("does not match wrong profile among multiple profiles", func(t *testing.T) {
		profiles := []service.ProfileFaces{
			{ID: 1, Embedding: bonaEmb},
			{ID: 2, Embedding: eunseo1},
		}

		id, matched := service.CompareProfileFaces(rec, []face.Descriptor{man1Emb}, profiles)

		assert.False(t, matched)
		assert.Equal(t, int32(0), id)
	})

	t.Run("matches same person with only one profile registered", func(t *testing.T) {
		profiles := []service.ProfileFaces{
			{ID: 42, Embedding: bona3Emb},
		}

		id, matched := service.CompareProfileFaces(rec, []face.Descriptor{bonaEmb}, profiles)

		assert.True(t, matched)
		assert.Equal(t, int32(42), id)
	})
}

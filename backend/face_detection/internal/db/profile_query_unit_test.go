//go:build unit

package db

import (
	"math"
	"testing"

	"github.com/Kagami/go-face"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchProfileFaceEmbForID(t *testing.T) {
	dbPool := newTestDBPool()
	tests := []struct {
		name      string
		profileID int32
	}{
		{"returns error for 0 profileID", 0},
		{"returns error for negative profileID", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := dbPool.FetchProfileFaceEmbForID(tt.profileID)
			require.Error(t, err)
			assert.Equal(t, "profileID must be positive", err.Error())
		})
	}
}

func makeDescriptor(val float32) face.Descriptor {
	var d face.Descriptor
	d[0] = val
	return d
}

func makeNaNDescriptor() face.Descriptor {
	var d face.Descriptor
	d[0] = float32(math.NaN())
	return d
}

func makeInfDescriptor() face.Descriptor {
	var d face.Descriptor
	d[0] = float32(math.Inf(1))
	return d
}

func TestAddNewFaceForProfile_InvalidEmbeddings(t *testing.T) {
	dbPool := newTestDBPool()

	validEmb := makeDescriptor(0.5)
	var zeroEmb face.Descriptor

	tests := []struct {
		name       string
		embeddings []face.Descriptor
		errMsg     string
	}{
		{
			"returns error for single all-zero embedding",
			[]face.Descriptor{zeroEmb},
			"embedding cannot be all zeros",
		},
		{
			"returns error when second embedding in batch is all zeros",
			[]face.Descriptor{validEmb, zeroEmb},
			"embedding cannot be all zeros",
		},
		{
			"returns error for embedding containing NaN",
			[]face.Descriptor{makeNaNDescriptor()},
			"embedding contains NaN or infinity values",
		},
		{
			"returns error for embedding containing infinity",
			[]face.Descriptor{makeInfDescriptor()},
			"embedding contains NaN or infinity values",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := dbPool.AddNewFaceForProfile(tt.embeddings)
			require.Error(t, err)
			assert.Nil(t, result)
			assert.Equal(t, tt.errMsg, err.Error())
		})
	}
}

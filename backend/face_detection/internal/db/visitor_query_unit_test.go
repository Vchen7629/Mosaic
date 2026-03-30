//go:build unit

package db

import (
	"testing"

	"github.com/Kagami/go-face"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchAllVisitorData(t *testing.T) {
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
			_, err := dbPool.FetchAllVisitorData(tt.profileID)
			require.Error(t, err)
			assert.Equal(t, "profileID must be positive", err.Error())
		})
	}
}

func TestFetchVisitorBriefing(t *testing.T) {
	dbPool := newTestDBPool()
	tests := []struct {
		name                 string
		profileID, visitorID int32
		errMsg               string
	}{
		{"returns error for 0 profileID", 0, 1, "profileID must be positive"},
		{"returns error for negative profileID", -1, 1, "profileID must be positive"},
		{"returns error for 0 visitorID", 1, 0, "visitorID must be positive"},
		{"returns error for negative visitorID", 1, -1, "visitorID must be positive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := dbPool.FetchVisitorBriefing(tt.profileID, tt.visitorID)
			require.Error(t, err)
			assert.Equal(t, tt.errMsg, err.Error())
		})
	}
}

func TestAddNewFaceForVisitor(t *testing.T) {
	dbPool := newTestDBPool()

	validEmb := makeDescriptor(0.5)
	var zeroEmb face.Descriptor

	tests := []struct {
		testName  string
		profileID int32
		name      string
		embedding face.Descriptor
		errMsg    string
	}{
		{"returns error for 0 profileID", 0, "hi", validEmb, "profileID must be positive"},
		{"returns error for negative profileID", -1, "hi", validEmb, "profileID must be positive"},
		{"returns error for empty name", 1, "", validEmb, "name must be a non empty string"},
		{"returns error for all-zero embedding", 1, "hi", zeroEmb, "embedding cannot be all zeros"},
		{"returns error for embedding containing NaN", 1, "hi", makeNaNDescriptor(), "embedding contains NaN or infinity values"},
		{"returns error for embedding containing infinity", 1, "hi", makeInfDescriptor(), "embedding contains NaN or infinity values"},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			_, err := dbPool.AddNewFaceForVisitor(tt.profileID, tt.name, tt.embedding)
			require.Error(t, err)
			assert.Equal(t, tt.errMsg, err.Error())
		})
	}
}

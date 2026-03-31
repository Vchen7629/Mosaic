package cache

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//
func TestCreateNewProfileSession_NilClient(t *testing.T) {
	err := CreateNewProfileSession(context.Background(), 1, nil, "some-token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no cache client provided")
}

func TestFetchProfileIDFromCache_NilClient(t *testing.T) {
	result, err := FetchProfileIDFromCache(context.Background(), nil, "some-token")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no cache client provided")
}

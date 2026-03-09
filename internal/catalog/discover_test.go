package catalog

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscoverRegistries_FindsCatalogs(t *testing.T) {
	runner := func(org string) ([]byte, error) {
		assert.Equal(t, "acme", org)
		packages := []ghPackage{
			{Name: "apis-catalog"},
			{Name: "shared-schemas-catalog"},
			{Name: "some-other-image"},
			{Name: "apx"}, // the CLI itself — should be skipped
		}
		return json.Marshal(packages)
	}

	sources := discoverRegistries("acme", runner)
	require.Len(t, sources, 2)
	assert.Contains(t, sources[0].Name(), "apis-catalog")
	assert.Contains(t, sources[1].Name(), "shared-schemas-catalog")
}

func TestDiscoverRegistries_NoCatalogs(t *testing.T) {
	runner := func(org string) ([]byte, error) {
		packages := []ghPackage{
			{Name: "my-app"},
			{Name: "other-service"},
		}
		return json.Marshal(packages)
	}

	sources := discoverRegistries("acme", runner)
	assert.Empty(t, sources)
}

func TestDiscoverRegistries_GHError(t *testing.T) {
	runner := func(org string) ([]byte, error) {
		return nil, assert.AnError
	}

	sources := discoverRegistries("acme", runner)
	assert.Nil(t, sources, "should silently return nil on gh error")
}

func TestDiscoverRegistries_InvalidJSON(t *testing.T) {
	runner := func(org string) ([]byte, error) {
		return []byte("not json"), nil
	}

	sources := discoverRegistries("acme", runner)
	assert.Nil(t, sources)
}

func TestDiscoverRegistries_EmptyRepoName(t *testing.T) {
	runner := func(org string) ([]byte, error) {
		// A package named exactly "-catalog" would yield empty repo name
		packages := []ghPackage{
			{Name: "-catalog"},
		}
		return json.Marshal(packages)
	}

	sources := discoverRegistries("acme", runner)
	assert.Empty(t, sources, "empty repo name should be skipped")
}

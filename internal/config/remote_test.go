package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFetchRemoteImportRoot_RawSuccess(t *testing.T) {
	origHTTP := httpGetFn
	origGH := ghAPIFn
	defer func() { httpGetFn = origHTTP; ghAPIFn = origGH }()

	httpGetFn = func(url string) ([]byte, error) {
		return []byte("version: 1\nimport_root: go.acme.dev/apis\norg: acme\n"), nil
	}
	ghAPIFn = func(endpoint string) ([]byte, error) {
		t.Fatal("gh api should not be called when raw succeeds")
		return nil, nil
	}

	result := FetchRemoteImportRoot("acme", "apis")
	assert.Equal(t, "go.acme.dev/apis", result)
}

func TestFetchRemoteImportRoot_RawNoImportRoot(t *testing.T) {
	origHTTP := httpGetFn
	origGH := ghAPIFn
	defer func() { httpGetFn = origHTTP; ghAPIFn = origGH }()

	httpGetFn = func(url string) ([]byte, error) {
		return []byte("version: 1\norg: acme\nrepo: apis\n"), nil
	}
	ghAPIFn = func(endpoint string) ([]byte, error) {
		return nil, fmt.Errorf("not found")
	}

	result := FetchRemoteImportRoot("acme", "apis")
	assert.Equal(t, "", result)
}

func TestFetchRemoteImportRoot_RawFails_GHSuccess(t *testing.T) {
	origHTTP := httpGetFn
	origGH := ghAPIFn
	defer func() { httpGetFn = origHTTP; ghAPIFn = origGH }()

	httpGetFn = func(url string) ([]byte, error) {
		return nil, fmt.Errorf("HTTP 404")
	}

	yamlContent := "version: 1\nimport_root: go.acme.dev/apis\norg: acme\n"
	encoded := base64.StdEncoding.EncodeToString([]byte(yamlContent))
	ghAPIFn = func(endpoint string) ([]byte, error) {
		return []byte(fmt.Sprintf(`{"content":"%s","encoding":"base64"}`, encoded)), nil
	}

	result := FetchRemoteImportRoot("acme", "apis")
	assert.Equal(t, "go.acme.dev/apis", result)
}

func TestFetchRemoteImportRoot_RemoteFails_CachedCatalog(t *testing.T) {
	origHTTP := httpGetFn
	origGH := ghAPIFn
	defer func() { httpGetFn = origHTTP; ghAPIFn = origGH }()

	httpGetFn = func(url string) ([]byte, error) {
		return nil, fmt.Errorf("network error")
	}
	ghAPIFn = func(endpoint string) ([]byte, error) {
		return nil, fmt.Errorf("gh not installed")
	}

	// Create a temp cache directory with a catalog
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome) // Windows: os.UserHomeDir uses USERPROFILE

	cacheDir := filepath.Join(tmpHome, ".cache", "apx", "catalogs", "acme", "apis")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	catalogYAML := "version: 1\norg: acme\nrepo: apis\nimport_root: go.acme.dev/apis\nmodules: []\n"
	if err := os.WriteFile(filepath.Join(cacheDir, "catalog.yaml"), []byte(catalogYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	result := FetchRemoteImportRoot("acme", "apis")
	assert.Equal(t, "go.acme.dev/apis", result)
}

func TestFetchRemoteImportRoot_AllFail(t *testing.T) {
	origHTTP := httpGetFn
	origGH := ghAPIFn
	defer func() { httpGetFn = origHTTP; ghAPIFn = origGH }()

	httpGetFn = func(url string) ([]byte, error) {
		return nil, fmt.Errorf("network error")
	}
	ghAPIFn = func(endpoint string) ([]byte, error) {
		return nil, fmt.Errorf("gh not installed")
	}

	// No cached catalog either (temp HOME has nothing)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome) // Windows: os.UserHomeDir uses USERPROFILE

	result := FetchRemoteImportRoot("acme", "apis")
	assert.Equal(t, "", result)
}

func TestFetchRemoteImportRoot_MalformedYAML(t *testing.T) {
	origHTTP := httpGetFn
	origGH := ghAPIFn
	defer func() { httpGetFn = origHTTP; ghAPIFn = origGH }()

	httpGetFn = func(url string) ([]byte, error) {
		return []byte("{{not valid yaml"), nil
	}
	ghAPIFn = func(endpoint string) ([]byte, error) {
		return nil, fmt.Errorf("not found")
	}

	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome) // Windows: os.UserHomeDir uses USERPROFILE

	result := FetchRemoteImportRoot("acme", "apis")
	assert.Equal(t, "", result)
}

func TestFetchRemoteImportRoot_RawURL(t *testing.T) {
	origHTTP := httpGetFn
	origGH := ghAPIFn
	defer func() { httpGetFn = origHTTP; ghAPIFn = origGH }()

	var capturedURL string
	httpGetFn = func(url string) ([]byte, error) {
		capturedURL = url
		return []byte("import_root: go.example.dev/apis"), nil
	}
	ghAPIFn = func(endpoint string) ([]byte, error) {
		return nil, fmt.Errorf("unused")
	}

	FetchRemoteImportRoot("myorg", "apis")
	assert.Equal(t, "https://raw.githubusercontent.com/myorg/apis/HEAD/apx.yaml", capturedURL)
}

func TestFetchRemoteImportRoot_GHEndpoint(t *testing.T) {
	origHTTP := httpGetFn
	origGH := ghAPIFn
	defer func() { httpGetFn = origHTTP; ghAPIFn = origGH }()

	httpGetFn = func(url string) ([]byte, error) {
		return nil, fmt.Errorf("404")
	}

	var capturedEndpoint string
	ghAPIFn = func(endpoint string) ([]byte, error) {
		capturedEndpoint = endpoint
		return nil, fmt.Errorf("not found")
	}

	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome) // Windows: os.UserHomeDir uses USERPROFILE

	FetchRemoteImportRoot("myorg", "apis")
	assert.Equal(t, "repos/myorg/apis/contents/apx.yaml", capturedEndpoint)
}

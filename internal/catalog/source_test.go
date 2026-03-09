package catalog

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// LocalSource
// ---------------------------------------------------------------------------

func TestLocalSource_Load(t *testing.T) {
	dir := t.TempDir()
	catPath := filepath.Join(dir, "catalog.yaml")

	content := `version: 1
org: acme
repo: apis
modules:
  - id: proto/payments/ledger/v1
    format: proto
    domain: payments
    api_line: v1
    path: proto/payments/ledger/v1
`
	require.NoError(t, os.WriteFile(catPath, []byte(content), 0644))

	src := &LocalSource{Path: catPath}
	cat, err := src.Load()
	require.NoError(t, err)
	assert.Equal(t, "acme", cat.Org)
	assert.Equal(t, "apis", cat.Repo)
	require.Len(t, cat.Modules, 1)
	assert.Equal(t, "proto/payments/ledger/v1", cat.Modules[0].ID)
}

func TestLocalSource_MissingFile(t *testing.T) {
	src := &LocalSource{Path: "/nonexistent/catalog.yaml"}
	cat, err := src.Load()
	require.NoError(t, err, "missing file should not be an error")
	assert.Equal(t, 1, cat.Version)
	assert.Empty(t, cat.Modules)
}

func TestLocalSource_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	catPath := filepath.Join(dir, "catalog.yaml")
	require.NoError(t, os.WriteFile(catPath, []byte(":::bad yaml"), 0644))

	src := &LocalSource{Path: catPath}
	_, err := src.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse catalog")
}

func TestLocalSource_Name(t *testing.T) {
	src := &LocalSource{Path: "/some/path/catalog.yaml"}
	assert.Equal(t, "/some/path/catalog.yaml", src.Name())
}

// ---------------------------------------------------------------------------
// HTTPSource
// ---------------------------------------------------------------------------

func TestHTTPSource_Load(t *testing.T) {
	body := `version: 1
org: acme
repo: apis
modules:
  - id: proto/billing/invoices/v1
    format: proto
    domain: billing
    api_line: v1
    path: proto/billing/invoices/v1
`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-yaml")
		w.Write([]byte(body))
	}))
	defer ts.Close()

	src := &HTTPSource{URL: ts.URL}
	cat, err := src.Load()
	require.NoError(t, err)
	assert.Equal(t, "acme", cat.Org)
	require.Len(t, cat.Modules, 1)
	assert.Equal(t, "proto/billing/invoices/v1", cat.Modules[0].ID)
}

func TestHTTPSource_NotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	src := &HTTPSource{URL: ts.URL}
	_, err := src.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 404")
}

func TestHTTPSource_InvalidYAML(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(":::bad yaml"))
	}))
	defer ts.Close()

	src := &HTTPSource{URL: ts.URL}
	_, err := src.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse remote catalog")
}

func TestHTTPSource_Name(t *testing.T) {
	src := &HTTPSource{URL: "https://example.com/catalog.yaml"}
	assert.Equal(t, "https://example.com/catalog.yaml", src.Name())
}

// ---------------------------------------------------------------------------
// SourceFor
// ---------------------------------------------------------------------------

func TestSourceFor_Local(t *testing.T) {
	src := SourceFor("catalog/catalog.yaml")
	_, ok := src.(*LocalSource)
	assert.True(t, ok, "should return LocalSource for a file path")
}

func TestSourceFor_HTTP(t *testing.T) {
	src := SourceFor("https://example.com/catalog.yaml")
	_, ok := src.(*HTTPSource)
	assert.True(t, ok, "should return HTTPSource for an HTTPS URL")
}

func TestSourceFor_HTTPInsecure(t *testing.T) {
	src := SourceFor("http://localhost:8080/catalog.yaml")
	_, ok := src.(*HTTPSource)
	assert.True(t, ok, "should return HTTPSource for an HTTP URL")
}

package catalog

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// RegistrySource — pull tests
// ---------------------------------------------------------------------------

func TestRegistrySource_Load(t *testing.T) {
	// Build a catalog and package it as the registry would serve it.
	catalogYAML := `version: 1
org: acme
repo: apis
modules:
  - id: proto/payments/ledger/v1
    format: proto
    domain: payments
    api_line: v1
    path: proto/payments/ledger/v1
`
	layerData, err := createTarGz("catalog.yaml", []byte(catalogYAML))
	require.NoError(t, err)
	layerDigest := fmt.Sprintf("sha256:%x", sha256.Sum256(layerData))

	configData := []byte("{}")
	configDigest := fmt.Sprintf("sha256:%x", sha256.Sum256(configData))

	manifest := ociManifest{
		SchemaVersion: 2,
		MediaType:     "application/vnd.oci.image.manifest.v1+json",
		Config: ociDescriptor{
			MediaType: "application/vnd.oci.image.config.v1+json",
			Digest:    configDigest,
			Size:      int64(len(configData)),
		},
		Layers: []ociDescriptor{
			{
				MediaType: "application/vnd.oci.image.layer.v1.tar+gzip",
				Digest:    layerDigest,
				Size:      int64(len(layerData)),
			},
		},
	}
	manifestJSON, _ := json.Marshal(manifest)

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/manifests/"):
			w.Header().Set("Content-Type", "application/vnd.oci.image.manifest.v1+json")
			w.Write(manifestJSON)
		case strings.Contains(r.URL.Path, "/blobs/"+layerDigest):
			w.Write(layerData)
		case strings.Contains(r.URL.Path, "/blobs/"+configDigest):
			w.Write(configData)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	// Extract host from test server URL (strip https://)
	host := strings.TrimPrefix(ts.URL, "https://")

	src := &RegistrySource{
		Org:        "acme",
		Repo:       "apis",
		Host:       host,
		GHTokenFn:  func() (string, error) { return "test-token", nil },
		HTTPClient: ts.Client(),
	}

	cat, err := src.Load()
	require.NoError(t, err)
	assert.Equal(t, "acme", cat.Org)
	assert.Equal(t, "apis", cat.Repo)
	require.Len(t, cat.Modules, 1)
	assert.Equal(t, "proto/payments/ledger/v1", cat.Modules[0].ID)
}

func TestRegistrySource_Load_RawYAMLLayer(t *testing.T) {
	// Test with a non-gzipped YAML layer (direct data)
	catalogYAML := `version: 1
org: acme
repo: apis
modules:
  - id: proto/orders/v1
    format: proto
    path: proto/orders/v1
`
	digest := fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(catalogYAML)))

	configData := []byte("{}")
	configDigest := fmt.Sprintf("sha256:%x", sha256.Sum256(configData))

	manifest := ociManifest{
		SchemaVersion: 2,
		MediaType:     "application/vnd.oci.image.manifest.v1+json",
		Config: ociDescriptor{
			MediaType: "application/vnd.oci.image.config.v1+json",
			Digest:    configDigest,
			Size:      int64(len(configData)),
		},
		Layers: []ociDescriptor{
			{
				MediaType: "application/vnd.oci.image.layer.v1.tar+gzip",
				Digest:    digest,
				Size:      int64(len(catalogYAML)),
			},
		},
	}
	manifestJSON, _ := json.Marshal(manifest)

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/manifests/"):
			w.Write(manifestJSON)
		case strings.Contains(r.URL.Path, "/blobs/"+digest):
			w.Write([]byte(catalogYAML))
		case strings.Contains(r.URL.Path, "/blobs/"+configDigest):
			w.Write(configData)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	host := strings.TrimPrefix(ts.URL, "https://")
	src := &RegistrySource{
		Org:        "acme",
		Repo:       "apis",
		Host:       host,
		GHTokenFn:  func() (string, error) { return "test-token", nil },
		HTTPClient: ts.Client(),
	}

	cat, err := src.Load()
	require.NoError(t, err)
	require.Len(t, cat.Modules, 1)
	assert.Equal(t, "proto/orders/v1", cat.Modules[0].ID)
}

func TestRegistrySource_Load_ManifestNotFound(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	host := strings.TrimPrefix(ts.URL, "https://")
	src := &RegistrySource{
		Org:        "acme",
		Repo:       "apis",
		Host:       host,
		GHTokenFn:  func() (string, error) { return "test-token", nil },
		HTTPClient: ts.Client(),
	}

	_, err := src.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 404")
}

func TestRegistrySource_Load_AuthFailure(t *testing.T) {
	src := &RegistrySource{
		Org:  "acme",
		Repo: "apis",
		GHTokenFn: func() (string, error) {
			return "", fmt.Errorf("gh not installed")
		},
	}

	_, err := src.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "registry auth")
}

func TestRegistrySource_Name(t *testing.T) {
	src := &RegistrySource{Org: "acme", Repo: "apis"}
	assert.Equal(t, "ghcr.io/acme/apis-catalog:latest", src.Name())
}

func TestRegistrySource_Name_Custom(t *testing.T) {
	src := &RegistrySource{Org: "acme", Repo: "apis", Host: "registry.example.com", Tag: "v2"}
	assert.Equal(t, "registry.example.com/acme/apis-catalog:v2", src.Name())
}

// ---------------------------------------------------------------------------
// PushCatalog — push tests
// ---------------------------------------------------------------------------

func TestPushCatalog(t *testing.T) {
	cat := &Catalog{
		Version: 1,
		Org:     "acme",
		Repo:    "apis",
		Modules: []Module{
			{ID: "proto/payments/ledger/v1", Format: "proto", Path: "proto/payments/ledger/v1"},
		},
	}

	var (
		uploadedBlobs    = map[string][]byte{}
		uploadedManifest []byte
	)

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		// Blob upload initiation
		case r.Method == "POST" && strings.Contains(r.URL.Path, "/blobs/uploads/"):
			w.Header().Set("Location", "https://"+r.Host+r.URL.Path+"upload-session-id")
			w.WriteHeader(http.StatusAccepted)

		// Blob upload completion
		case r.Method == "PUT" && strings.Contains(r.URL.Path, "/blobs/uploads/"):
			digest := r.URL.Query().Get("digest")
			body, _ := readAll(r.Body)
			uploadedBlobs[digest] = body
			w.WriteHeader(http.StatusCreated)

		// Manifest push
		case r.Method == "PUT" && strings.Contains(r.URL.Path, "/manifests/"):
			body, _ := readAll(r.Body)
			uploadedManifest = body
			w.WriteHeader(http.StatusCreated)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	host := strings.TrimPrefix(ts.URL, "https://")

	err := PushCatalog(cat, PushOptions{
		Org:        "acme",
		Repo:       "apis",
		Host:       host,
		GHTokenFn:  func() (string, error) { return "test-token", nil },
		HTTPClient: ts.Client(),
	})
	require.NoError(t, err)

	// Verify blobs were uploaded (config + layer)
	assert.Len(t, uploadedBlobs, 2, "expected config blob + layer blob")

	// Verify manifest was pushed
	require.NotNil(t, uploadedManifest, "manifest should have been pushed")
	var m ociManifest
	require.NoError(t, json.Unmarshal(uploadedManifest, &m))
	assert.Equal(t, 2, m.SchemaVersion)
	assert.Len(t, m.Layers, 1)
	assert.Equal(t, "catalog.yaml", m.Layers[0].Annotations["org.opencontainers.image.title"])
	assert.Equal(t, "https://github.com/acme/apis", m.Annotations["org.opencontainers.image.source"])
	assert.Equal(t, "acme", m.Annotations["dev.apx.catalog.org"])
	assert.Equal(t, "1", m.Annotations["dev.apx.catalog.module_count"])
}

func TestPushCatalog_AuthFailure(t *testing.T) {
	cat := &Catalog{Version: 1, Org: "acme", Repo: "apis"}

	err := PushCatalog(cat, PushOptions{
		Org:  "acme",
		Repo: "apis",
		GHTokenFn: func() (string, error) {
			return "", fmt.Errorf("gh not installed")
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "registry auth")
}

// ---------------------------------------------------------------------------
// tar.gz round-trip
// ---------------------------------------------------------------------------

func TestCreateAndExtractTarGz(t *testing.T) {
	content := []byte("hello world")
	archive, err := createTarGz("test.txt", content)
	require.NoError(t, err)

	extracted, err := extractFromTarGz(archive, "test.txt")
	require.NoError(t, err)
	assert.Equal(t, content, extracted)
}

func TestExtractFromTarGz_FileNotFound(t *testing.T) {
	archive, err := createTarGz("other.txt", []byte("data"))
	require.NoError(t, err)

	_, err = extractFromTarGz(archive, "missing.txt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestIsGzipped(t *testing.T) {
	gz, _ := createTarGz("f", []byte("x"))
	assert.True(t, isGzipped(gz))
	assert.False(t, isGzipped([]byte("not gzipped")))
	assert.False(t, isGzipped([]byte{}))
	assert.False(t, isGzipped([]byte{0x1f}))
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func readAll(r interface{ Read([]byte) (int, error) }) ([]byte, error) {
	var buf []byte
	tmp := make([]byte, 4096)
	for {
		n, err := r.Read(tmp)
		buf = append(buf, tmp[:n]...)
		if err != nil {
			break
		}
	}
	return buf, nil
}

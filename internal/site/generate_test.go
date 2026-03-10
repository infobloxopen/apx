package site

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerate_CreatesDirectoryStructure(t *testing.T) {
	tmpDir := t.TempDir()

	data := &SiteData{
		Org:         "acme",
		Repo:        "apis",
		GeneratedAt: "2026-03-09T00:00:00Z",
		APIs:        []APIEntry{},
	}

	err := Generate(data, tmpDir, "")
	require.NoError(t, err)

	// Verify expected files exist.
	assert.FileExists(t, filepath.Join(tmpDir, "index.html"))
	assert.FileExists(t, filepath.Join(tmpDir, "data", "index.json"))
	assert.FileExists(t, filepath.Join(tmpDir, "assets", "app.js"))
	assert.FileExists(t, filepath.Join(tmpDir, "assets", "style.css"))
}

func TestGenerate_IndexJSONContent(t *testing.T) {
	tmpDir := t.TempDir()

	data := &SiteData{
		Org:         "acme",
		Repo:        "apis",
		ImportRoot:  "go.acme.dev/apis",
		GeneratedAt: "2026-03-09T00:00:00Z",
		APIs: []APIEntry{
			{
				ID:        "proto/payments/ledger/v1",
				Format:    "proto",
				Domain:    "payments",
				Name:      "ledger",
				Line:      "v1",
				Lifecycle: "stable",
			},
		},
	}

	err := Generate(data, tmpDir, "")
	require.NoError(t, err)

	// Read and parse index.json.
	raw, err := os.ReadFile(filepath.Join(tmpDir, "data", "index.json"))
	require.NoError(t, err)

	var loaded SiteData
	err = json.Unmarshal(raw, &loaded)
	require.NoError(t, err)

	assert.Equal(t, "acme", loaded.Org)
	assert.Equal(t, "apis", loaded.Repo)
	assert.Equal(t, "go.acme.dev/apis", loaded.ImportRoot)
	require.Len(t, loaded.APIs, 1)
	assert.Equal(t, "proto/payments/ledger/v1", loaded.APIs[0].ID)
	assert.Equal(t, "stable", loaded.APIs[0].Lifecycle)
}

func TestGenerate_BasePathTemplating(t *testing.T) {
	tmpDir := t.TempDir()

	data := &SiteData{
		Org:         "acme",
		Repo:        "apis",
		GeneratedAt: "2026-03-09T00:00:00Z",
		APIs:        []APIEntry{},
	}

	err := Generate(data, tmpDir, "/my-apis/catalog")
	require.NoError(t, err)

	// Read index.html and verify base path was templated.
	html, err := os.ReadFile(filepath.Join(tmpDir, "index.html"))
	require.NoError(t, err)

	assert.Contains(t, string(html), `<base href="/my-apis/catalog/">`)
	assert.NotContains(t, string(html), "{{BASE_PATH}}")
}

func TestGenerate_EmptyBasePath(t *testing.T) {
	tmpDir := t.TempDir()

	data := &SiteData{
		Org:         "acme",
		Repo:        "apis",
		GeneratedAt: "2026-03-09T00:00:00Z",
		APIs:        []APIEntry{},
	}

	err := Generate(data, tmpDir, "")
	require.NoError(t, err)

	html, err := os.ReadFile(filepath.Join(tmpDir, "index.html"))
	require.NoError(t, err)

	// Empty base path should result in <base href="/">
	assert.Contains(t, string(html), `<base href="/">`)
}

func TestGenerate_MultipleAPIs(t *testing.T) {
	tmpDir := t.TempDir()

	data := &SiteData{
		Org:         "acme",
		Repo:        "apis",
		GeneratedAt: "2026-03-09T00:00:00Z",
		APIs: []APIEntry{
			{ID: "proto/payments/ledger/v1", Format: "proto"},
			{ID: "openapi/billing/invoices/v2", Format: "openapi"},
			{ID: "avro/events/clicks/v1", Format: "avro"},
		},
	}

	err := Generate(data, tmpDir, "")
	require.NoError(t, err)

	raw, err := os.ReadFile(filepath.Join(tmpDir, "data", "index.json"))
	require.NoError(t, err)

	var loaded SiteData
	err = json.Unmarshal(raw, &loaded)
	require.NoError(t, err)

	assert.Len(t, loaded.APIs, 3)
}

func TestGenerate_NestedOutputDir(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "deep", "nested", "output")

	data := &SiteData{
		Org:         "acme",
		Repo:        "apis",
		GeneratedAt: "2026-03-09T00:00:00Z",
		APIs:        []APIEntry{},
	}

	err := Generate(data, outputDir, "")
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(outputDir, "index.html"))
	assert.FileExists(t, filepath.Join(outputDir, "data", "index.json"))
}

func TestGenerate_LanguageCoordsInJSON(t *testing.T) {
	tmpDir := t.TempDir()

	data := &SiteData{
		Org:         "acme",
		Repo:        "apis",
		GeneratedAt: "2026-03-09T00:00:00Z",
		APIs: []APIEntry{
			{
				ID:     "proto/payments/ledger/v1",
				Format: "proto",
				Languages: map[string][]LanguageCoord{
					"go": {
						{Label: "Go module", Value: "github.com/acme/apis/proto/payments/ledger"},
						{Label: "Go import", Value: "github.com/acme/apis/proto/payments/ledger/v1"},
					},
					"python": {
						{Label: "Py dist", Value: "acme-payments-ledger-v1"},
						{Label: "Py import", Value: "acme_apis.payments.ledger.v1"},
					},
				},
			},
		},
	}

	err := Generate(data, tmpDir, "")
	require.NoError(t, err)

	raw, err := os.ReadFile(filepath.Join(tmpDir, "data", "index.json"))
	require.NoError(t, err)

	var loaded SiteData
	err = json.Unmarshal(raw, &loaded)
	require.NoError(t, err)

	require.Len(t, loaded.APIs, 1)
	goCoords := loaded.APIs[0].Languages["go"]
	require.Len(t, goCoords, 2)
	assert.Equal(t, "Go module", goCoords[0].Label)
	assert.Equal(t, "github.com/acme/apis/proto/payments/ledger", goCoords[0].Value)

	pyCoords := loaded.APIs[0].Languages["python"]
	require.Len(t, pyCoords, 2)
	assert.Equal(t, "Py dist", pyCoords[0].Label)
}

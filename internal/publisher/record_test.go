package publisher

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/infobloxopen/apx/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewReleaseRecord(t *testing.T) {
	api := &config.APIIdentity{
		ID: "proto/payments/ledger/v1", Format: "proto",
		Domain: "payments", Name: "ledger", Line: "v1",
		Lifecycle: "beta",
	}
	source := &config.SourceIdentity{
		Repo: "github.com/acme/apis",
		Path: "proto/payments/ledger/v1",
	}
	langs := map[string]config.LanguageCoords{
		"go": {
			Module: "github.com/acme/apis/proto/payments/ledger",
			Import: "github.com/acme/apis/proto/payments/ledger/v1",
		},
	}

	m := NewManifest(api, source, langs, "v1.2.0-beta.1", "github.com/acme/apis")
	m.SourceCommit = "abc123"
	m.PreparedAt = "2025-01-01T00:00:00Z"
	m.SubmittedAt = "2025-01-01T00:01:00Z"

	record := NewReleaseRecord(m)

	assert.Equal(t, "1", record.SchemaVersion)
	assert.Equal(t, "release-record", record.Kind)
	assert.Equal(t, "proto/payments/ledger/v1", record.APIID)
	assert.Equal(t, "proto", record.Format)
	assert.Equal(t, "payments", record.Domain)
	assert.Equal(t, "ledger", record.Name)
	assert.Equal(t, "v1", record.Line)
	assert.Equal(t, "beta", record.Lifecycle)
	assert.Equal(t, "github.com/acme/apis", record.SourceRepo)
	assert.Equal(t, "abc123", record.SourceCommit)
	assert.Equal(t, "v1.2.0-beta.1", record.Version)
	assert.Equal(t, "proto/payments/ledger/v1/v1.2.0-beta.1", record.Tag)
	assert.Equal(t, "github.com/acme/apis/proto/payments/ledger", record.GoModule)
	assert.Equal(t, "github.com/acme/apis/proto/payments/ledger/v1", record.GoImport)
	assert.Equal(t, "2025-01-01T00:00:00Z", record.PreparedAt)
	assert.Equal(t, "2025-01-01T00:01:00Z", record.SubmittedAt)
	assert.NotEmpty(t, record.FinalizedAt)
	assert.False(t, record.CatalogUpdated)
}

func TestReleaseRecord_AddArtifact(t *testing.T) {
	record := &ReleaseRecord{Kind: "release-record"}

	record.AddArtifact("go-module", "github.com/acme/apis/proto/payments/ledger", "v1.2.0-beta.1", "published")
	record.AddArtifact("proto-descriptor", "ledger.desc", "", "skipped")

	require.Len(t, record.Artifacts, 2)
	assert.Equal(t, "go-module", record.Artifacts[0].Type)
	assert.Equal(t, "published", record.Artifacts[0].Status)
	assert.Equal(t, "proto-descriptor", record.Artifacts[1].Type)
	assert.Equal(t, "skipped", record.Artifacts[1].Status)
}

func TestReleaseRecord_DetectCI_NoCI(t *testing.T) {
	record := &ReleaseRecord{Kind: "release-record"}
	// Clear CI env vars to test no-CI detection
	origGHA := os.Getenv("GITHUB_ACTIONS")
	origGitlab := os.Getenv("GITLAB_CI")
	origJenkins := os.Getenv("JENKINS_URL")
	os.Unsetenv("GITHUB_ACTIONS")
	os.Unsetenv("GITLAB_CI")
	os.Unsetenv("JENKINS_URL")
	defer func() {
		if origGHA != "" {
			os.Setenv("GITHUB_ACTIONS", origGHA)
		}
		if origGitlab != "" {
			os.Setenv("GITLAB_CI", origGitlab)
		}
		if origJenkins != "" {
			os.Setenv("JENKINS_URL", origJenkins)
		}
	}()

	record.DetectCI()
	assert.Empty(t, record.CIProvider)
}

func TestWriteReadReleaseRecord(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "release-record.yaml")

	original := &ReleaseRecord{
		SchemaVersion:  "1",
		Kind:           "release-record",
		APIID:          "proto/payments/ledger/v1",
		Format:         "proto",
		Domain:         "payments",
		Name:           "ledger",
		Line:           "v1",
		Lifecycle:      "beta",
		SourceRepo:     "github.com/acme/service",
		SourcePath:     "proto/payments/ledger/v1",
		SourceCommit:   "abc123",
		Version:        "v1.2.0-beta.1",
		Tag:            "proto/payments/ledger/v1/v1.2.0-beta.1",
		CanonicalRepo:  "github.com/acme/apis",
		CanonicalPath:  "proto/payments/ledger/v1",
		GoModule:       "github.com/acme/apis/proto/payments/ledger",
		GoImport:       "github.com/acme/apis/proto/payments/ledger/v1",
		CatalogUpdated: true,
		CatalogPath:    "catalog.yaml",
		FinalizedAt:    "2025-01-01T00:02:00Z",
		Artifacts: []ReleaseArtifact{
			{Type: "go-module", Name: "github.com/acme/apis/proto/payments/ledger", Version: "v1.2.0-beta.1", Status: "published"},
		},
	}

	require.NoError(t, WriteReleaseRecord(original, path))

	loaded, err := ReadReleaseRecord(path)
	require.NoError(t, err)

	assert.Equal(t, original.SchemaVersion, loaded.SchemaVersion)
	assert.Equal(t, original.Kind, loaded.Kind)
	assert.Equal(t, original.APIID, loaded.APIID)
	assert.Equal(t, original.Version, loaded.Version)
	assert.Equal(t, original.Tag, loaded.Tag)
	assert.Equal(t, original.GoModule, loaded.GoModule)
	assert.Equal(t, original.CatalogUpdated, loaded.CatalogUpdated)
	assert.Equal(t, original.FinalizedAt, loaded.FinalizedAt)
	require.Len(t, loaded.Artifacts, 1)
	assert.Equal(t, "go-module", loaded.Artifacts[0].Type)
}

func TestFormatRecordReport(t *testing.T) {
	record := &ReleaseRecord{
		APIID:          "proto/payments/ledger/v1",
		Version:        "v1.0.0",
		Tag:            "proto/payments/ledger/v1/v1.0.0",
		Lifecycle:      "stable",
		Format:         "proto",
		SourceRepo:     "github.com/acme/service",
		SourcePath:     "proto/payments/ledger/v1",
		SourceCommit:   "abc123",
		CanonicalRepo:  "github.com/acme/apis",
		CanonicalPath:  "proto/payments/ledger/v1",
		GoModule:       "github.com/acme/apis/proto/payments/ledger",
		GoImport:       "github.com/acme/apis/proto/payments/ledger/v1",
		CatalogUpdated: true,
		FinalizedAt:    "2025-01-01T00:00:00Z",
		Artifacts: []ReleaseArtifact{
			{Type: "go-module", Name: "github.com/acme/apis/proto/payments/ledger", Status: "published"},
		},
	}

	report := FormatRecordReport(record)
	assert.Contains(t, report, "proto/payments/ledger/v1")
	assert.Contains(t, report, "v1.0.0")
	assert.Contains(t, report, "stable")
	assert.Contains(t, report, "go-module")
	assert.Contains(t, report, "true")
}

func TestMarshalReleaseRecord(t *testing.T) {
	record := &ReleaseRecord{
		SchemaVersion: "1",
		Kind:          "release-record",
		APIID:         "proto/payments/ledger/v1",
		Version:       "v1.0.0",
		FinalizedAt:   "2025-01-01T00:00:00Z",
	}

	data, err := MarshalReleaseRecord(record)
	require.NoError(t, err)
	assert.Contains(t, string(data), "release-record")
	assert.Contains(t, string(data), "proto/payments/ledger/v1")
}

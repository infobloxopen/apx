package publisher

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/infobloxopen/apx/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManifest(t *testing.T) {
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

	assert.Equal(t, "1", m.SchemaVersion)
	assert.Equal(t, StateDraft, m.State)
	assert.Equal(t, "proto/payments/ledger/v1", m.APIID)
	assert.Equal(t, "proto", m.Format)
	assert.Equal(t, "payments", m.Domain)
	assert.Equal(t, "ledger", m.Name)
	assert.Equal(t, "v1", m.Line)
	assert.Equal(t, "beta", m.Lifecycle)
	assert.Equal(t, "v1.2.0-beta.1", m.RequestedVersion)
	assert.Equal(t, "github.com/acme/apis/proto/payments/ledger", m.GoModule)
	assert.Equal(t, "github.com/acme/apis/proto/payments/ledger/v1", m.GoImport)
	assert.Equal(t, "proto/payments/ledger/v1/v1.2.0-beta.1", m.Tag)
}

func TestManifest_SetState(t *testing.T) {
	m := &ReleaseManifest{State: StateDraft}

	require.NoError(t, m.SetState(StateValidated))
	assert.Equal(t, StateValidated, m.State)

	require.NoError(t, m.SetState(StatePrepared))
	assert.Equal(t, StatePrepared, m.State)
	assert.NotEmpty(t, m.PreparedAt)

	// Backward should fail
	assert.Error(t, m.SetState(StateDraft))
}

func TestManifest_Fail(t *testing.T) {
	m := &ReleaseManifest{State: StatePrepared}
	m.Fail("VERSION_TAKEN", "version already exists", "submit")

	assert.Equal(t, StateFailed, m.State)
	require.NotNil(t, m.Error)
	assert.Equal(t, "VERSION_TAKEN", m.Error.Code)
	assert.Equal(t, "version already exists", m.Error.Message)
	assert.Equal(t, "submit", m.Error.Phase)
}

func TestWriteReadManifest(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, ".apx-release.yaml")

	original := &ReleaseManifest{
		SchemaVersion:    "1",
		State:            StatePrepared,
		APIID:            "proto/payments/ledger/v1",
		Format:           "proto",
		Domain:           "payments",
		Name:             "ledger",
		Line:             "v1",
		Lifecycle:        "beta",
		SourceRepo:       "github.com/acme/apis",
		SourcePath:       "proto/payments/ledger/v1",
		SourceCommit:     "abc123",
		RequestedVersion: "v1.2.0-beta.1",
		CanonicalRepo:    "github.com/acme/apis",
		CanonicalPath:    "proto/payments/ledger/v1",
		GoModule:         "github.com/acme/apis/proto/payments/ledger",
		GoImport:         "github.com/acme/apis/proto/payments/ledger/v1",
		Tag:              "proto/payments/ledger/v1/v1.2.0-beta.1",
		Validation: &ValidationResults{
			Lint:     ValidationPassed,
			Breaking: ValidationPassed,
			Policy:   ValidationSkipped,
		},
	}

	require.NoError(t, WriteManifest(original, path))

	// File should exist
	_, err := os.Stat(path)
	require.NoError(t, err)

	// Read it back
	loaded, err := ReadManifest(path)
	require.NoError(t, err)

	assert.Equal(t, original.SchemaVersion, loaded.SchemaVersion)
	assert.Equal(t, original.State, loaded.State)
	assert.Equal(t, original.APIID, loaded.APIID)
	assert.Equal(t, original.Format, loaded.Format)
	assert.Equal(t, original.RequestedVersion, loaded.RequestedVersion)
	assert.Equal(t, original.SourceCommit, loaded.SourceCommit)
	assert.Equal(t, original.GoModule, loaded.GoModule)
	assert.Equal(t, original.Tag, loaded.Tag)
	require.NotNil(t, loaded.Validation)
	assert.Equal(t, ValidationPassed, loaded.Validation.Lint)
	assert.Equal(t, ValidationPassed, loaded.Validation.Breaking)
	assert.Equal(t, ValidationSkipped, loaded.Validation.Policy)
}

func TestReadManifest_NotFound(t *testing.T) {
	_, err := ReadManifest("/nonexistent/path/.apx-release.yaml")
	assert.Error(t, err)
}

func TestFormatManifestReport(t *testing.T) {
	m := &ReleaseManifest{
		State:            StatePrepared,
		APIID:            "proto/payments/ledger/v1",
		Format:           "proto",
		Domain:           "payments",
		Name:             "ledger",
		Line:             "v1",
		Lifecycle:        "beta",
		RequestedVersion: "v1.2.0-beta.1",
		Tag:              "proto/payments/ledger/v1/v1.2.0-beta.1",
		SourceRepo:       "github.com/acme/apis",
		SourcePath:       "proto/payments/ledger/v1",
		CanonicalRepo:    "github.com/acme/apis",
		CanonicalPath:    "proto/payments/ledger/v1",
		GoModule:         "github.com/acme/apis/proto/payments/ledger",
		GoImport:         "github.com/acme/apis/proto/payments/ledger/v1",
	}

	report := FormatManifestReport(m)
	assert.Contains(t, report, "prepared")
	assert.Contains(t, report, "proto/payments/ledger/v1")
	assert.Contains(t, report, "v1.2.0-beta.1")
	assert.Contains(t, report, "github.com/acme/apis")
}

func TestWriteReadManifest_PRMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, ".apx-release.yaml")

	original := &ReleaseManifest{
		SchemaVersion:    "1",
		State:            StateCanonicalPROpen,
		APIID:            "proto/payments/ledger/v1",
		Format:           "proto",
		Domain:           "payments",
		Name:             "ledger",
		Line:             "v1",
		SourceRepo:       "github.com/acme/apis",
		SourcePath:       "proto/payments/ledger/v1",
		RequestedVersion: "v1.2.0",
		CanonicalRepo:    "github.com/acme/apis",
		CanonicalPath:    "proto/payments/ledger/v1",
		Tag:              "proto/payments/ledger/v1/v1.2.0",
		PRNumber:         42,
		PRURL:            "https://github.com/acme/apis/pull/42",
		PRBranch:         "apx/release/proto-payments-ledger-v1/v1.2.0",
	}

	require.NoError(t, WriteManifest(original, path))

	loaded, err := ReadManifest(path)
	require.NoError(t, err)

	assert.Equal(t, 42, loaded.PRNumber)
	assert.Equal(t, "https://github.com/acme/apis/pull/42", loaded.PRURL)
	assert.Equal(t, "apx/release/proto-payments-ledger-v1/v1.2.0", loaded.PRBranch)
	assert.Equal(t, StateCanonicalPROpen, loaded.State)
}

func TestFormatManifestReport_WithPRMetadata(t *testing.T) {
	m := &ReleaseManifest{
		State:            StateCanonicalPROpen,
		APIID:            "proto/payments/ledger/v1",
		Format:           "proto",
		Domain:           "payments",
		Name:             "ledger",
		Line:             "v1",
		RequestedVersion: "v1.2.0",
		Tag:              "proto/payments/ledger/v1/v1.2.0",
		SourceRepo:       "github.com/acme/apis",
		SourcePath:       "proto/payments/ledger/v1",
		CanonicalRepo:    "github.com/acme/apis",
		CanonicalPath:    "proto/payments/ledger/v1",
		PRNumber:         42,
		PRURL:            "https://github.com/acme/apis/pull/42",
		PRBranch:         "apx/release/proto-payments-ledger-v1/v1.2.0",
	}

	report := FormatManifestReport(m)
	assert.Contains(t, report, "PR URL:      https://github.com/acme/apis/pull/42")
	assert.Contains(t, report, "PR number:   42")
	assert.Contains(t, report, "PR branch:   apx/release/proto-payments-ledger-v1/v1.2.0")
}

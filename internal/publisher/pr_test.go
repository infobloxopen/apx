package publisher

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// ParseCanonicalNWO
// ---------------------------------------------------------------------------

func TestParseCanonicalNWO(t *testing.T) {
	tests := []struct {
		input   string
		want    string
		wantErr bool
	}{
		{"github.com/acme/apis", "acme/apis", false},
		{"https://github.com/acme/apis", "acme/apis", false},
		{"https://github.com/acme/apis.git", "acme/apis", false},
		{"http://github.com/acme/apis/", "acme/apis", false},
		{"acme/apis", "acme/apis", false},
		{"", "", true},
		{"github.com/", "", true},
		{"github.com/acme", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseCanonicalNWO(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// CreatePR (stubbed gh CLI)
// ---------------------------------------------------------------------------

func TestCreatePR_Success(t *testing.T) {
	origGH := GHRun
	defer func() { GHRun = origGH }()

	GHRun = func(args ...string) (string, error) {
		// Verify the arguments look right
		assert.Contains(t, args, "pr")
		assert.Contains(t, args, "create")
		assert.Contains(t, args, "--repo")
		assert.Contains(t, args, "acme/apis")
		return "https://github.com/acme/apis/pull/42", nil
	}

	resp, err := CreatePR("acme/apis", "apx/publish/test", "main", "title", "body")
	require.NoError(t, err)
	assert.Equal(t, "https://github.com/acme/apis/pull/42", resp.HTMLURL)
}

func TestCreatePR_JSONResponse(t *testing.T) {
	origGH := GHRun
	defer func() { GHRun = origGH }()

	GHRun = func(args ...string) (string, error) {
		return `{"number":42,"url":"https://github.com/acme/apis/pull/42","state":"open"}`, nil
	}

	resp, err := CreatePR("acme/apis", "branch", "main", "title", "body")
	require.NoError(t, err)
	assert.Equal(t, 42, resp.Number)
	assert.Equal(t, "https://github.com/acme/apis/pull/42", resp.HTMLURL)
	assert.Equal(t, "open", resp.State)
}

func TestCreatePR_Failure(t *testing.T) {
	origGH := GHRun
	defer func() { GHRun = origGH }()

	GHRun = func(args ...string) (string, error) {
		return "permission denied", assert.AnError
	}

	_, err := CreatePR("acme/apis", "branch", "main", "title", "body")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "gh pr create failed")
}

// ---------------------------------------------------------------------------
// copyDir
// ---------------------------------------------------------------------------

func TestCopyDir(t *testing.T) {
	src := t.TempDir()
	dst := filepath.Join(t.TempDir(), "dest")

	// Create nested source structure
	require.NoError(t, os.MkdirAll(filepath.Join(src, "v1"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(src, "v1", "ledger.proto"), []byte("syntax = \"proto3\";"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(src, "README.md"), []byte("# API"), 0o644))

	// Copy
	require.NoError(t, copyDir(src, dst))

	// Verify
	data, err := os.ReadFile(filepath.Join(dst, "v1", "ledger.proto"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "proto3")

	data, err = os.ReadFile(filepath.Join(dst, "README.md"))
	require.NoError(t, err)
	assert.Equal(t, "# API", string(data))
}

// ---------------------------------------------------------------------------
// CheckGHCLI
// ---------------------------------------------------------------------------

func TestCheckGHCLI_AuthFails(t *testing.T) {
	origGH := GHRun
	defer func() { GHRun = origGH }()

	GHRun = func(args ...string) (string, error) {
		return "not logged in", assert.AnError
	}

	err := CheckGHCLI()
	// If gh is not in PATH this will fail at LookPath; if it is, it will
	// hit our stubbed auth failure.  Either way we expect an error.
	assert.Error(t, err)
}

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

// ---------------------------------------------------------------------------
// FindExistingPR
// ---------------------------------------------------------------------------

func TestFindExistingPR_Found(t *testing.T) {
	origGH := GHRun
	defer func() { GHRun = origGH }()

	GHRun = func(args ...string) (string, error) {
		assert.Contains(t, args, "pr")
		assert.Contains(t, args, "list")
		assert.Contains(t, args, "--head")
		assert.Contains(t, args, "apx/release/proto-payments-ledger-v1/v1.2.0")
		return `[{"number":42,"url":"https://github.com/acme/apis/pull/42","state":"open"}]`, nil
	}

	resp, err := FindExistingPR("acme/apis", "apx/release/proto-payments-ledger-v1/v1.2.0")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 42, resp.Number)
	assert.Equal(t, "https://github.com/acme/apis/pull/42", resp.HTMLURL)
	assert.Equal(t, "open", resp.State)
}

func TestFindExistingPR_NotFound(t *testing.T) {
	origGH := GHRun
	defer func() { GHRun = origGH }()

	GHRun = func(args ...string) (string, error) {
		return "[]", nil
	}

	resp, err := FindExistingPR("acme/apis", "apx/release/test/v1.0.0")
	require.NoError(t, err)
	assert.Nil(t, resp)
}

func TestFindExistingPR_EmptyResponse(t *testing.T) {
	origGH := GHRun
	defer func() { GHRun = origGH }()

	GHRun = func(args ...string) (string, error) {
		return "", nil
	}

	resp, err := FindExistingPR("acme/apis", "apx/release/test/v1.0.0")
	require.NoError(t, err)
	assert.Nil(t, resp)
}

func TestFindExistingPR_GHError(t *testing.T) {
	origGH := GHRun
	defer func() { GHRun = origGH }()

	GHRun = func(args ...string) (string, error) {
		return "network error", assert.AnError
	}

	_, err := FindExistingPR("acme/apis", "apx/release/test/v1.0.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "gh pr list failed")
}

// ---------------------------------------------------------------------------
// ComputeReleaseBranchName
// ---------------------------------------------------------------------------

func TestComputeReleaseBranchName(t *testing.T) {
	tests := []struct {
		apiID   string
		version string
		want    string
	}{
		{"proto/payments/ledger/v1", "v1.2.0", "apx/release/proto-payments-ledger-v1/v1.2.0"},
		{"openapi/users/v2", "v2.0.0-beta.1", "apx/release/openapi-users-v2/v2.0.0-beta.1"},
		{"simple", "v1.0.0", "apx/release/simple/v1.0.0"},
	}
	for _, tt := range tests {
		t.Run(tt.apiID+"/"+tt.version, func(t *testing.T) {
			got := ComputeReleaseBranchName(tt.apiID, tt.version)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// SubmitReleaseWithPR (unit tests with stubbed commands)
// ---------------------------------------------------------------------------

// stubGit replaces runGit and runGitIn for tests, restoring them on cleanup.
func stubGit(t *testing.T, gitFn func(args ...string) (string, error)) {
	t.Helper()
	origGit := runGitFn
	origGitIn := runGitInFn
	t.Cleanup(func() {
		runGitFn = origGit
		runGitInFn = origGitIn
	})
	runGitFn = gitFn
	runGitInFn = func(_ string, args ...string) (string, error) {
		return gitFn(args...)
	}
}

func TestSubmitReleaseWithPR_BranchAndCommit(t *testing.T) {
	origGH := GHRun
	defer func() { GHRun = origGH }()

	// Track git commands to verify branch naming and commit message
	var gitArgs [][]string
	stubGit(t, func(args ...string) (string, error) {
		gitArgs = append(gitArgs, args)
		return "", nil
	})

	// Stub gh CLI: FindExistingPR returns nothing, CreatePR returns URL
	ghCalls := 0
	GHRun = func(args ...string) (string, error) {
		ghCalls++
		if args[0] == "auth" {
			return "Logged in", nil
		}
		if args[0] == "pr" && args[1] == "list" {
			return "[]", nil
		}
		if args[0] == "pr" && args[1] == "create" {
			// Verify title contains release prefix
			for i, a := range args {
				if a == "--title" && i+1 < len(args) {
					assert.Contains(t, args[i+1], "release:")
				}
			}
			return `{"number":99,"url":"https://github.com/acme/apis/pull/99","state":"open"}`, nil
		}
		return "", nil
	}

	// Create a temp snapshot directory
	snapshotDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(snapshotDir, "ledger.proto"), []byte("syntax = \"proto3\";"), 0o644))

	manifest := &ReleaseManifest{
		SchemaVersion:    "1",
		State:            StatePrepared,
		APIID:            "proto/payments/ledger/v1",
		Format:           "proto",
		Domain:           "payments",
		Name:             "ledger",
		Line:             "v1",
		RequestedVersion: "v1.2.0",
		SourceRepo:       "github.com/acme/app",
		SourcePath:       "proto/payments/ledger/v1",
		CanonicalRepo:    "github.com/acme/apis",
		CanonicalPath:    "proto/payments/ledger/v1",
		Tag:              "proto/payments/ledger/v1/v1.2.0",
	}

	resp, err := SubmitReleaseWithPR(manifest, snapshotDir, "")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 99, resp.Number)
	assert.Equal(t, "https://github.com/acme/apis/pull/99", resp.HTMLURL)

	// Verify git checkout used the correct branch name
	found := false
	for _, args := range gitArgs {
		if len(args) >= 3 && args[0] == "checkout" && args[1] == "-b" {
			assert.Equal(t, "apx/release/proto-payments-ledger-v1/v1.2.0", args[2])
			found = true
		}
	}
	assert.True(t, found, "expected checkout -b with release branch name")

	// Verify force push was used
	pushFound := false
	for _, args := range gitArgs {
		if len(args) >= 2 && args[0] == "push" && args[1] == "--force" {
			pushFound = true
		}
	}
	assert.True(t, pushFound, "expected push --force for retry safety")
}

func TestSubmitReleaseWithPR_ExistingPR(t *testing.T) {
	origGH := GHRun
	defer func() { GHRun = origGH }()

	stubGit(t, func(args ...string) (string, error) {
		return "", nil
	})

	GHRun = func(args ...string) (string, error) {
		if args[0] == "auth" {
			return "Logged in", nil
		}
		if args[0] == "pr" && args[1] == "list" {
			// Return an existing PR
			return `[{"number":55,"url":"https://github.com/acme/apis/pull/55","state":"open"}]`, nil
		}
		t.Fatal("should not call pr create when existing PR found")
		return "", nil
	}

	snapshotDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(snapshotDir, "test.proto"), []byte("proto"), 0o644))

	manifest := &ReleaseManifest{
		APIID:            "proto/test/v1",
		RequestedVersion: "v1.0.0",
		CanonicalRepo:    "github.com/acme/apis",
		CanonicalPath:    "proto/test/v1",
		SourceRepo:       "github.com/acme/app",
		SourcePath:       "proto/test/v1",
		Tag:              "proto/test/v1/v1.0.0",
	}

	resp, err := SubmitReleaseWithPR(manifest, snapshotDir, "")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 55, resp.Number)
	assert.Equal(t, "https://github.com/acme/apis/pull/55", resp.HTMLURL)
}

func TestSubmitReleaseWithPR_WithCIProvenance(t *testing.T) {
	origGH := GHRun
	defer func() { GHRun = origGH }()

	stubGit(t, func(args ...string) (string, error) {
		return "", nil
	})

	var capturedBody string
	GHRun = func(args ...string) (string, error) {
		if args[0] == "auth" {
			return "Logged in", nil
		}
		if args[0] == "pr" && args[1] == "list" {
			return "[]", nil
		}
		if args[0] == "pr" && args[1] == "create" {
			for i, a := range args {
				if a == "--body" && i+1 < len(args) {
					capturedBody = args[i+1]
				}
			}
			return `{"number":77,"url":"https://github.com/acme/apis/pull/77","state":"open"}`, nil
		}
		return "", nil
	}

	snapshotDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(snapshotDir, "test.proto"), []byte("proto"), 0o644))

	manifest := &ReleaseManifest{
		APIID:            "proto/test/v1",
		RequestedVersion: "v1.0.0",
		CanonicalRepo:    "github.com/acme/apis",
		CanonicalPath:    "proto/test/v1",
		SourceRepo:       "github.com/acme/app",
		SourcePath:       "proto/test/v1",
		Tag:              "proto/test/v1/v1.0.0",
	}

	ciExtra := "**CI**: github-actions\n**Run**: https://github.com/acme/app/actions/runs/12345"
	resp, err := SubmitReleaseWithPR(manifest, snapshotDir, ciExtra)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Contains(t, capturedBody, "CI")
	assert.Contains(t, capturedBody, "github-actions")
}

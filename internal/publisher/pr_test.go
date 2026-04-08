package publisher

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/infobloxopen/apx/pkg/githubauth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestClient creates an httptest server and a githubauth.Client pointed at it.
// The handler receives all API requests for assertions.
func setupTestClient(t *testing.T, handler http.HandlerFunc) (*githubauth.Client, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	origBase := githubauth.APIBaseURL
	t.Cleanup(func() { githubauth.APIBaseURL = origBase })
	githubauth.APIBaseURL = server.URL

	return githubauth.NewClient("test-token"), server
}

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
// CreatePR (via httptest)
// ---------------------------------------------------------------------------

func TestCreatePR_Success(t *testing.T) {
	client, _ := setupTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/repos/acme/apis/pulls", r.URL.Path)

		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "title", body["title"])
		assert.Equal(t, "apx/release/test", body["head"])
		assert.Equal(t, "main", body["base"])

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"number":42,"html_url":"https://github.com/acme/apis/pull/42","state":"open"}`)
	})

	resp, err := CreatePR(client, "acme/apis", "apx/release/test", "main", "title", "body")
	require.NoError(t, err)
	assert.Equal(t, 42, resp.Number)
	assert.Equal(t, "https://github.com/acme/apis/pull/42", resp.HTMLURL)
	assert.Equal(t, "open", resp.State)
}

func TestCreatePR_Failure(t *testing.T) {
	client, _ := setupTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"message":"permission denied"}`)
	})

	_, err := CreatePR(client, "acme/apis", "branch", "main", "title", "body")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "PR create failed")
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
// FindExistingPR
// ---------------------------------------------------------------------------

func TestFindExistingPR_Found(t *testing.T) {
	client, _ := setupTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/repos/acme/apis/pulls")
		assert.Contains(t, r.URL.RawQuery, "state=open")
		// head must be qualified as owner:branch for GitHub API
		assert.Contains(t, r.URL.RawQuery, "head=acme:")

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"number":42,"html_url":"https://github.com/acme/apis/pull/42","state":"open"}]`)
	})

	resp, err := FindExistingPR(client, "acme/apis", "apx/release/proto-payments-ledger-v1/v1.2.0")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 42, resp.Number)
	assert.Equal(t, "https://github.com/acme/apis/pull/42", resp.HTMLURL)
	assert.Equal(t, "open", resp.State)
}

func TestFindExistingPR_NotFound(t *testing.T) {
	client, _ := setupTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	})

	resp, err := FindExistingPR(client, "acme/apis", "apx/release/test/v1.0.0")
	require.NoError(t, err)
	assert.Nil(t, resp)
}

func TestFindExistingPR_APIError(t *testing.T) {
	client, _ := setupTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"message":"internal error"}`)
	})

	_, err := FindExistingPR(client, "acme/apis", "apx/release/test/v1.0.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "PR list failed")
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
	// Track git commands to verify branch naming and commit message
	var gitArgs [][]string
	stubGit(t, func(args ...string) (string, error) {
		gitArgs = append(gitArgs, args)
		return "", nil
	})

	// Mock GitHub API: FindExistingPR returns empty, CreatePR returns PR
	reqNum := 0
	client, _ := setupTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		reqNum++
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == "GET" && r.URL.Path == "/repos/acme/apis/pulls":
			fmt.Fprint(w, `[]`) // no existing PR
		case r.Method == "POST" && r.URL.Path == "/repos/acme/apis/pulls":
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			assert.Contains(t, body["title"], "release:")
			fmt.Fprint(w, `{"number":99,"html_url":"https://github.com/acme/apis/pull/99","state":"open"}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

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

	resp, err := SubmitReleaseWithPR(client, manifest, snapshotDir, "")
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
	stubGit(t, func(args ...string) (string, error) {
		return "", nil
	})

	client, _ := setupTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == "GET" && r.URL.Path == "/repos/acme/apis/pulls":
			fmt.Fprint(w, `[{"number":55,"html_url":"https://github.com/acme/apis/pull/55","state":"open"}]`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})

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

	resp, err := SubmitReleaseWithPR(client, manifest, snapshotDir, "")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 55, resp.Number)
	assert.Equal(t, "https://github.com/acme/apis/pull/55", resp.HTMLURL)
}

func TestSubmitReleaseWithPR_WithCIProvenance(t *testing.T) {
	stubGit(t, func(args ...string) (string, error) {
		return "", nil
	})

	var capturedBody map[string]string
	client, _ := setupTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == "GET" && r.URL.Path == "/repos/acme/apis/pulls":
			fmt.Fprint(w, `[]`)
		case r.Method == "POST" && r.URL.Path == "/repos/acme/apis/pulls":
			_ = json.NewDecoder(r.Body).Decode(&capturedBody)
			fmt.Fprint(w, `{"number":77,"html_url":"https://github.com/acme/apis/pull/77","state":"open"}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

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
	resp, err := SubmitReleaseWithPR(client, manifest, snapshotDir, ciExtra)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Contains(t, capturedBody["body"], "CI")
	assert.Contains(t, capturedBody["body"], "github-actions")
}

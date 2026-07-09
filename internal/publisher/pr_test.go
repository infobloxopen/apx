package publisher

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

// isDiffQuiet reports whether the git args are the "diff --cached --quiet"
// no-diff probe. Real git exits non-zero from this command when changes are
// staged, so a stub that simulates "there are changes" must return an error
// for it — otherwise SubmitReleaseWithPR treats the release as empty.
func isDiffQuiet(args []string) bool {
	return len(args) == 3 && args[0] == "diff" && args[1] == "--cached" && args[2] == "--quiet"
}

func TestSubmitReleaseWithPR_BranchAndCommit(t *testing.T) {
	// Track git commands to verify branch naming and commit message
	var gitArgs [][]string
	stubGit(t, func(args ...string) (string, error) {
		gitArgs = append(gitArgs, args)
		if isDiffQuiet(args) {
			return "", fmt.Errorf("exit status 1") // staged changes present
		}
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

	resp, err := SubmitReleaseWithPR(client, manifest, snapshotDir, "", "main")
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
		if isDiffQuiet(args) {
			return "", fmt.Errorf("exit status 1") // staged changes present
		}
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

	resp, err := SubmitReleaseWithPR(client, manifest, snapshotDir, "", "main")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 55, resp.Number)
	assert.Equal(t, "https://github.com/acme/apis/pull/55", resp.HTMLURL)
}

func TestSubmitReleaseWithPR_WithCIProvenance(t *testing.T) {
	stubGit(t, func(args ...string) (string, error) {
		if isDiffQuiet(args) {
			return "", fmt.Errorf("exit status 1") // staged changes present
		}
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
	resp, err := SubmitReleaseWithPR(client, manifest, snapshotDir, ciExtra, "main")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Contains(t, capturedBody["body"], "CI")
	assert.Contains(t, capturedBody["body"], "github-actions")
}

// TestSubmitReleaseWithPR_BaseBranch verifies the resolved base branch (ARCH-271)
// flows through to the created PR — a develop publish must target apis "develop",
// not the hardcoded "main".
func TestSubmitReleaseWithPR_BaseBranch(t *testing.T) {
	stubGit(t, func(args ...string) (string, error) {
		if isDiffQuiet(args) {
			return "", fmt.Errorf("exit status 1") // staged changes present
		}
		return "", nil
	})

	var capturedBase string
	client, _ := setupTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == "GET" && r.URL.Path == "/repos/acme/apis/pulls":
			fmt.Fprint(w, `[]`)
		case r.Method == "POST" && r.URL.Path == "/repos/acme/apis/pulls":
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			capturedBase = body["base"]
			fmt.Fprint(w, `{"number":123,"html_url":"https://github.com/acme/apis/pull/123","state":"open"}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	snapshotDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(snapshotDir, "u.yaml"), []byte("openapi: 3.0.0"), 0o644))

	manifest := &ReleaseManifest{
		APIID:            "openapi/users/v1",
		RequestedVersion: "v1.2.0-beta.1.g1a2b3c4d5e6f",
		CanonicalRepo:    "github.com/acme/apis",
		CanonicalPath:    "openapi/users/v1",
		SourceRepo:       "github.com/acme/app",
		SourcePath:       "openapi/users/v1",
		Tag:              "openapi/users/v1/v1.2.0-beta.1.g1a2b3c4d5e6f",
		BaseBranch:       "develop",
	}

	resp, err := SubmitReleaseWithPR(client, manifest, snapshotDir, "", manifest.BaseBranch)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "develop", capturedBase, "release PR must target the resolved base branch")
}

// TestSubmitReleaseWithPR_NoDiff verifies that when the prepared snapshot is
// identical to canonical (nothing staged), submit returns ErrNoReleaseDiff and
// never reaches PR creation — instead of pushing an empty branch and getting an
// opaque HTTP 422.
func TestSubmitReleaseWithPR_NoDiff(t *testing.T) {
	var pushed, prCreated bool
	stubGit(t, func(args ...string) (string, error) {
		if len(args) >= 1 && args[0] == "push" {
			pushed = true
		}
		// isDiffQuiet returns nil error here → no staged changes.
		return "", nil
	})

	client, _ := setupTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/repos/acme/apis/pulls" {
			prCreated = true
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
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

	resp, err := SubmitReleaseWithPR(client, manifest, snapshotDir, "", "main")
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrNoReleaseDiff)
	assert.False(t, pushed, "must not push an empty branch")
	assert.False(t, prCreated, "must not create a PR from an empty diff")
}

// ---------------------------------------------------------------------------
// SubmitReleaseWithPR — apx#34 regression (real git against a diverged repo)
// ---------------------------------------------------------------------------

// mustGit runs a real git command and fails the test on error.
func mustGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := runGitInReal(dir, args...)
	require.NoError(t, err, "git %v: %s", args, out)
	return out
}

// writeTestFile writes content to path, creating parent directories.
func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

// gitSetIdentity sets a local commit identity so commits work in CI where no
// global git identity is configured.
func gitSetIdentity(t *testing.T, dir string) {
	t.Helper()
	mustGit(t, dir, "config", "user.email", "test@example.com")
	mustGit(t, dir, "config", "user.name", "apx test")
}

// TestSubmitReleaseWithPR_DivergedDevelopBaseBranch is the apx#34 regression:
// a develop publish must cut the release branch from — and diff the snapshot
// against — the resolved base branch (develop), not the repo's default branch
// (main). It drives real git against a local bare repo whose main and develop
// have diverged.
//
// Scenario:
//   - main:    users.yaml == snapshotContent (byte-identical to the snapshot)
//   - develop: users.yaml == developContent + a develop-only marker file
//     (content absent from main)
//   - snapshot: users.yaml == snapshotContent
//
// On the pre-fix code the clone fetches main (the default branch), so staging
// the snapshot over main's tree produces no diff → a false ErrNoReleaseDiff
// ("Nothing to release"), and any branch would be cut from main. After the fix
// the clone fetches develop, so the snapshot differs from develop → a real
// diff; the release branch carries develop's develop-only file (proving it was
// cut from develop) and the PR is opened with base=develop.
func TestSubmitReleaseWithPR_DivergedDevelopBaseBranch(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	const (
		snapshotContent = "openapi: 3.0.0\ninfo:\n  title: Users\n  version: 1.1.1\n"
		developContent  = "openapi: 3.0.0\ninfo:\n  title: Users\n  version: 1.1.2-beta\n"
		developOnlyFile = "DEVELOP_ONLY.md"
		canonicalPath   = "openapi/users/v1"
	)

	// ── Build a local bare "canonical" repo with diverged main + develop ──
	root := t.TempDir()
	origin := filepath.Join(root, "origin.git")
	work := filepath.Join(root, "work")

	mustGit(t, "", "init", "--bare", "-b", "main", origin)

	// Seed main with the snapshot content.
	mustGit(t, "", "clone", origin, work)
	gitSetIdentity(t, work)
	writeTestFile(t, filepath.Join(work, canonicalPath, "users.yaml"), snapshotContent)
	mustGit(t, work, "add", "--all")
	mustGit(t, work, "commit", "-m", "main: users v1.1.1")
	mustGit(t, work, "push", "origin", "main")

	// Diverge develop: change users.yaml and add a develop-only file.
	mustGit(t, work, "checkout", "-b", "develop")
	writeTestFile(t, filepath.Join(work, canonicalPath, "users.yaml"), developContent)
	writeTestFile(t, filepath.Join(work, canonicalPath, developOnlyFile), "develop-only content\n")
	mustGit(t, work, "add", "--all")
	mustGit(t, work, "commit", "-m", "develop: diverge from main")
	mustGit(t, work, "push", "origin", "develop")

	// Origin's default branch is main, so a plain (pre-fix) clone checks out main.
	mustGit(t, "", "--git-dir", origin, "symbolic-ref", "HEAD", "refs/heads/main")

	// ── Redirect the production clone at the local bare repo and run real git
	// for every other step. runGit is used only for the clone; runGitIn for
	// checkout/add/commit/diff/push. ──
	origGit := runGitFn
	origGitIn := runGitInFn
	t.Cleanup(func() { runGitFn = origGit; runGitInFn = origGitIn })
	runGitFn = func(args ...string) (string, error) {
		rewritten := make([]string, len(args))
		for i, a := range args {
			if strings.HasPrefix(a, "https://github.com/") {
				rewritten[i] = origin
			} else {
				rewritten[i] = a
			}
		}
		out, err := runGitInReal("", rewritten...)
		if err == nil {
			// The clone's push identity must be set so the commit succeeds.
			gitSetIdentity(t, filepath.Join(rewritten[len(rewritten)-1]))
		}
		return out, err
	}
	runGitInFn = runGitInReal

	// ── Snapshot is byte-identical to main's content. ──
	snapshotDir := t.TempDir()
	writeTestFile(t, filepath.Join(snapshotDir, "users.yaml"), snapshotContent)

	// ── GitHub API mock: capture the PR base. ──
	var capturedBase string
	client, _ := setupTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == "GET" && r.URL.Path == "/repos/acme/apis/pulls":
			fmt.Fprint(w, `[]`)
		case r.Method == "POST" && r.URL.Path == "/repos/acme/apis/pulls":
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			capturedBase = body["base"]
			fmt.Fprint(w, `{"number":34,"html_url":"https://github.com/acme/apis/pull/34","state":"open"}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	manifest := &ReleaseManifest{
		APIID:            "openapi/users/v1",
		RequestedVersion: "v1.1.2-beta.1.gabcdef123456",
		CanonicalRepo:    "github.com/acme/apis",
		CanonicalPath:    canonicalPath,
		SourceRepo:       "github.com/acme/app",
		SourcePath:       canonicalPath,
		Tag:              "openapi/users/v1/v1.1.2-beta.1.gabcdef123456",
		BaseBranch:       "develop",
	}

	resp, err := SubmitReleaseWithPR(client, manifest, snapshotDir, "", "develop")

	// (b) No false no-op: the snapshot differs from develop, so there is a diff.
	require.NoError(t, err, "must not false-report ErrNoReleaseDiff — snapshot differs from develop")
	require.NotNil(t, resp)

	// (c) PR opened against develop.
	assert.Equal(t, "develop", capturedBase, "PR must target the resolved base branch develop")

	// (a) Release branch cut from develop: it carries develop's develop-only
	// file (absent on main), proving the branch was based on develop's tree.
	branch := ComputeReleaseBranchName(manifest.APIID, manifest.RequestedVersion)
	tree, treeErr := runGitInReal("", "--git-dir", origin, "ls-tree", "-r", "--name-only", branch)
	require.NoError(t, treeErr, "release branch must have been pushed to origin: %s", tree)
	assert.Contains(t, tree, canonicalPath+"/"+developOnlyFile,
		"release branch must be cut from develop (carrying develop's develop-only file), not main")
}

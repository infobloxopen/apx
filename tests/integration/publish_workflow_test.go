package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/infobloxopen/apx/internal/publisher"
)

func TestPublishWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if we're in a CI environment or have Gitea available
	if os.Getenv("CI") == "" && os.Getenv("GITEA_URL") == "" {
		t.Skip("Skipping publish workflow test (requires Gitea or CI environment)")
	}

	// Create temporary directories for test repos
	tmpDir := t.TempDir()
	appRepoDir := filepath.Join(tmpDir, "app-repo")
	canonicalRepoDir := filepath.Join(tmpDir, "canonical-repo")

	// Setup canonical repository
	if err := os.MkdirAll(canonicalRepoDir, 0755); err != nil {
		t.Fatalf("Failed to create canonical repo dir: %v", err)
	}

	// Initialize canonical repo as bare git repo
	cmd := exec.Command("git", "init", "--bare", canonicalRepoDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to init canonical repo: %v\nOutput: %s", err, string(output))
	}

	// Setup app repository
	if err := os.MkdirAll(appRepoDir, 0755); err != nil {
		t.Fatalf("Failed to create app repo dir: %v", err)
	}

	cmd = exec.Command("git", "init")
	cmd.Dir = appRepoDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to init app repo: %v\nOutput: %s", err, string(output))
	}

	// Configure git user
	configCmds := [][]string{
		{"git", "config", "user.name", "Test User"},
		{"git", "config", "user.email", "test@example.com"},
	}
	for _, cmdArgs := range configCmds {
		cmd = exec.Command(cmdArgs[0], cmdArgs[1:]...)
		cmd.Dir = appRepoDir
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to configure git: %v\nOutput: %s", err, string(output))
		}
	}

	// Create module directory structure
	moduleDir := filepath.Join(appRepoDir, "internal/apis/proto/payments/ledger/v1")
	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		t.Fatalf("Failed to create module dir: %v", err)
	}

	// Create a simple proto file
	protoContent := `syntax = "proto3";

package payments.ledger.v1;

message Transaction {
  string id = 1;
  int64 amount = 2;
  string currency = 3;
}

service LedgerService {
  rpc CreateTransaction(Transaction) returns (Transaction);
}
`
	protoFile := filepath.Join(moduleDir, "ledger.proto")
	if err := os.WriteFile(protoFile, []byte(protoContent), 0644); err != nil {
		t.Fatalf("Failed to write proto file: %v", err)
	}

	// Create apx.yaml
	apxYaml := `kind: proto
module: payments.ledger.v1
org: testorg
version: 1
`
	apxFile := filepath.Join(moduleDir, "apx.yaml")
	if err := os.WriteFile(apxFile, []byte(apxYaml), 0644); err != nil {
		t.Fatalf("Failed to write apx.yaml: %v", err)
	}

	// Create some other files in the repo (to verify subtree isolation)
	otherFile := filepath.Join(appRepoDir, "README.md")
	if err := os.WriteFile(otherFile, []byte("# App Repo\n"), 0644); err != nil {
		t.Fatalf("Failed to write README: %v", err)
	}

	// Commit the module
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = appRepoDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to git add: %v\nOutput: %s", err, string(output))
	}

	cmd = exec.Command("git", "commit", "-m", "Add ledger module v1.0.0")
	cmd.Dir = appRepoDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to git commit: %v\nOutput: %s", err, string(output))
	}

	// Test subtree split
	t.Run("SubtreeSplit", func(t *testing.T) {
		sp := publisher.NewSubtreePublisher(appRepoDir)

		branchName := "test-publish-ledger"
		commitHash, err := sp.Split("internal/apis/proto/payments/ledger/v1", branchName)
		if err != nil {
			t.Fatalf("Subtree split failed: %v", err)
		}

		if commitHash == "" {
			t.Fatal("Expected non-empty commit hash from split")
		}

		// Verify the branch was created
		cmd := exec.Command("git", "rev-parse", branchName)
		cmd.Dir = appRepoDir
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Split branch not found: %v\nOutput: %s", err, string(output))
		}

		t.Logf("Subtree split successful, commit: %s", commitHash)
	})

	// Test publish to local bare repo
	t.Run("PublishToLocal", func(t *testing.T) {
		sp := publisher.NewSubtreePublisher(appRepoDir)

		// Use file:// URL for local bare repo
		remoteURL := "file://" + canonicalRepoDir

		commitHash, err := sp.PublishModule("internal/apis/proto/payments/ledger/v1", remoteURL, "v1.0.0")
		if err != nil {
			t.Fatalf("Publish failed: %v", err)
		}

		if commitHash == "" {
			t.Fatal("Expected non-empty commit hash from publish")
		}

		// Verify the canonical repo received the push
		cmd := exec.Command("git", "log", "--oneline")
		cmd.Dir = canonicalRepoDir
		output, err := cmd.CombinedOutput()
		if err == nil && len(output) > 0 {
			t.Logf("Canonical repo log:\n%s", string(output))
		}

		t.Logf("Publish successful, commit: %s", commitHash)
	})

	// Test tag creation
	t.Run("TagCreation", func(t *testing.T) {
		tm := publisher.NewTagManager(appRepoDir, "proto/payments/ledger/{version}")

		// Get the latest commit
		cmd := exec.Command("git", "rev-parse", "HEAD")
		cmd.Dir = appRepoDir
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to get HEAD: %v", err)
		}
		commitHash := string(output[:len(output)-1]) // trim newline

		tag, err := tm.CreateAndPushTag("v1", "v1.0.0", commitHash)
		// Push will fail since we don't have a real remote configured
		// But we should still get the tag name back
		if tag == "" {
			t.Fatalf("Tag name not returned: %v", err)
		}

		if err != nil {
			t.Logf("Tag created (push failed as expected): %s, error: %v", tag, err)
		} else {
			t.Logf("Tag created and pushed: %s", tag)
		}

		// Verify tag exists locally
		cmd = exec.Command("git", "tag", "-l", tag)
		cmd.Dir = appRepoDir
		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to list tags: %v", err)
		}

		if len(output) == 0 {
			t.Fatalf("Tag not found: %s", tag)
		}

		t.Logf("Tag verified: %s", tag)
	})

	// Test version validation
	t.Run("VersionValidation", func(t *testing.T) {
		tm := publisher.NewTagManager(appRepoDir, "{subdir}/v{version}")

		validVersions := []string{"v1.0.0", "v2.3.4", "v0.1.0-alpha", "v1.2.3+build123"}
		for _, version := range validVersions {
			if err := tm.ValidateVersion(version); err != nil {
				t.Errorf("Expected version %s to be valid, got error: %v", version, err)
			}
		}

		invalidVersions := []string{"1.0.0", "v1", "v1.0", "invalid", ""}
		for _, version := range invalidVersions {
			if err := tm.ValidateVersion(version); err == nil {
				t.Errorf("Expected version %s to be invalid, but it passed validation", version)
			}
		}
	})
}

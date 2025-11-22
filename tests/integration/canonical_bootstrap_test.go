package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCanonicalBootstrap(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create temporary directory for test
	tmpDir := t.TempDir()

	// Build apx binary
	apxBin := filepath.Join(tmpDir, "apx")
	buildCmd := exec.Command("go", "build", "-o", apxBin, "../../cmd/apx")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build apx: %v\n%s", err, output)
	}

	// Create test workspace
	workspaceDir := filepath.Join(tmpDir, "canonical-repo")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	// Run apx init canonical
	initCmd := exec.Command(apxBin, "init", "canonical",
		"--org=infoblox",
		"--repo=api-schemas",
		"--skip-git",
		"--non-interactive")
	initCmd.Dir = workspaceDir
	initCmd.Env = append(os.Environ(), "NO_COLOR=1", "CI=1")

	output, err := initCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("apx init canonical failed: %v\n%s", err, output)
	}

	outputStr := string(output)

	// Verify output messages
	expectedMessages := []string{
		"Initializing canonical API repository",
		"Organization: infoblox",
		"Repository: api-schemas",
		"✓ Created directory structure",
		"✓ Generated buf.yaml",
		"✓ Generated CODEOWNERS",
		"✓ Generated catalog.yaml",
		"✓ Canonical API repository initialized successfully",
	}

	for _, msg := range expectedMessages {
		if !strings.Contains(outputStr, msg) {
			t.Errorf("Expected output to contain %q, got:\n%s", msg, outputStr)
		}
	}

	// Verify directory structure
	expectedDirs := []string{
		"proto",
		"openapi",
		"avro",
		"jsonschema",
		"parquet",
	}

	for _, dir := range expectedDirs {
		dirPath := filepath.Join(workspaceDir, dir)
		if stat, err := os.Stat(dirPath); os.IsNotExist(err) {
			t.Errorf("Expected directory %s to exist", dir)
		} else if !stat.IsDir() {
			t.Errorf("Expected %s to be a directory", dir)
		}

		// Verify .gitkeep exists
		gitkeepPath := filepath.Join(dirPath, ".gitkeep")
		if _, err := os.Stat(gitkeepPath); os.IsNotExist(err) {
			t.Errorf("Expected .gitkeep in %s", dir)
		}
	}

	// Verify generated files exist and have content
	tests := []struct {
		file             string
		expectedContents []string
	}{
		{
			file: "buf.yaml",
			expectedContents: []string{
				"version: v2",
				"path: proto",
				"lint:",
				"STANDARD",
			},
		},
		{
			file: "CODEOWNERS",
			expectedContents: []string{
				"@infoblox/api-owners",
				"@infoblox/proto-owners",
				"@infoblox/openapi-owners",
				"@infoblox/avro-owners",
				"@infoblox/jsonschema-owners",
				"@infoblox/parquet-owners",
			},
		},
		{
			file: "catalog/catalog.yaml",
			expectedContents: []string{
				"version: 1",
				"org: infoblox",
				"repo: api-schemas",
				"modules: []",
			},
		},
		{
			file: "README.md",
			expectedContents: []string{
				"# infoblox/api-schemas",
				"Canonical API Schema Repository",
			},
		},
	}

	for _, tt := range tests {
		filePath := filepath.Join(workspaceDir, tt.file)
		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Errorf("Failed to read %s: %v", tt.file, err)
			continue
		}

		contentStr := string(content)
		for _, expected := range tt.expectedContents {
			if !strings.Contains(contentStr, expected) {
				t.Errorf("File %s should contain %q, but doesn't.\nContent:\n%s",
					tt.file, expected, contentStr)
			}
		}
	}
}

func TestCanonicalBootstrapWithGit(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if git is available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available in PATH")
	}

	// Create temporary directory for test
	tmpDir := t.TempDir()

	// Build apx binary
	apxBin := filepath.Join(tmpDir, "apx")
	buildCmd := exec.Command("go", "build", "-o", apxBin, "../../cmd/apx")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build apx: %v\n%s", err, output)
	}

	// Create test workspace
	workspaceDir := filepath.Join(tmpDir, "canonical-with-git")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	// Run apx init canonical (without skip-git)
	initCmd := exec.Command(apxBin, "init", "canonical",
		"--org=testorg",
		"--repo=test-apis",
		"--non-interactive")
	initCmd.Dir = workspaceDir
	initCmd.Env = append(os.Environ(), "NO_COLOR=1", "CI=1")

	output, err := initCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("apx init canonical failed: %v\n%s", err, output)
	}

	outputStr := string(output)

	// Verify branch protection guidance is in output
	expectedGuidance := []string{
		"Next steps:",
		"Initialize git:",
		"branch protection:",
		"Require pull request reviews",
		"Require status checks (lint, breaking)",
		"CODEOWNERS review",
	}

	for _, msg := range expectedGuidance {
		if !strings.Contains(outputStr, msg) {
			t.Errorf("Expected guidance output to contain %q, got:\n%s", msg, outputStr)
		}
	}

	// Initialize git repo and verify structure can be committed
	gitInit := exec.Command("git", "init")
	gitInit.Dir = workspaceDir
	if output, err := gitInit.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, output)
	}

	// Configure git user for test
	gitConfigName := exec.Command("git", "config", "user.name", "Test User")
	gitConfigName.Dir = workspaceDir
	gitConfigName.Run()

	gitConfigEmail := exec.Command("git", "config", "user.email", "test@example.com")
	gitConfigEmail.Dir = workspaceDir
	gitConfigEmail.Run()

	// Add all files
	gitAdd := exec.Command("git", "add", ".")
	gitAdd.Dir = workspaceDir
	if output, err := gitAdd.CombinedOutput(); err != nil {
		t.Fatalf("git add failed: %v\n%s", err, output)
	}

	// Commit
	gitCommit := exec.Command("git", "commit", "-m", "Initial canonical scaffold")
	gitCommit.Dir = workspaceDir
	if output, err := gitCommit.CombinedOutput(); err != nil {
		t.Fatalf("git commit failed: %v\n%s", err, output)
	}

	// Verify commit was successful
	gitLog := exec.Command("git", "log", "--oneline")
	gitLog.Dir = workspaceDir
	output, err = gitLog.CombinedOutput()
	if err != nil {
		t.Fatalf("git log failed: %v\n%s", err, output)
	}

	if !strings.Contains(string(output), "Initial canonical scaffold") {
		t.Errorf("Expected commit 'Initial canonical scaffold' in git log")
	}
}

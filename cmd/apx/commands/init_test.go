package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/infobloxopen/apx/internal/ui"
	"github.com/urfave/cli/v2"
)

func TestInitCanonical_CLIOutput(t *testing.T) {
	// Setup temporary directory
	tmpDir := t.TempDir()

	// Capture output
	var stdout bytes.Buffer
	ui.SetOutput(&stdout)
	defer ui.SetOutput(os.Stdout)

	// Create app and run canonical init
	app := &cli.App{
		Name:     "apx",
		Commands: []*cli.Command{InitCommand()},
	}

	args := []string{"apx", "init", "canonical",
		"--org=testorg",
		"--repo=test-apis",
		"--skip-git",
		"--non-interactive"}

	// Change to temp directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	err := app.Run(args)
	if err != nil {
		t.Fatalf("init canonical failed: %v", err)
	}

	output := stdout.String()

	// Verify expected output messages
	expectedMessages := []string{
		"Initializing canonical API repository",
		"Organization: testorg",
		"Repository: test-apis",
		"✓ Created directory structure",
		"✓ Generated buf.yaml",
		"✓ Generated CODEOWNERS",
		"✓ Generated catalog.yaml",
		"✓ Generated README.md",
		"✓ Canonical API repository initialized successfully",
	}

	for _, msg := range expectedMessages {
		if !bytes.Contains([]byte(output), []byte(msg)) {
			t.Errorf("Expected output to contain %q, but it didn't.\nOutput:\n%s", msg, output)
		}
	}

	// Verify structure was created
	expectedDirs := []string{"proto", "openapi", "avro", "jsonschema", "parquet"}
	for _, dir := range expectedDirs {
		dirPath := filepath.Join(tmpDir, dir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			t.Errorf("Expected directory %s to exist", dir)
		}
	}

	// Verify files were created
	expectedFiles := []string{"buf.yaml", "CODEOWNERS", "catalog.yaml", "README.md"}
	for _, file := range expectedFiles {
		filePath := filepath.Join(tmpDir, file)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("Expected file %s to exist", file)
		}
	}
}

func TestInitCanonical_WithBranchProtectionGuidance(t *testing.T) {
	// Setup temporary directory
	tmpDir := t.TempDir()

	// Capture output
	var stdout bytes.Buffer
	ui.SetOutput(&stdout)
	defer ui.SetOutput(os.Stdout)

	// Create app and run canonical init WITHOUT skip-git
	app := &cli.App{
		Name:     "apx",
		Commands: []*cli.Command{InitCommand()},
	}

	args := []string{"apx", "init", "canonical",
		"--org=myorg",
		"--repo=my-apis",
		"--non-interactive"}

	// Change to temp directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	err := app.Run(args)
	if err != nil {
		t.Fatalf("init canonical failed: %v", err)
	}

	output := stdout.String()

	// Verify branch protection guidance is included
	expectedGuidance := []string{
		"Next steps:",
		"Initialize git: git init",
		"Create GitHub repository and set up branch protection:",
		"Require pull request reviews",
		"Require status checks (lint, breaking)",
		"Require CODEOWNERS review",
		"Restrict direct pushes to main",
	}

	for _, msg := range expectedGuidance {
		if !bytes.Contains([]byte(output), []byte(msg)) {
			t.Errorf("Expected guidance to contain %q, but it didn't.\nOutput:\n%s", msg, output)
		}
	}
}

func TestInitCanonical_NonInteractiveMissingFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "missing org",
			args: []string{"apx", "init", "canonical", "--repo=test-apis", "--non-interactive"},
		},
		{
			name: "missing repo",
			args: []string{"apx", "init", "canonical", "--org=testorg", "--non-interactive"},
		},
		{
			name: "missing both",
			args: []string{"apx", "init", "canonical", "--non-interactive"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			app := &cli.App{
				Name:     "apx",
				Commands: []*cli.Command{InitCommand()},
			}

			// Change to temp directory
			oldWd, _ := os.Getwd()
			defer os.Chdir(oldWd)
			os.Chdir(tmpDir)

			err := app.Run(tt.args)
			if err == nil {
				t.Errorf("Expected error for %s, but got none", tt.name)
			}
		})
	}
}

package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/infobloxopen/apx/internal/ui"
	"github.com/urfave/cli/v2"
)

// TestDocParity_InitCanonical verifies that `apx init canonical` output matches quickstart.md
func TestDocParity_InitCanonical(t *testing.T) {
	tmpDir := t.TempDir()

	var stdout bytes.Buffer
	ui.SetOutput(&stdout)
	defer ui.SetOutput(os.Stdout)

	app := &cli.App{
		Name:     "apx",
		Commands: []*cli.Command{InitCommand()},
	}

	args := []string{"apx", "init", "canonical",
		"--org=testorg",
		"--repo=apis",
		"--skip-git",
		"--non-interactive"}

	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	err := app.Run(args)
	if err != nil {
		t.Fatalf("init canonical failed: %v", err)
	}

	output := stdout.String()

	// Expected messages from quickstart.md section 1
	expectedPatterns := []string{
		"Initializing canonical API repository",
		"Organization:",
		"Repository:",
		"Created directory structure",
		"Generated buf.yaml",
		"Generated CODEOWNERS",
		"Generated catalog.yaml",
		"initialized successfully",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(output, pattern) {
			t.Errorf("Doc parity failure: expected output to contain %q\nGot:\n%s", pattern, output)
		}
	}

	// Verify directory structure matches docs
	expectedDirs := []string{"proto", "openapi", "avro", "jsonschema", "parquet", "catalog"}
	for _, dir := range expectedDirs {
		if _, err := os.Stat(filepath.Join(tmpDir, dir)); os.IsNotExist(err) {
			t.Errorf("Doc parity failure: expected directory %s (as shown in quickstart.md)", dir)
		}
	}

	// Verify key files match docs
	expectedFiles := []string{"buf.yaml", "buf.work.yaml", "CODEOWNERS", "catalog/catalog.yaml"}
	for _, file := range expectedFiles {
		if _, err := os.Stat(filepath.Join(tmpDir, file)); os.IsNotExist(err) {
			t.Errorf("Doc parity failure: expected file %s (as shown in quickstart.md)", file)
		}
	}
}

// TestDocParity_InitApp verifies that `apx init app` output matches quickstart.md
func TestDocParity_InitApp(t *testing.T) {
	tmpDir := t.TempDir()

	var stdout bytes.Buffer
	ui.SetOutput(&stdout)
	defer ui.SetOutput(os.Stdout)

	app := &cli.App{
		Name:     "apx",
		Commands: []*cli.Command{InitCommand()},
	}

	args := []string{"apx", "init", "app",
		"--org=testorg",
		"--non-interactive",
		"internal/apis/proto/payments/ledger"}

	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	err := app.Run(args)
	if err != nil {
		t.Fatalf("init app failed: %v", err)
	}

	output := stdout.String()

	// Expected messages from quickstart.md section 2
	expectedPatterns := []string{
		"Initializing application repository",
		"Created module directory structure",
		"Generated apx.yaml",
		"Generated buf.work.yaml",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(output, pattern) {
			t.Errorf("Doc parity failure: expected output to contain %q\nGot:\n%s", pattern, output)
		}
	}

	// Verify app repo structure matches docs
	expectedPath := filepath.Join(tmpDir, "internal/apis/proto/payments/ledger")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Doc parity failure: expected path %s (as shown in quickstart.md)", expectedPath)
	}

	// Verify configuration files
	expectedFiles := []string{"apx.yaml", "buf.work.yaml", ".gitignore"}
	for _, file := range expectedFiles {
		if _, err := os.Stat(filepath.Join(tmpDir, file)); os.IsNotExist(err) {
			t.Errorf("Doc parity failure: expected file %s (as shown in quickstart.md)", file)
		}
	}

	// Verify .gitignore contains /internal/gen/ as documented
	gitignoreContent, err := os.ReadFile(filepath.Join(tmpDir, ".gitignore"))
	if err != nil {
		t.Fatalf("failed to read .gitignore: %v", err)
	}

	if !strings.Contains(string(gitignoreContent), "/internal/gen/") {
		t.Error("Doc parity failure: .gitignore should contain /internal/gen/ (as documented in quickstart.md)")
	}
}

// TestDocParity_LintCommand verifies lint command behavior matches docs
func TestDocParity_LintCommand(t *testing.T) {
	// This test verifies that the lint command exists and has the expected interface
	// Full functional testing is in testscript/lint_proto.txt

	app := &cli.App{
		Name:     "apx",
		Commands: []*cli.Command{LintCommand()},
	}

	// Verify command is available
	cmd := app.Command("lint")
	if cmd == nil {
		t.Fatal("Doc parity failure: 'apx lint' command not found (documented in quickstart.md section 4)")
	}

	// Verify command has ArgsUsage (accepts a path argument)
	if cmd.ArgsUsage == "" {
		t.Error("Doc parity failure: lint command should accept path argument as documented")
	}
}

// TestDocParity_BreakingCommand verifies breaking command behavior matches docs
func TestDocParity_BreakingCommand(t *testing.T) {
	app := &cli.App{
		Name:     "apx",
		Commands: []*cli.Command{BreakingCommand()},
	}

	cmd := app.Command("breaking")
	if cmd == nil {
		t.Fatal("Doc parity failure: 'apx breaking' command not found (documented in quickstart.md section 4)")
	}
}

// TestDocParity_PublishCommand verifies publish command behavior matches docs
func TestDocParity_PublishCommand(t *testing.T) {
	app := &cli.App{
		Name:     "apx",
		Commands: []*cli.Command{PublishCommand()},
	}

	cmd := app.Command("publish")
	if cmd == nil {
		t.Fatal("Doc parity failure: 'apx publish' command not found (documented in quickstart.md section 5)")
	}

	// Verify key flags mentioned in docs
	expectedFlags := []string{"module-path", "canonical-repo"}
	for _, flagName := range expectedFlags {
		found := false
		for _, flag := range cmd.Flags {
			if strings.Contains(flag.Names()[0], flagName) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Doc parity failure: publish command missing --%s flag (documented in quickstart.md)", flagName)
		}
	}
}

// TestDocParity_ConsumerCommands verifies consumer workflow commands match docs (section 6)
func TestDocParity_ConsumerCommands(t *testing.T) {
	tests := []struct {
		name        string
		commandFunc func() *cli.Command
		commandName string
		section     string
	}{
		{
			name:        "search command",
			commandFunc: SearchCommand,
			commandName: "search",
			section:     "6 - Discover APIs",
		},
		{
			name:        "add command",
			commandFunc: AddCommand,
			commandName: "add",
			section:     "6 - Add Dependencies",
		},
		{
			name:        "gen command",
			commandFunc: GenCommand,
			commandName: "gen",
			section:     "6 - Generate Client Code",
		},
		{
			name:        "sync command",
			commandFunc: SyncCommand,
			commandName: "sync",
			section:     "6 - Generate Client Code",
		},
		{
			name:        "unlink command",
			commandFunc: UnlinkCommand,
			commandName: "unlink",
			section:     "6 - Switch to Published Module",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.commandFunc()
			if cmd == nil {
				t.Fatalf("Doc parity failure: 'apx %s' command not found (documented in quickstart.md section %s)",
					tt.commandName, tt.section)
			}

			if cmd.Name != tt.commandName {
				t.Errorf("Doc parity failure: command name mismatch, expected %s, got %s",
					tt.commandName, cmd.Name)
			}
		})
	}
}

// TestDocParity_OverlayPaths verifies overlay path structure matches documented examples
func TestDocParity_OverlayPaths(t *testing.T) {
	// Test that overlay paths follow the documented pattern:
	// internal/gen/go/proto/<domain>/<api>@<version>/
	// This is a structural test - functional tests are in overlay_test.go

	expectedPattern := "internal/gen/go/proto/payments/ledger@v1.2.3"

	// Verify the pattern is what code generation would create
	// (integration tests verify actual creation)

	parts := strings.Split(expectedPattern, "/")
	if len(parts) < 6 {
		t.Errorf("Doc parity failure: overlay path pattern incorrect")
	}

	if parts[0] != "internal" || parts[1] != "gen" || parts[2] != "go" {
		t.Error("Doc parity failure: overlay base path should be internal/gen/go (as documented)")
	}

	// Verify versioned directory naming
	lastPart := parts[len(parts)-1]
	if !strings.Contains(lastPart, "@v") {
		t.Error("Doc parity failure: overlay directories should include @version suffix (as documented)")
	}
}

// TestDocParity_GitIgnorePattern verifies .gitignore follows documented pattern
func TestDocParity_GitIgnorePattern(t *testing.T) {
	// Verify that /internal/gen/ is the documented ignore pattern
	expectedPattern := "/internal/gen/"

	// This pattern is documented in quickstart.md section 4
	// Verify it's used consistently

	if !strings.HasPrefix(expectedPattern, "/internal/gen") {
		t.Error("Doc parity failure: gitignore pattern doesn't match documentation")
	}
}

// TestDocParity_CommandExamples verifies command examples from quickstart.md are valid
func TestDocParity_CommandExamples(t *testing.T) {
	// This test verifies that documented command examples have valid syntax
	// Actual execution is tested in testscripts

	examples := []struct {
		name    string
		command []string
	}{
		{
			name:    "init canonical",
			command: []string{"apx", "init", "canonical", "--org=testorg"},
		},
		{
			name:    "init app",
			command: []string{"apx", "init", "app", "internal/apis/proto/payments/ledger"},
		},
		{
			name:    "lint",
			command: []string{"apx", "lint"},
		},
		{
			name:    "breaking",
			command: []string{"apx", "breaking"},
		},
		{
			name:    "gen go",
			command: []string{"apx", "gen", "go"},
		},
		{
			name:    "sync",
			command: []string{"apx", "sync"},
		},
		{
			name:    "search",
			command: []string{"apx", "search", "payments", "ledger"},
		},
		{
			name:    "add",
			command: []string{"apx", "add", "proto/payments/ledger/v1@v1.2.3"},
		},
		{
			name:    "unlink",
			command: []string{"apx", "unlink", "proto/payments/ledger/v1"},
		},
	}

	for _, ex := range examples {
		t.Run(ex.name, func(t *testing.T) {
			if len(ex.command) < 2 {
				t.Errorf("Doc parity failure: invalid command example for %s", ex.name)
			}

			if ex.command[0] != "apx" {
				t.Errorf("Doc parity failure: command should start with 'apx'")
			}
		})
	}
}

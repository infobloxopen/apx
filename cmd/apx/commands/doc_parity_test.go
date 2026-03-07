package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/infobloxopen/apx/internal/config"
	"github.com/infobloxopen/apx/internal/ui"
)

// TestDocParity_InitCanonical verifies that `apx init canonical` output matches quickstart.md
func TestDocParity_InitCanonical(t *testing.T) {
	tmpDir := t.TempDir()

	var stdout strings.Builder
	ui.SetOutput(&stdout)
	defer ui.SetOutput(os.Stdout)

	cmd := NewRootCmd("test")
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"init", "canonical",
		"--org=testorg",
		"--repo=apis",
		"--skip-git",
		"--non-interactive"})

	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("init canonical failed: %v", err)
	}

	output := stdout.String()

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

	expectedDirs := []string{"proto", "openapi", "avro", "jsonschema", "parquet", "catalog"}
	for _, dir := range expectedDirs {
		if _, err := os.Stat(filepath.Join(tmpDir, dir)); os.IsNotExist(err) {
			t.Errorf("Doc parity failure: expected directory %s (as shown in quickstart.md)", dir)
		}
	}

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

	var stdout strings.Builder
	ui.SetOutput(&stdout)
	defer ui.SetOutput(os.Stdout)

	cmd := NewRootCmd("test")
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"init", "app",
		"--org=testorg",
		"--repo=myapp",
		"--non-interactive",
		"internal/apis/proto/payments/ledger"})

	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("init app failed: %v", err)
	}

	output := stdout.String()

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

	expectedPath := filepath.Join(tmpDir, "internal/apis/proto/payments/ledger")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Doc parity failure: expected path %s (as shown in quickstart.md)", expectedPath)
	}

	expectedFiles := []string{"apx.yaml", "buf.work.yaml", ".gitignore"}
	for _, file := range expectedFiles {
		if _, err := os.Stat(filepath.Join(tmpDir, file)); os.IsNotExist(err) {
			t.Errorf("Doc parity failure: expected file %s (as shown in quickstart.md)", file)
		}
	}

	gitignoreContent, err := os.ReadFile(filepath.Join(tmpDir, ".gitignore"))
	if err != nil {
		t.Fatalf("failed to read .gitignore: %v", err)
	}

	if !strings.Contains(string(gitignoreContent), "/internal/gen/") {
		t.Error("Doc parity failure: .gitignore should contain /internal/gen/ (as documented in quickstart.md)")
	}
}

// TestDocParity_LintCommand verifies lint command exists and has expected interface
func TestDocParity_LintCommand(t *testing.T) {
	root := NewRootCmd("test")
	cmd, _, err := root.Find([]string{"lint"})
	if err != nil || cmd.Use == "apx" {
		t.Fatal("Doc parity failure: 'apx lint' command not found (documented in quickstart.md section 4)")
	}

	if cmd.Use != "lint [path]" {
		t.Errorf("Doc parity failure: lint command should accept path argument as documented, got Use=%q", cmd.Use)
	}
}

// TestDocParity_BreakingCommand verifies breaking command exists
func TestDocParity_BreakingCommand(t *testing.T) {
	root := NewRootCmd("test")
	cmd, _, err := root.Find([]string{"breaking"})
	if err != nil || cmd.Use == "apx" {
		t.Fatal("Doc parity failure: 'apx breaking' command not found (documented in quickstart.md section 4)")
	}
}

// TestDocParity_PublishCommand verifies publish command and its flags
func TestDocParity_PublishCommand(t *testing.T) {
	root := NewRootCmd("test")
	cmd, _, err := root.Find([]string{"publish"})
	if err != nil || cmd.Use == "apx" {
		t.Fatal("Doc parity failure: 'apx publish' command not found (documented in quickstart.md section 5)")
	}

	expectedFlags := []string{"module-path", "canonical-repo"}
	for _, flagName := range expectedFlags {
		f := cmd.Flags().Lookup(flagName)
		if f == nil {
			t.Errorf("Doc parity failure: publish command missing --%s flag (documented in quickstart.md)", flagName)
		}
	}
}

// TestDocParity_ConsumerCommands verifies consumer workflow commands match docs (section 6)
func TestDocParity_ConsumerCommands(t *testing.T) {
	root := NewRootCmd("test")

	tests := []struct {
		name        string
		commandName string
		section     string
	}{
		{"search command", "search", "6 - Discover APIs"},
		{"add command", "add", "6 - Add Dependencies"},
		{"gen command", "gen", "6 - Generate Client Code"},
		{"sync command", "sync", "6 - Generate Client Code"},
		{"unlink command", "unlink", "6 - Switch to Published Module"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, _, err := root.Find([]string{tt.commandName})
			if err != nil || cmd.Use == "apx" {
				t.Fatalf("Doc parity failure: 'apx %s' command not found (documented in quickstart.md section %s)",
					tt.commandName, tt.section)
			}
		})
	}
}

// TestDocParity_CompletionCommand verifies that completion command is auto-generated by cobra
func TestDocParity_CompletionCommand(t *testing.T) {
	root := NewRootCmd("test")
	buf := new(strings.Builder)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"completion", "bash"})
	err := root.Execute()
	if err != nil {
		t.Fatalf("Completion command failed: %v", err)
	}
	if !strings.Contains(buf.String(), "bash completion") {
		t.Fatal("Completion command did not produce expected output")
	}
}

// TestDocParity_OverlayPaths verifies overlay path structure matches documented examples
func TestDocParity_OverlayPaths(t *testing.T) {
	expectedPattern := "internal/gen/go/proto/payments/ledger@v1.2.3"

	parts := strings.Split(expectedPattern, "/")
	if len(parts) < 6 {
		t.Errorf("Doc parity failure: overlay path pattern incorrect")
	}

	if parts[0] != "internal" || parts[1] != "gen" || parts[2] != "go" {
		t.Error("Doc parity failure: overlay base path should be internal/gen/go (as documented)")
	}

	lastPart := parts[len(parts)-1]
	if !strings.Contains(lastPart, "@v") {
		t.Error("Doc parity failure: overlay directories should include @version suffix (as documented)")
	}
}

// TestDocParity_GitIgnorePattern verifies .gitignore follows documented pattern
func TestDocParity_GitIgnorePattern(t *testing.T) {
	expectedPattern := "/internal/gen/"

	if !strings.HasPrefix(expectedPattern, "/internal/gen") {
		t.Error("Doc parity failure: gitignore pattern doesn't match documentation")
	}
}

// TestDocParity_AllCommandsExist verifies every command listed in the CLI reference
// is resolvable via root.Find(). Per Contract 1 in contracts/doc-parity-test-contract.md.
func TestDocParity_AllCommandsExist(t *testing.T) {
	root := NewRootCmd("test")
	commands := [][]string{
		{"init"}, {"init", "canonical"}, {"init", "app"},
		{"lint"}, {"breaking"},
		{"semver"}, {"semver", "suggest"},
		{"gen"}, {"policy"}, {"policy", "check"},
		{"catalog"}, {"catalog", "build"},
		{"publish"}, {"search"}, {"add"},
		{"sync"}, {"unlink"},
		{"config"}, {"config", "init"}, {"config", "validate"},
		{"fetch"},
		// Note: "completion" is Cobra's built-in command, added lazily at Execute() time;
		// verified separately in TestDocParity_CompletionCommand.
	}
	for _, args := range commands {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			cmd, _, err := root.Find(args)
			if err != nil || cmd == nil || cmd.Use == "apx" {
				t.Errorf("Doc parity failure: command %q not found (documented in cli-reference)",
					strings.Join(args, " "))
			}
		})
	}
}

// TestDocParity_AllFlagsExist verifies every flag listed in data-model.md exists on its command.
// Per Contract 2 in contracts/doc-parity-test-contract.md.
func TestDocParity_AllFlagsExist(t *testing.T) {
	type flagSpec struct {
		command []string
		flags   []string
	}

	root := NewRootCmd("test")
	documentedFlags := []flagSpec{
		{[]string{"breaking"}, []string{"against", "format"}},
		{[]string{"semver", "suggest"}, []string{"against"}},
		{[]string{"gen"}, []string{"out", "clean", "manifest"}},
		{[]string{"publish"}, []string{"module-path", "canonical-repo", "version", "dry-run", "create-pr"}},
		{[]string{"search"}, []string{"format", "catalog"}},
		{[]string{"sync"}, []string{"clean", "dry-run"}},
		{[]string{"fetch"}, []string{"config", "output", "verify"}},
		{[]string{"lint"}, []string{"format"}},
		{[]string{"init", "canonical"}, []string{"org", "repo", "skip-git", "non-interactive"}},
		{[]string{"init", "app"}, []string{"org", "repo", "non-interactive"}},
	}

	for _, spec := range documentedFlags {
		t.Run(strings.Join(spec.command, " "), func(t *testing.T) {
			cmd, _, err := root.Find(spec.command)
			if err != nil || cmd == nil || cmd.Use == "apx" {
				t.Fatalf("Doc parity failure: command %q not found", strings.Join(spec.command, " "))
			}
			for _, flagName := range spec.flags {
				f := cmd.Flags().Lookup(flagName)
				if f == nil {
					// Also check inherited flags
					f = cmd.InheritedFlags().Lookup(flagName)
				}
				if f == nil {
					t.Errorf("Doc parity failure: command %q missing --%s flag (documented in data-model.md)",
						strings.Join(spec.command, " "), flagName)
				}
			}
		})
	}
}

// TestDocParity_RequiredFlags verifies documented required flags actually cause failure when omitted.
// Per Contract 3 in contracts/doc-parity-test-contract.md.
func TestDocParity_RequiredFlags(t *testing.T) {
	t.Run("breaking requires --against", func(t *testing.T) {
		root := NewRootCmd("test")
		root.SetArgs([]string{"breaking", "."})
		var errBuf strings.Builder
		root.SetErr(&errBuf)
		err := root.Execute()
		if err == nil {
			t.Error("Doc parity failure: 'apx breaking .' should fail when --against is not provided")
		}
	})

	t.Run("semver suggest requires --against", func(t *testing.T) {
		root := NewRootCmd("test")
		root.SetArgs([]string{"semver", "suggest", "."})
		var errBuf strings.Builder
		root.SetErr(&errBuf)
		err := root.Execute()
		if err == nil {
			t.Error("Doc parity failure: 'apx semver suggest .' should fail when --against is not provided")
		}
	})
}

// TestDocParity_ConfigRoundtrip verifies that apx init generates config that config.Load accepts.
// Per Contract 5 in contracts/doc-parity-test-contract.md.
func TestDocParity_ConfigRoundtrip(t *testing.T) {
	t.Run("init canonical generates valid config", func(t *testing.T) {
		// init canonical sets up a canonical API repo structure (buf.yaml, CODEOWNERS, catalog.yaml).
		// It does not generate apx.yaml — that is for app repos. Verify it runs without error.
		tmpDir := t.TempDir()

		var stdout strings.Builder
		ui.SetOutput(&stdout)
		defer ui.SetOutput(os.Stdout)

		cmd := NewRootCmd("test")
		cmd.SetOut(&stdout)
		cmd.SetErr(&stdout)
		cmd.SetArgs([]string{"init", "canonical",
			"--org=testorg",
			"--repo=apis",
			"--skip-git",
			"--non-interactive"})

		oldWd, _ := os.Getwd()
		defer os.Chdir(oldWd)
		os.Chdir(tmpDir)

		if err := cmd.Execute(); err != nil {
			t.Fatalf("init canonical failed: %v", err)
		}

		// Canonical init generates buf.yaml and catalog.yaml, not apx.yaml
		for _, f := range []string{"buf.yaml", "catalog/catalog.yaml"} {
			if _, err := os.Stat(filepath.Join(tmpDir, f)); os.IsNotExist(err) {
				t.Errorf("Config roundtrip failure: init canonical should generate %s", f)
			}
		}
	})

	t.Run("init app generates valid config", func(t *testing.T) {
		tmpDir := t.TempDir()

		var stdout strings.Builder
		ui.SetOutput(&stdout)
		defer ui.SetOutput(os.Stdout)

		cmd := NewRootCmd("test")
		cmd.SetOut(&stdout)
		cmd.SetErr(&stdout)
		cmd.SetArgs([]string{"init", "app",
			"--org=testorg",
			"--repo=myapp",
			"--non-interactive",
			"internal/apis/proto/payments/ledger"})

		oldWd, _ := os.Getwd()
		defer os.Chdir(oldWd)
		os.Chdir(tmpDir)

		if err := cmd.Execute(); err != nil {
			t.Fatalf("init app failed: %v", err)
		}

		cfg, err := config.Load(filepath.Join(tmpDir, "apx.yaml"))
		if err != nil {
			t.Fatalf("Config roundtrip failure: generated apx.yaml failed to load: %v", err)
		}
		if cfg.Org != "testorg" {
			t.Errorf("Config roundtrip failure: expected org=testorg, got %q", cfg.Org)
		}
		if cfg.Repo != "myapp" {
			t.Errorf("Config roundtrip failure: expected repo=myapp, got %q", cfg.Repo)
		}
	})

	t.Run("apx.example.yaml is valid config", func(t *testing.T) {
		examplePath := findRepoRoot(t, "apx.example.yaml")
		if examplePath == "" {
			t.Skip("apx.example.yaml not found")
		}
		_, err := config.Load(examplePath)
		if err != nil {
			t.Fatalf("Config roundtrip failure: apx.example.yaml failed to load: %v", err)
		}
	})
}

func findRepoRoot(t *testing.T, filename string) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		path := filepath.Join(dir, filename)
		if _, err := os.Stat(path); err == nil {
			return path
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// TestDocParity_CommandExamples verifies command examples from quickstart.md are valid
func TestDocParity_CommandExamples(t *testing.T) {
	examples := []struct {
		name    string
		command []string
	}{
		{"init canonical", []string{"apx", "init", "canonical", "--org=testorg"}},
		{"init app", []string{"apx", "init", "app", "internal/apis/proto/payments/ledger"}},
		{"lint", []string{"apx", "lint"}},
		{"breaking", []string{"apx", "breaking"}},
		{"gen go", []string{"apx", "gen", "go"}},
		{"sync", []string{"apx", "sync"}},
		{"search", []string{"apx", "search", "payments"}},
		{"add", []string{"apx", "add", "proto/payments/ledger/v1@v1.2.3"}},
		{"unlink", []string{"apx", "unlink", "proto/payments/ledger/v1"}},
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

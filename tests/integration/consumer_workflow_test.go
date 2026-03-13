package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestConsumerWorkflow validates the full consumer experience
// following the documented workflow: init → add → gen → sync → unlink
func TestConsumerWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Build apx binary
	apxBin := filepath.Join(tmpDir, "apx")
	if runtime.GOOS == "windows" {
		apxBin += ".exe"
	}
	buildCmd := exec.Command("go", "build", "-o", apxBin, "../../cmd/apx")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build apx: %v\n%s", err, output)
	}

	workDir := filepath.Join(tmpDir, "app")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	env := append(os.Environ(), "NO_COLOR=1", "CI=1", "APX_DISABLE_TTY=1")

	run := func(name string, args ...string) string {
		t.Helper()
		cmd := exec.Command(name, args...)
		cmd.Dir = workDir
		cmd.Env = env
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s %v failed: %v\n%s", name, args, err, output)
		}
		return string(output)
	}

	// Step 0: Initialize git (required by apx init)
	run("git", "init")
	run("git", "config", "user.name", "Test User")
	run("git", "config", "user.email", "test@example.com")

	// Step 1: Initialize app repo
	output := run(apxBin, "init", "app",
		"--org=testorg", "--repo=myapp", "--non-interactive",
		"internal/apis/proto/services/users")

	if !strings.Contains(output, "Application repository initialized") {
		t.Fatalf("Expected init success message, got:\n%s", output)
	}

	// Verify apx.yaml created with correct org/repo
	apxYaml, err := os.ReadFile(filepath.Join(workDir, "apx.yaml"))
	if err != nil {
		t.Fatalf("Failed to read apx.yaml: %v", err)
	}
	if !strings.Contains(string(apxYaml), "org: testorg") {
		t.Errorf("apx.yaml missing org field")
	}
	if !strings.Contains(string(apxYaml), "repo: myapp") {
		t.Errorf("apx.yaml missing repo field")
	}

	// Step 2: Add dependencies
	output = run(apxBin, "add", "proto/payments/ledger/v1@v1.2.3")
	if !strings.Contains(output, "Added dependency") {
		t.Fatalf("Expected add success message, got:\n%s", output)
	}

	// Verify lock file has correct repo from config (not hardcoded placeholder)
	lockData, err := os.ReadFile(filepath.Join(workDir, "apx.lock"))
	if err != nil {
		t.Fatalf("Failed to read apx.lock: %v", err)
	}
	lockStr := string(lockData)
	if !strings.Contains(lockStr, "github.com/testorg/myapp") {
		t.Errorf("Lock file should contain resolved repo github.com/testorg/myapp, got:\n%s", lockStr)
	}
	if strings.Contains(lockStr, `"github.com/org/apis"`) {
		t.Errorf("Lock file should NOT contain old hardcoded placeholder")
	}

	// Add a second dependency
	run(apxBin, "add", "proto/users/profile/v1@v1.0.1")

	// Verify both dependencies in apx.yaml
	apxYaml, _ = os.ReadFile(filepath.Join(workDir, "apx.yaml"))
	yamlStr := string(apxYaml)
	if !strings.Contains(yamlStr, "proto/payments/ledger/v1") {
		t.Error("apx.yaml missing first dependency")
	}
	if !strings.Contains(yamlStr, "proto/users/profile/v1") {
		t.Error("apx.yaml missing second dependency")
	}

	// Step 3: Generate Go code
	output = run(apxBin, "gen", "go")
	if !strings.Contains(output, "Generating go code") {
		t.Fatalf("Expected gen output, got:\n%s", output)
	}

	// Verify overlay directories created
	genGoDir := filepath.Join(workDir, "internal", "gen", "go")
	assertDirContains(t, genGoDir, "proto/payments/ledger", "ledger Go overlay")
	assertDirContains(t, genGoDir, "proto/users/profile", "profile Go overlay")

	// Step 4: Sync go.work
	run(apxBin, "sync")

	// Verify go.work exists and contains overlay entries
	goWork, err := os.ReadFile(filepath.Join(workDir, "go.work"))
	if err != nil {
		t.Fatalf("Failed to read go.work: %v", err)
	}
	goWorkStr := string(goWork)
	if !strings.Contains(goWorkStr, "internal/gen/go/proto/payments/ledger") {
		t.Errorf("go.work should contain ledger overlay entry, got:\n%s", goWorkStr)
	}
	if !strings.Contains(goWorkStr, "internal/gen/go/proto/users/profile") {
		t.Errorf("go.work should contain profile overlay entry, got:\n%s", goWorkStr)
	}

	// Step 5: Generate Python code
	output = run(apxBin, "gen", "python")
	if !strings.Contains(output, "Generating python code") {
		t.Fatalf("Expected gen python output, got:\n%s", output)
	}

	// Verify Python package scaffolding
	pyOverlay := filepath.Join(workDir, "internal", "gen", "python", "proto", "payments", "ledger", "v1")
	if _, err := os.Stat(pyOverlay); os.IsNotExist(err) {
		t.Errorf("Expected Python overlay directory at %s", pyOverlay)
	} else {
		// Verify pyproject.toml exists with correct dist name
		pyprojectPath := filepath.Join(pyOverlay, "pyproject.toml")
		pyproject, err := os.ReadFile(pyprojectPath)
		if err != nil {
			t.Errorf("Failed to read pyproject.toml: %v", err)
		} else {
			pyStr := string(pyproject)
			if !strings.Contains(pyStr, `name = "testorg-payments-ledger-v1"`) {
				t.Errorf("pyproject.toml should contain dist name testorg-payments-ledger-v1, got:\n%s", pyStr)
			}
		}

		// Verify namespace __init__.py exists with pkgutil
		nsInit := filepath.Join(pyOverlay, "testorg_apis", "__init__.py")
		initContent, err := os.ReadFile(nsInit)
		if err != nil {
			t.Errorf("Failed to read namespace __init__.py: %v", err)
		} else if !strings.Contains(string(initContent), "pkgutil") {
			t.Errorf("Namespace __init__.py should use pkgutil.extend_path")
		}
	}

	// Step 6: Unlink first dependency
	output = run(apxBin, "unlink", "proto/payments/ledger/v1")
	if !strings.Contains(output, "Unlinked") {
		t.Fatalf("Expected unlink success message, got:\n%s", output)
	}

	// Verify ledger overlay removed from go.work
	goWork, _ = os.ReadFile(filepath.Join(workDir, "go.work"))
	goWorkStr = string(goWork)
	if strings.Contains(goWorkStr, "proto/payments/ledger") {
		t.Errorf("go.work should NOT contain ledger overlay after unlink, got:\n%s", goWorkStr)
	}

	// Profile overlay should still be present
	if !strings.Contains(goWorkStr, "internal/gen/go/proto/users/profile") {
		t.Errorf("go.work should still contain profile overlay after unlinking ledger, got:\n%s", goWorkStr)
	}

	// Verify unlink output includes Go and Python hints
	if !strings.Contains(output, "go get") {
		t.Errorf("Unlink output should include Go hint, got:\n%s", output)
	}
	if !strings.Contains(output, "pip install") {
		t.Errorf("Unlink output should include Python hint, got:\n%s", output)
	}

	// Step 7: Verify import paths remain stable
	// The remaining overlay should still work with the same path structure
	if !strings.Contains(goWorkStr, "proto/users/profile") {
		t.Error("Import path structure should remain consistent after partial unlink")
	}
}

// TestConsumerWorkflow_SyncGo verifies that apx sync go updates go.work
func TestConsumerWorkflow_SyncGo(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	apxBin := filepath.Join(tmpDir, "apx")
	if runtime.GOOS == "windows" {
		apxBin += ".exe"
	}
	buildCmd := exec.Command("go", "build", "-o", apxBin, "../../cmd/apx")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build apx: %v\n%s", err, output)
	}

	workDir := filepath.Join(tmpDir, "app")
	os.MkdirAll(workDir, 0755)
	env := append(os.Environ(), "NO_COLOR=1", "CI=1", "APX_DISABLE_TTY=1")

	// git init + apx init
	gitInit := exec.Command("git", "init")
	gitInit.Dir = workDir
	gitInit.Env = env
	gitInit.Run()

	initCmd := exec.Command(apxBin, "init", "app", "--org=testorg", "--repo=myapp",
		"--non-interactive", "internal/apis/proto/test/v1")
	initCmd.Dir = workDir
	initCmd.Env = env
	initCmd.Run()

	// apx sync go should succeed and update go.work
	syncCmd := exec.Command(apxBin, "sync", "go")
	syncCmd.Dir = workDir
	syncCmd.Env = env
	output, err := syncCmd.CombinedOutput()
	if err != nil {
		t.Errorf("Expected 'apx sync go' to succeed, got error: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "go.work updated") {
		t.Errorf("Expected 'apx sync go' to mention go.work updated, got:\n%s", output)
	}
}

// TestConsumerWorkflow_SyncPythonNoVenv verifies apx sync python fails without virtualenv
func TestConsumerWorkflow_SyncPythonNoVenv(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	apxBin := filepath.Join(tmpDir, "apx")
	if runtime.GOOS == "windows" {
		apxBin += ".exe"
	}
	buildCmd := exec.Command("go", "build", "-o", apxBin, "../../cmd/apx")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build apx: %v\n%s", err, output)
	}

	workDir := filepath.Join(tmpDir, "app")
	os.MkdirAll(workDir, 0755)

	// Ensure VIRTUAL_ENV is NOT set
	env := []string{
		"NO_COLOR=1", "CI=1", "APX_DISABLE_TTY=1",
		"HOME=" + tmpDir,
		"PATH=" + os.Getenv("PATH"),
		"GIT_CONFIG_NOSYSTEM=1",
	}

	// git init + apx init
	gitInit := exec.Command("git", "init")
	gitInit.Dir = workDir
	gitInit.Env = env
	gitInit.Run()

	initCmd := exec.Command(apxBin, "init", "app", "--org=testorg", "--repo=myapp",
		"--non-interactive", "internal/apis/proto/test/v1")
	initCmd.Dir = workDir
	initCmd.Env = env
	initCmd.Run()

	// apx sync python without VIRTUAL_ENV should fail (explicit language = strict error)
	syncCmd := exec.Command(apxBin, "sync", "python")
	syncCmd.Dir = workDir
	syncCmd.Env = env
	output, err := syncCmd.CombinedOutput()
	if err == nil {
		t.Errorf("Expected 'apx sync python' to fail without virtualenv, output:\n%s", output)
	}
	outStr := string(output)
	if !strings.Contains(outStr, "virtualenv") && !strings.Contains(outStr, "VIRTUAL_ENV") {
		t.Errorf("Expected error about virtualenv, got:\n%s", outStr)
	}
}

// assertDirContains checks that a directory tree contains a path prefix.
func assertDirContains(t *testing.T, baseDir, pathPrefix, label string) {
	t.Helper()
	target := filepath.Join(baseDir, filepath.FromSlash(pathPrefix))
	// Walk up to 2 levels beyond the prefix to find versioned dirs (e.g. ledger@v1.2.3)
	parent := filepath.Dir(target)
	base := filepath.Base(target)
	if _, err := os.Stat(parent); os.IsNotExist(err) {
		t.Errorf("Expected %s directory to exist: %s", label, parent)
		return
	}
	entries, err := os.ReadDir(parent)
	if err != nil {
		t.Errorf("Failed to read directory %s: %v", parent, err)
		return
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), base) {
			return // found (exact match or versioned like ledger@v1.2.3)
		}
	}
	t.Errorf("Expected %s under %s (looking for prefix %q)", label, parent, base)
}

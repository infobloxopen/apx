package apx_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

func TestScript(t *testing.T) {
	// Ensure the binary exists before running tests.
	binPath, err := filepath.Abs(filepath.Join("bin", getBinaryName("apx")))
	if err != nil {
		t.Fatalf("failed to resolve bin path: %v", err)
	}
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(binPath), 0755); err != nil {
			t.Fatalf("failed to create bin directory: %v", err)
		}
		if err := buildBinary(binPath); err != nil {
			t.Fatalf("failed to build binary: %v", err)
		}
	}

	testscript.Run(t, testscript.Params{
		Dir:                 "testdata/script",
		Setup:               setupTestScript,
		RequireExplicitExec: true,
		Condition: func(cond string) (bool, error) {
			// Support conditional test execution based on environment
			switch cond {
			case "e2e":
				// E2E tests require E2E_ENABLED=1 environment variable
				// This prevents running E2E tests in regular CI/local runs
				return os.Getenv("E2E_ENABLED") == "1", nil
			default:
				return false, nil
			}
		},
	})
}

func setupTestScript(env *testscript.Env) error {
	// Use the absolute path to the pre-built binary directory directly on PATH.
	// This avoids copying the binary per-test, which can trigger ETXTBSY
	// (text file busy) on Linux when parallel tests share the same source
	// binary while other packages rebuild it.
	absDir, err := filepath.Abs("bin")
	if err != nil {
		return fmt.Errorf("failed to resolve bin directory: %w", err)
	}

	if os.Getenv("CI") != "" {
		println("DEBUG: GOOS =", runtime.GOOS)
		println("DEBUG: absDir =", absDir)
		println("DEBUG: PathListSeparator =", string(os.PathListSeparator))
	}

	// Add the absolute bin directory to PATH
	newPath := absDir + string(os.PathListSeparator) + env.Getenv("PATH")
	env.Setenv("PATH", newPath)

	// Set testing environment variables
	env.Setenv("APX_DISABLE_TTY", "1")
	env.Setenv("NO_COLOR", "1")
	env.Setenv("CI", "1")

	return nil
}

// buildBinary builds the apx binary to the specified path
func buildBinary(destPath string) error {
	cmd := exec.Command("go", "build", "-o", destPath, "./cmd/apx")
	cmd.Env = os.Environ()
	return cmd.Run()
}

// getBinaryName returns the correct binary name for the current OS
func getBinaryName(baseName string) string {
	if runtime.GOOS == "windows" {
		return baseName + ".exe"
	}
	return baseName
}

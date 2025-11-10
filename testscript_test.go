package apx_test

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

func TestScript(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir:                 "testdata/script",
		Setup:               setupTestScript,
		RequireExplicitExec: true,
	})
}

func setupTestScript(env *testscript.Env) error {
	// Create bin directory in the test workspace
	binDir := filepath.Join(env.WorkDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}

	// Get the correct binary names for the current OS
	preBuildBinaryName := getBinaryName("apx")
	destBinaryName := getBinaryName("apx")

	// Debug logging
	if os.Getenv("CI") != "" {
		println("DEBUG: GOOS =", runtime.GOOS)
		println("DEBUG: preBuildBinaryName =", preBuildBinaryName)
		println("DEBUG: destBinaryName =", destBinaryName)
		println("DEBUG: binDir =", binDir)
	}

	// Check if the binary already exists in ./bin/ (built by CI)
	apxBinaryPath := filepath.Join(".", "bin", preBuildBinaryName)
	if _, err := os.Stat(apxBinaryPath); err == nil {
		if os.Getenv("CI") != "" {
			println("DEBUG: Found pre-built binary at", apxBinaryPath)
		}
		// Copy the pre-built binary to the test workspace
		destPath := filepath.Join(binDir, destBinaryName)
		if err := copyFile(apxBinaryPath, destPath); err != nil {
			return err
		}
		if err := os.Chmod(destPath, 0755); err != nil {
			return err
		}
		if os.Getenv("CI") != "" {
			println("DEBUG: Copied binary to", destPath)
			// Verify the binary exists and is executable
			if stat, err := os.Stat(destPath); err == nil {
				println("DEBUG: Binary exists, size:", stat.Size(), "mode:", stat.Mode())
			} else {
				println("DEBUG: Error stating binary:", err)
			}
		}
	} else {
		if os.Getenv("CI") != "" {
			println("DEBUG: Pre-built binary not found at", apxBinaryPath, "- building fresh")
		}
		// Binary doesn't exist, build it in the test workspace
		destPath := filepath.Join(binDir, destBinaryName)
		if err := buildBinary(destPath); err != nil {
			return err
		}
		if os.Getenv("CI") != "" {
			println("DEBUG: Built binary at", destPath)
			// Verify the binary exists and is executable
			if stat, err := os.Stat(destPath); err == nil {
				println("DEBUG: Binary exists, size:", stat.Size(), "mode:", stat.Mode())
			} else {
				println("DEBUG: Error stating binary:", err)
			}
		}
	}

	// Add the bin directory to PATH
	newPath := binDir + string(os.PathListSeparator) + env.Getenv("PATH")
	env.Setenv("PATH", newPath)
	if os.Getenv("CI") != "" {
		println("DEBUG: PATH =", newPath[:100], "...")
		println("DEBUG: PathListSeparator =", string(os.PathListSeparator))
	}

	// Set testing environment variables
	env.Setenv("APX_DISABLE_TTY", "1")
	env.Setenv("NO_COLOR", "1")
	env.Setenv("CI", "1")

	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
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

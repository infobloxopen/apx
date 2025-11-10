package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	ciEnv        = "CI=1"
	noColorEnv   = "NO_COLOR=1"
	disableTTY   = "APX_DISABLE_TTY=1"
	cmdPath      = "./cmd/apx"
	buildFailMsg = "Failed to build binary"
)

// getBinaryName returns the correct binary name for the current OS
func getBinaryName(baseName string) string {
	if runtime.GOOS == "windows" {
		return baseName + ".exe"
	}
	return baseName
}

// getBinaryPath returns the path to the binary for building
func getBinaryPath() string {
	return filepath.Join(".", "bin", getBinaryName("apx"))
}

// getRelativeBinaryPath returns the relative path to the binary from test directories
func getRelativeBinaryPath() string {
	return filepath.Join("..", "..", "bin", getBinaryName("apx"))
}

func TestBinaryExecution(t *testing.T) {
	// Debug logging
	if os.Getenv("CI") != "" {
		t.Logf("DEBUG: GOOS = %s", runtime.GOOS)
		t.Logf("DEBUG: getBinaryPath() = %s", getBinaryPath())
		t.Logf("DEBUG: getRelativeBinaryPath() = %s", getRelativeBinaryPath())
	}

	// Build the binary first
	buildCmd := exec.Command("go", "build", "-o", getBinaryPath(), cmdPath)
	buildCmd.Dir = "../.."
	err := buildCmd.Run()
	require.NoError(t, err, buildFailMsg)

	// Check if binary was actually created
	binaryPath := getBinaryPath()
	if _, err := os.Stat(filepath.Join("..", "..", binaryPath)); err != nil {
		t.Fatalf("Binary was not created at %s: %v", binaryPath, err)
	}

	// Test basic commands
	apxBinary := getRelativeBinaryPath()
	if os.Getenv("CI") != "" {
		t.Logf("DEBUG: About to execute binary at: %s", apxBinary)
	}

	// Test version command
	versionCmd := exec.Command(apxBinary, "--version")
	versionCmd.Env = append(os.Environ(), ciEnv, noColorEnv, disableTTY)
	output, err := versionCmd.Output()
	require.NoError(t, err)
	require.Contains(t, string(output), "apx version")

	// Test help command
	helpCmd := exec.Command(apxBinary, "help")
	helpCmd.Env = append(os.Environ(), ciEnv, noColorEnv, disableTTY)
	output, err = helpCmd.Output()
	require.NoError(t, err)
	require.Contains(t, string(output), "API Publishing eXperience CLI")
	require.Contains(t, string(output), "USAGE:")
	require.Contains(t, string(output), "COMMANDS:")
}

func TestConfigCommands(t *testing.T) {
	// Build the binary first
	buildCmd := exec.Command("go", "build", "-o", getBinaryPath(), cmdPath)
	buildCmd.Dir = "../.."
	err := buildCmd.Run()
	require.NoError(t, err, buildFailMsg)

	// Get absolute path to binary before changing directories
	oldCwd, _ := os.Getwd()
	apxBinary, err := filepath.Abs(getRelativeBinaryPath())
	require.NoError(t, err)

	// Create temporary directory for config test
	tmpDir := t.TempDir()
	defer func() { _ = os.Chdir(oldCwd) }()
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Test config init
	initCmd := exec.Command(apxBinary, "config", "init")
	initCmd.Env = append(os.Environ(), ciEnv, noColorEnv, disableTTY)
	err = initCmd.Run()
	require.NoError(t, err)

	// Check that apx.yaml was created
	require.FileExists(t, "apx.yaml")

	// Test config validate
	validateCmd := exec.Command(apxBinary, "config", "validate")
	validateCmd.Env = append(os.Environ(), ciEnv, noColorEnv, disableTTY)
	output, err := validateCmd.Output()
	require.NoError(t, err)
	require.Contains(t, string(output), "Configuration is valid")
}

func TestErrorHandling(t *testing.T) {
	// Build the binary first
	buildCmd := exec.Command("go", "build", "-o", getBinaryPath(), cmdPath)
	buildCmd.Dir = "../.."
	err := buildCmd.Run()
	require.NoError(t, err, buildFailMsg)

	apxBinary := getRelativeBinaryPath()

	// Test with invalid command
	invalidCmd := exec.Command(apxBinary, "nonexistent")
	invalidCmd.Env = append(os.Environ(), ciEnv, noColorEnv, disableTTY)
	err = invalidCmd.Run()
	require.Error(t, err)

	if exitError, ok := err.(*exec.ExitError); ok {
		require.NotEqual(t, 0, exitError.ExitCode())
	}
}

func TestDeterministicOutput(t *testing.T) {
	// Build the binary first
	buildCmd := exec.Command("go", "build", "-o", getBinaryPath(), cmdPath)
	buildCmd.Dir = "../.."
	err := buildCmd.Run()
	require.NoError(t, err, buildFailMsg)

	apxBinary := getRelativeBinaryPath()

	// Run the same command multiple times and verify output is identical
	var outputs []string

	for i := 0; i < 3; i++ {
		cmd := exec.Command(apxBinary, "help")
		cmd.Env = append(os.Environ(), ciEnv, noColorEnv, disableTTY)
		output, err := cmd.Output()
		require.NoError(t, err)
		outputs = append(outputs, string(output))
	}

	// All outputs should be identical
	for i := 1; i < len(outputs); i++ {
		require.Equal(t, outputs[0], outputs[i], "Output should be deterministic")
	}
}

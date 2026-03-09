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

// getBinaryPath returns the path to the binary relative to the repo root
func getBinaryPath() string {
	return filepath.Join(".", "bin", getBinaryName("apx"))
}

// getRelativeBinaryPath returns the relative path to the binary from test directories
func getRelativeBinaryPath() string {
	return filepath.Join("..", "..", "bin", getBinaryName("apx"))
}

// TestMain builds the binary once before all integration tests run.
// Building in each test function is wasteful and causes ETXTBSY (text file
// busy) on Linux when multiple tests rebuild the same binary in parallel
// with testscript tests reading from it.
func TestMain(m *testing.M) {
	absPath, err := filepath.Abs(getRelativeBinaryPath())
	if err != nil {
		panic("failed to resolve binary path: " + err.Error())
	}

	// Only build if the binary doesn't exist (CI pre-builds it; local dev may not)
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
			panic("failed to create bin directory: " + err.Error())
		}
		buildCmd := exec.Command("go", "build", "-o", getBinaryPath(), cmdPath)
		buildCmd.Dir = "../.."
		if err := buildCmd.Run(); err != nil {
			panic("failed to build binary: " + err.Error())
		}
	}

	os.Exit(m.Run())
}

func TestBinaryExecution(t *testing.T) {
	apxBinary := getRelativeBinaryPath()

	// Test version command
	versionCmd := exec.Command(apxBinary, "--version")
	versionCmd.Env = append(os.Environ(), ciEnv, noColorEnv, disableTTY)
	output, err := versionCmd.Output()
	require.NoError(t, err)
	require.Contains(t, string(output), "apx")

	// Test help command
	helpCmd := exec.Command(apxBinary, "help")
	helpCmd.Env = append(os.Environ(), ciEnv, noColorEnv, disableTTY)
	output, err = helpCmd.Output()
	require.NoError(t, err)
	require.Contains(t, string(output), "API schemas across organizations")
	require.Contains(t, string(output), "Usage:")
	require.Contains(t, string(output), "Available Commands:")
}

func TestConfigCommands(t *testing.T) {
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
	apxBinary := getRelativeBinaryPath()

	// Test with invalid command - urfave/cli shows help for unknown commands
	// and returns 0, so we test with invalid flags instead which should error
	invalidCmd := exec.Command(apxBinary, "init", "--invalid-flag-that-does-not-exist")
	invalidCmd.Env = append(os.Environ(), ciEnv, noColorEnv, disableTTY)
	err := invalidCmd.Run()
	require.Error(t, err, "Invalid flag should cause an error")

	if exitError, ok := err.(*exec.ExitError); ok {
		require.NotEqual(t, 0, exitError.ExitCode(), "Exit code should be non-zero for invalid flag")
	}
}

func TestDeterministicOutput(t *testing.T) {
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

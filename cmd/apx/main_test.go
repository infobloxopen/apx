package main

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/infobloxopen/apx/internal/testhelpers"
	"github.com/stretchr/testify/require"
)

func TestNewApp(t *testing.T) {
	app := NewApp()
	require.NotNil(t, app)
	require.Equal(t, "apx", app.Name)
	require.Equal(t, "API Publishing eXperience CLI", app.Usage)
}

func TestHelpCommand(t *testing.T) {
	// Setup test output
	output := testhelpers.NewTestOutput()
	output.Setup()
	defer output.Restore()

	app := NewApp()
	err := app.RunContext(context.Background(), []string{"apx", "help"})
	require.NoError(t, err)

	// Restore to flush all output before reading
	output.Restore()

	stdout := output.StdoutString()
	require.Contains(t, stdout, "API Publishing eXperience CLI")
	require.Contains(t, stdout, "USAGE:")
	require.Contains(t, stdout, "COMMANDS:")
}

func TestVersionFlag(t *testing.T) {
	// Setup test output
	output := testhelpers.NewTestOutput()
	output.Setup()
	defer output.Restore()

	app := NewApp()
	err := app.RunContext(context.Background(), []string{"apx", "--version"})
	require.NoError(t, err)

	// Restore to flush all output before reading
	output.Restore()

	stdout := output.StdoutString()
	require.Contains(t, stdout, "dev") // default version
}

func TestConfigInitCommand(t *testing.T) {
	// Change to temp directory
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldDir)

	// Setup test output
	output := testhelpers.NewTestOutput()
	output.Setup()
	defer output.Restore()

	app := NewApp()
	err := app.RunContext(context.Background(), []string{"apx", "config", "init"})

	// Restore to flush all output before reading
	output.Restore()

	// This should succeed and create apx.yaml
	require.NoError(t, err)

	// Check that apx.yaml was created
	_, err = os.Stat("apx.yaml")
	require.NoError(t, err)
}

func TestExitCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{
			name:     "no error",
			err:      nil,
			expected: 0,
		},
		{
			name:     "generic error",
			err:      fmt.Errorf("test error"),
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := exitCode(tt.err)
			require.Equal(t, tt.expected, code)
		})
	}
}

func TestGlobalFlags(t *testing.T) {
	// Test that help output is captured (both quiet and verbose modes show CLI help)
	output := testhelpers.NewTestOutput()
	output.Setup()
	defer output.Restore()

	app := NewApp()
	err := app.RunContext(context.Background(), []string{"apx", "--verbose", "help"})
	require.NoError(t, err)

	// Restore to flush all output
	output.Restore()

	// Should have normal output containing help
	stdout := output.StdoutString()
	require.Contains(t, stdout, "USAGE:")
	require.Contains(t, stdout, "API Publishing eXperience CLI")
}

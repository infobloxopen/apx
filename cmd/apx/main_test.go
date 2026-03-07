package main

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewApp(t *testing.T) {
	app := NewApp()
	require.NotNil(t, app)
	require.Equal(t, "apx", app.Use)
	require.Equal(t, "API Publishing eXperience CLI", app.Short)
}

func TestHelpCommand(t *testing.T) {
	app := NewApp()
	buf := new(bytes.Buffer)
	app.SetOut(buf)
	app.SetErr(buf)
	app.SetArgs([]string{"help"})
	err := app.Execute()
	require.NoError(t, err)

	stdout := buf.String()
	require.Contains(t, stdout, "API schemas across organizations")
	require.Contains(t, stdout, "Usage:")
	require.Contains(t, stdout, "Available Commands:")
}

func TestVersionFlag(t *testing.T) {
	app := NewApp()
	buf := new(bytes.Buffer)
	app.SetOut(buf)
	app.SetErr(buf)
	app.SetArgs([]string{"--version"})
	err := app.Execute()
	require.NoError(t, err)

	stdout := buf.String()
	require.Contains(t, stdout, "dev")
}

func TestConfigInitCommand(t *testing.T) {
	// Change to temp directory
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldDir)

	app := NewApp()
	buf := new(bytes.Buffer)
	app.SetOut(buf)
	app.SetErr(buf)
	app.SetArgs([]string{"config", "init"})
	err := app.Execute()

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
	app := NewApp()
	buf := new(bytes.Buffer)
	app.SetOut(buf)
	app.SetErr(buf)
	app.SetArgs([]string{"--verbose", "help"})
	err := app.Execute()
	require.NoError(t, err)

	stdout := buf.String()
	require.Contains(t, stdout, "Usage:")
	require.Contains(t, stdout, "API schemas across organizations")
}

func TestCompletionCommand(t *testing.T) {
	app := NewApp()
	buf := new(bytes.Buffer)
	app.SetOut(buf)
	app.SetErr(buf)

	// Cobra auto-generates a completion command
	app.SetArgs([]string{"completion", "bash"})
	err := app.Execute()
	require.NoError(t, err)

	stdout := buf.String()
	require.Contains(t, stdout, "bash completion")
}

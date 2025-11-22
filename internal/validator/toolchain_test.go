package validator

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestToolchainResolver(t *testing.T) {
	t.Run("resolves buf from PATH", func(t *testing.T) {
		// Skip on Windows in CI - buf.exe is in $HOME/bin which isn't in PATH yet
		if runtime.GOOS == "windows" && os.Getenv("CI") == "1" {
			t.Skip("Skipping on Windows CI - buf not in PATH")
		}

		resolver := NewToolchainResolver()

		path, err := resolver.ResolveTool("buf", "v1.45.0")
		require.NoError(t, err)
		require.NotEmpty(t, path)

		// Verify the tool is executable
		info, err := os.Stat(path)
		require.NoError(t, err)
		require.NotEqual(t, 0, info.Mode()&0111, "buf should be executable")
	})

	t.Run("returns error for missing tool", func(t *testing.T) {
		resolver := NewToolchainResolver()

		_, err := resolver.ResolveTool("nonexistent-tool", "v1.0.0")
		require.Error(t, err)
		require.Contains(t, err.Error(), "tool not found")
	})

	t.Run("resolves from offline bundle", func(t *testing.T) {
		tmpDir := t.TempDir()
		bundleDir := filepath.Join(tmpDir, "bin")
		require.NoError(t, os.MkdirAll(bundleDir, 0755))

		// Create a fake tool binary
		toolPath := filepath.Join(bundleDir, "spectral")
		require.NoError(t, os.WriteFile(toolPath, []byte("#!/bin/sh\necho test"), 0755))

		resolver := NewToolchainResolver(WithBundlePath(bundleDir))

		path, err := resolver.ResolveTool("spectral", "v6.15.0")
		require.NoError(t, err)
		require.Equal(t, toolPath, path)
	})
}

func TestToolchainProfile(t *testing.T) {
	t.Run("loads profile from apx.lock", func(t *testing.T) {
		tmpDir := t.TempDir()
		lockPath := filepath.Join(tmpDir, "apx.lock")

		lockContent := `version: 1
tools:
  buf:
    version: v1.45.0
    checksum: abc123
  spectral:
    version: v6.15.0
    checksum: def456
`
		require.NoError(t, os.WriteFile(lockPath, []byte(lockContent), 0644))

		profile, err := LoadToolchainProfile(lockPath)
		require.NoError(t, err)
		require.NotNil(t, profile)
		require.Len(t, profile.Tools, 2)
		require.Equal(t, "v1.45.0", profile.Tools["buf"].Version)
		require.Equal(t, "abc123", profile.Tools["buf"].Checksum)
	})

	t.Run("returns error for missing lock file", func(t *testing.T) {
		_, err := LoadToolchainProfile("/nonexistent/apx.lock")
		require.Error(t, err)
	})

	t.Run("validates tool versions", func(t *testing.T) {
		profile := &ToolchainProfile{
			Version: 1,
			Tools: map[string]ToolRef{
				"buf": {Version: "v1.45.0", Checksum: "abc"},
			},
		}

		err := profile.ValidateTool("buf", "v1.45.0")
		require.NoError(t, err)

		err = profile.ValidateTool("buf", "v1.44.0")
		require.Error(t, err)
		require.Contains(t, err.Error(), "version mismatch")
	})
}

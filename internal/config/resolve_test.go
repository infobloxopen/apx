package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveAPIPath_ExistingPath(t *testing.T) {
	// An existing directory should be returned as-is (absolute).
	tmpDir := t.TempDir()
	got, err := ResolveAPIPath(tmpDir, nil)
	require.NoError(t, err)
	assert.Equal(t, tmpDir, got)
}

func TestResolveAPIPath_ExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "test.proto")
	require.NoError(t, os.WriteFile(f, []byte("syntax = \"proto3\";"), 0644))

	got, err := ResolveAPIPath(f, nil)
	require.NoError(t, err)
	assert.Equal(t, f, got)
}

func TestResolveAPIPath_APIID_DirectRelative(t *testing.T) {
	// Create proto/payments/ledger/v1 directory structure in a tmpdir,
	// then chdir into it so ResolveAPIPath finds it via the fallback.
	tmpDir := t.TempDir()
	// Resolve symlinks (macOS /var → /private/var)
	tmpDir, _ = filepath.EvalSymlinks(tmpDir)
	apiDir := filepath.Join(tmpDir, "proto", "payments", "ledger", "v1")
	require.NoError(t, os.MkdirAll(apiDir, 0755))

	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmpDir))
	defer os.Chdir(origDir)

	got, err := ResolveAPIPath("proto/payments/ledger/v1", nil)
	require.NoError(t, err)
	assert.Equal(t, apiDir, got)
}

func TestResolveAPIPath_APIID_ModuleRoots(t *testing.T) {
	tmpDir := t.TempDir()
	root := filepath.Join(tmpDir, "schemas")
	apiDir := filepath.Join(root, "proto", "billing", "invoices", "v2")
	require.NoError(t, os.MkdirAll(apiDir, 0755))

	cfg := &Config{
		ModuleRoots: []string{root},
	}

	// chdir somewhere else so fallback won't find it
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmpDir))
	defer os.Chdir(origDir)

	got, err := ResolveAPIPath("proto/billing/invoices/v2", cfg)
	require.NoError(t, err)
	assert.Equal(t, apiDir, got)
}

func TestResolveAPIPath_APIID_FallbackSchemas(t *testing.T) {
	tmpDir := t.TempDir()
	// Resolve symlinks (macOS /var → /private/var)
	tmpDir, _ = filepath.EvalSymlinks(tmpDir)
	apiDir := filepath.Join(tmpDir, "schemas", "avro", "events", "click", "v1")
	require.NoError(t, os.MkdirAll(apiDir, 0755))

	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmpDir))
	defer os.Chdir(origDir)

	got, err := ResolveAPIPath("avro/events/click/v1", nil)
	require.NoError(t, err)
	assert.Equal(t, apiDir, got)
}

func TestResolveAPIPath_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmpDir))
	defer os.Chdir(origDir)

	_, err := ResolveAPIPath("proto/payments/ledger/v1", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not resolve API ID")
}

func TestResolveAPIPath_InvalidArgument(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmpDir))
	defer os.Chdir(origDir)

	// Not a valid path and not a valid API ID
	_, err := ResolveAPIPath("not-a-path-or-id", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a valid API ID")
}

func TestResolveAPIFormat(t *testing.T) {
	tests := []struct {
		arg  string
		want string
	}{
		{"proto/payments/ledger/v1", "proto"},
		{"openapi/billing/invoices/v2", "openapi"},
		{"avro/events/click/v3", "avro"},
		{"not-an-api-id", ""},
		{"./some/path", ""},
	}
	for _, tt := range tests {
		t.Run(tt.arg, func(t *testing.T) {
			assert.Equal(t, tt.want, ResolveAPIFormat(tt.arg))
		})
	}
}

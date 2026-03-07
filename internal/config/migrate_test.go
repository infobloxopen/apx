package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T035: MigrateFile on an already-current file returns Migrated=false with no changes.
func TestMigrateFile_AlreadyCurrent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "apx.yaml")

	content := `version: 1
org: myorg
repo: myrepo
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	result, err := MigrateFile(path)
	require.NoError(t, err)
	assert.False(t, result.Migrated)
	assert.Equal(t, CurrentSchemaVersion, result.FromVersion)
	assert.Equal(t, CurrentSchemaVersion, result.ToVersion)
	assert.Empty(t, result.Changes)
	assert.Empty(t, result.Backup)
}

// T036: MigrateFile on a future version returns an error.
func TestMigrateFile_FutureVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "apx.yaml")

	content := `version: 999
org: myorg
repo: myrepo
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	result, err := MigrateFile(path)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "999")
	assert.Contains(t, err.Error(), "upgrade")
}

// T036 extra: MigrateFile on version 0 returns unsupported.
func TestMigrateFile_UnsupportedVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "apx.yaml")

	content := `version: 0
org: myorg
repo: myrepo
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	_, err := MigrateFile(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported")
}

// T037: backupFile creates a .bak file with the correct content.
func TestBackupFile_CreatesBackup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "apx.yaml")
	content := []byte("version: 1\norg: test\n")
	require.NoError(t, os.WriteFile(path, content, 0644))

	bakName, err := backupFile(path)
	require.NoError(t, err)
	assert.Equal(t, "apx.yaml.bak", bakName)

	bakPath := filepath.Join(dir, bakName)
	bakData, err := os.ReadFile(bakPath)
	require.NoError(t, err)
	assert.Equal(t, content, bakData)
}

// T037 extra: backupFile with existing .bak uses timestamp suffix.
func TestBackupFile_TimestampFallback(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "apx.yaml")
	content := []byte("version: 1\norg: test\n")
	require.NoError(t, os.WriteFile(path, content, 0644))

	// Create existing .bak
	require.NoError(t, os.WriteFile(path+".bak", []byte("old backup"), 0644))

	bakName, err := backupFile(path)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(bakName, "apx.yaml.bak."), "backup name should have timestamp: %s", bakName)
	assert.NotEqual(t, "apx.yaml.bak", bakName)

	bakPath := filepath.Join(dir, bakName)
	bakData, err := os.ReadFile(bakPath)
	require.NoError(t, err)
	assert.Equal(t, content, bakData)

	// Original .bak should be untouched
	oldBak, err := os.ReadFile(path + ".bak")
	require.NoError(t, err)
	assert.Equal(t, []byte("old backup"), oldBak)
}

// T037 extra: MigrateFile on non-existent file returns error.
func TestMigrateFile_NonExistent(t *testing.T) {
	_, err := MigrateFile("/nonexistent/path/apx.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read")
}

// T037 extra: MigrateFile on invalid YAML returns error.
func TestMigrateFile_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "apx.yaml")
	require.NoError(t, os.WriteFile(path, []byte("{{invalid yaml"), 0644))

	_, err := MigrateFile(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse")
}

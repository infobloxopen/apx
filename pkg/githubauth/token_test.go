package githubauth

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withTempConfigDir(t *testing.T) (restore func()) {
	t.Helper()
	tmp := t.TempDir()

	orig := ConfigDir
	ConfigDir = func() (string, error) { return tmp, nil }
	return func() { ConfigDir = orig }
}

func TestSaveAndLoadToken(t *testing.T) {
	restore := withTempConfigDir(t)
	defer restore()

	tok := &Token{
		AccessToken: "ghu_abc123",
		TokenType:   "bearer",
		Scope:       "repo",
		CreatedAt:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	err := SaveToken("myorg", tok)
	require.NoError(t, err)

	loaded, err := LoadToken("myorg")
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, "ghu_abc123", loaded.AccessToken)
	assert.Equal(t, "bearer", loaded.TokenType)
	assert.Equal(t, "repo", loaded.Scope)
}

func TestLoadToken_NotFound(t *testing.T) {
	restore := withTempConfigDir(t)
	defer restore()

	tok, err := LoadToken("noorg")
	require.NoError(t, err)
	assert.Nil(t, tok)
}

func TestClearToken(t *testing.T) {
	restore := withTempConfigDir(t)
	defer restore()

	tok := &Token{AccessToken: "ghu_to_clear"}
	require.NoError(t, SaveToken("myorg", tok))

	require.NoError(t, ClearToken("myorg"))

	loaded, err := LoadToken("myorg")
	require.NoError(t, err)
	assert.Nil(t, loaded)
}

func TestTokenPath(t *testing.T) {
	restore := withTempConfigDir(t)
	defer restore()

	p, err := TokenPath("acme")
	require.NoError(t, err)
	assert.True(t, filepath.IsAbs(p))
	assert.Contains(t, p, "apx-acme-user-token.json")
}

func TestTokenFilePermissions(t *testing.T) {
	restore := withTempConfigDir(t)
	defer restore()

	tok := &Token{AccessToken: "ghu_secret"}
	require.NoError(t, SaveToken("myorg", tok))

	p, _ := TokenPath("myorg")
	info, err := os.Stat(p)
	require.NoError(t, err)
	// File should be readable/writable only by owner (0600).
	// Windows does not support Unix file permission bits.
	if runtime.GOOS != "windows" {
		assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
	}
}

func TestReadWriteCache(t *testing.T) {
	restore := withTempConfigDir(t)
	defer restore()

	err := WriteCache("myorg", "user-app-client-id", "Iv1.abcdef123456")
	require.NoError(t, err)

	val, err := ReadCache("myorg", "user-app-client-id")
	require.NoError(t, err)
	assert.Equal(t, "Iv1.abcdef123456", val)
}

func TestReadCache_NotFound(t *testing.T) {
	restore := withTempConfigDir(t)
	defer restore()

	val, err := ReadCache("myorg", "nonexistent")
	require.NoError(t, err)
	assert.Equal(t, "", val)
}

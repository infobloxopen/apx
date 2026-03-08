package publisher

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some files
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "a.proto"), []byte("syntax = \"proto3\";"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "sub"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "sub", "b.proto"), []byte("message Foo {}"), 0o644))

	hash1, err := HashDirectory(tmpDir)
	require.NoError(t, err)
	assert.NotEmpty(t, hash1)

	// Same content should produce the same hash
	hash2, err := HashDirectory(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, hash1, hash2)

	// Different content should produce a different hash
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "a.proto"), []byte("syntax = \"proto3\";\nmessage Bar {}"), 0o644))
	hash3, err := HashDirectory(tmpDir)
	require.NoError(t, err)
	assert.NotEqual(t, hash1, hash3)
}

func TestHashDirectory_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	hash, err := HashDirectory(tmpDir)
	require.NoError(t, err)
	// Empty directory should still produce a valid (but empty) hash
	assert.NotEmpty(t, hash)
}

func TestHashDirectory_NotFound(t *testing.T) {
	_, err := HashDirectory("/nonexistent/dir")
	assert.Error(t, err)
}

func TestPublishError_Error(t *testing.T) {
	e := NewPublishError(ErrCodeVersionTaken, "version v1.0.0 already taken")
	assert.Contains(t, e.Error(), "VERSION_TAKEN")
	assert.Contains(t, e.Error(), "version v1.0.0 already taken")

	// With hint
	eHint := e.WithHint("choose a different version")
	assert.Contains(t, eHint.Error(), "choose a different version")
}

func TestPublishError_AllCodesHaveDescriptions(t *testing.T) {
	codes := []PublishErrorCode{
		ErrCodeVersionTaken, ErrCodeLifecycleBlocked, ErrCodeLifecycleMismatch,
		ErrCodeValidationFailed, ErrCodeGoPackageMismatch, ErrCodeGoModMismatch,
		ErrCodeMergeConflict, ErrCodeCanonicalMoved, ErrCodePolicyFailed,
		ErrCodePackagePublishFailed, ErrCodeCatalogUpdateFailed,
		ErrCodeNotGitRepo, ErrCodeInvalidVersion, ErrCodeMissingConfig,
		ErrCodePushFailed, ErrCodePRCreationFailed,
	}

	for _, code := range codes {
		desc, ok := ErrorCodeDescriptions[code]
		assert.True(t, ok, "error code %s should have a description", code)
		assert.NotEmpty(t, desc)
	}
}

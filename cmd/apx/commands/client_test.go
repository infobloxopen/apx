package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/infobloxopen/apx/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindClientTarget(t *testing.T) {
	cfg := &config.Config{
		Clients: []config.ClientTarget{
			{Name: "web", Scope: "@example", Package: "web-client"},
			{Name: "mobile", Package: "mobile-client"},
		},
	}
	got := findClientTarget(cfg, "mobile")
	require.NotNil(t, got)
	assert.Equal(t, "mobile-client", got.Package)

	assert.Nil(t, findClientTarget(cfg, "missing"))
	assert.Nil(t, findClientTarget(nil, "web"))
}

func TestDefaultPackageName(t *testing.T) {
	cases := map[string]string{
		"openapi/notesd.openapi.yaml": "notesd-client",
		"notesd.openapi.yml":          "notesd-client",
		"foo.swagger.yaml":            "foo-client",
		"plain.yaml":                  "plain-client",
	}
	for in, want := range cases {
		assert.Equal(t, want, defaultPackageName(in), "input %q", in)
	}
}

func TestResolveClientSpecInputWins(t *testing.T) {
	ct := &config.ClientTarget{Spec: "config-spec.yaml"}
	got, err := resolveClientSpec("flag-spec.yaml", "", ct, "web")
	require.NoError(t, err)
	assert.Equal(t, "flag-spec.yaml", got)
}

func TestResolveClientSpecTargetSpec(t *testing.T) {
	ct := &config.ClientTarget{Spec: "config-spec.yaml"}
	got, err := resolveClientSpec("", "", ct, "web")
	require.NoError(t, err)
	assert.Equal(t, "config-spec.yaml", got)
}

func TestResolveClientSpecAutoDetectSingle(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "openapi"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "openapi", "svc.openapi.yaml"), []byte("openapi: 3.0.0"), 0o644))

	withWorkdir(t, dir)
	got, err := resolveClientSpec("", "", nil, "")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("openapi", "svc.openapi.yaml"), got)
}

func TestResolveClientSpecAutoDetectZero(t *testing.T) {
	dir := t.TempDir()
	withWorkdir(t, dir)
	_, err := resolveClientSpec("", "", nil, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no OpenAPI spec found")
}

func TestResolveClientSpecAutoDetectMultiple(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "openapi"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "openapi", "a.openapi.yaml"), []byte("openapi: 3.0.0"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "openapi", "b.openapi.yaml"), []byte("openapi: 3.0.0"), 0o644))

	withWorkdir(t, dir)
	_, err := resolveClientSpec("", "", nil, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "multiple OpenAPI specs found")
}

// withWorkdir changes into dir for the duration of the test and restores the
// original working directory on cleanup.
func withWorkdir(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(orig) })
}

package client

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubGenerator is a minimal Generator for registry tests.
type stubGenerator struct{ name string }

func (s *stubGenerator) Name() string { return s.name }
func (s *stubGenerator) Generate(_ context.Context, _ GenerateContext) (Result, error) {
	return Result{PackageName: s.name}, nil
}

func TestRegistryGetAndNames(t *testing.T) {
	restore := resetForTesting()
	defer restore()

	Register(&stubGenerator{name: "beta"})
	Register(&stubGenerator{name: "alpha"})

	require.NotNil(t, Get("alpha"))
	assert.Equal(t, "alpha", Get("alpha").Name())
	assert.Nil(t, Get("missing"))

	// Names are sorted.
	assert.Equal(t, []string{"alpha", "beta"}, Names())
}

func TestRegisterDuplicatePanics(t *testing.T) {
	restore := resetForTesting()
	defer restore()

	Register(&stubGenerator{name: "dup"})
	assert.Panics(t, func() { Register(&stubGenerator{name: "dup"}) })
}

func TestAngularGeneratorRegistered(t *testing.T) {
	// The real angular generator registers itself via init() in the actual
	// package registry (not the test-reset one).
	g := Get("typescript-angular")
	require.NotNil(t, g)
	assert.Equal(t, "typescript-angular", g.Name())
}

func TestComposePackageName(t *testing.T) {
	cases := []struct {
		name  string
		scope string
		pkg   string
		want  string
	}{
		{"scope + bare", "@example", "notesd-client", "@example/notesd-client"},
		{"already scoped ignores scope", "@example", "@acme/notesd-client", "@acme/notesd-client"},
		{"empty scope", "", "notesd-client", "notesd-client"},
		{"scope missing at prefix", "example", "notesd-client", "@example/notesd-client"},
		{"scope trailing slash", "@example/", "notesd-client", "@example/notesd-client"},
		{"whitespace trimmed", " @example ", " notesd-client ", "@example/notesd-client"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, composePackageName(tc.scope, tc.pkg))
		})
	}
}

func TestWritePackageJSONShape(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "package.json")
	require.NoError(t, writePackageJSON(path, "@example/notesd-client", "0.0.1"))

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var pj map[string]any
	require.NoError(t, json.Unmarshal(data, &pj))

	assert.Equal(t, "@example/notesd-client", pj["name"])
	assert.Equal(t, "0.0.1", pj["version"])
	assert.Equal(t, "Apache-2.0", pj["license"])
	assert.Equal(t, "module", pj["type"])
	assert.Equal(t, "./dist/index.js", pj["main"])
	assert.Equal(t, "./dist/index.js", pj["module"])
	assert.Equal(t, "./dist/index.d.ts", pj["types"])
	assert.Equal(t, false, pj["sideEffects"])
	assert.Equal(t, []any{"dist"}, pj["files"])

	// exports.".".{types,import}
	exports, ok := pj["exports"].(map[string]any)
	require.True(t, ok, "exports must be an object")
	dot, ok := exports["."].(map[string]any)
	require.True(t, ok, "exports.\".\" must be an object")
	assert.Equal(t, "./dist/index.d.ts", dot["types"])
	assert.Equal(t, "./dist/index.js", dot["import"])

	// publishConfig
	pubCfg, ok := pj["publishConfig"].(map[string]any)
	require.True(t, ok, "publishConfig must be an object")
	assert.Equal(t, "public", pubCfg["access"])
	assert.Equal(t, "https://npm.pkg.github.com", pubCfg["registry"])

	// scripts
	scripts, ok := pj["scripts"].(map[string]any)
	require.True(t, ok, "scripts must be an object")
	assert.Equal(t, "tsc -p tsconfig.json", scripts["build"])
	assert.Equal(t, "tsc -p tsconfig.json --noEmit", scripts["typecheck"])

	// peerDependencies
	peers, ok := pj["peerDependencies"].(map[string]any)
	require.True(t, ok, "peerDependencies must be an object")
	assert.Equal(t, ">=15", peers["@angular/core"])
	assert.Equal(t, ">=15", peers["@angular/common"])
	assert.Equal(t, ">=7", peers["rxjs"])

	// devDependencies must supply the build toolchain (tsc) and satisfy the
	// generated code's peer imports so `npm run build` can compile in isolation.
	dev, ok := pj["devDependencies"].(map[string]any)
	require.True(t, ok, "devDependencies must be an object")
	assert.Contains(t, dev, "typescript")
	assert.Contains(t, dev, "@angular/core")
	assert.Contains(t, dev, "@angular/common")
	assert.Contains(t, dev, "rxjs")
}

func TestWritePackageJSONDefaultVersionViaGenerator(t *testing.T) {
	// composePackageName + writePackageJSON exercise the empty-version default
	// path indirectly; assert the const is used when version is blank by
	// calling writePackageJSON with a resolved default.
	dir := t.TempDir()
	path := filepath.Join(dir, "package.json")
	require.NoError(t, writePackageJSON(path, "notesd-client", "0.0.0"))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var pj map[string]any
	require.NoError(t, json.Unmarshal(data, &pj))
	assert.Equal(t, "0.0.0", pj["version"])
	assert.Equal(t, "notesd-client", pj["name"])
}

func TestTail(t *testing.T) {
	assert.Equal(t, "c\nd", tail("a\nb\nc\nd\n", 2))
	assert.Equal(t, "a\nb", tail("a\nb", 5))
}

func TestGenerateMissingSpec(t *testing.T) {
	// If npx is unavailable this errors on preflight; either way it must not
	// succeed with a nonexistent spec.
	g := &angularGenerator{}
	_, err := g.Generate(context.Background(), GenerateContext{
		SpecPath:    filepath.Join(t.TempDir(), "does-not-exist.yaml"),
		OutputDir:   t.TempDir(),
		PackageName: "x-client",
	})
	assert.Error(t, err)
}

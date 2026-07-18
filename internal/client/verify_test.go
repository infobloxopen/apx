package client

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeGen is a configurable Generator/Builder/ToolchainChecker test double. It
// never shells out: Generate returns a Result pointing at the output dir and
// Build returns buildErr, so Verify can be exercised without a real toolchain.
type fakeGen struct {
	name       string
	genErr     error
	buildErr   error
	toolMisss  bool // when true, ToolchainAvailable reports unavailable
	toolReason string
}

func (f *fakeGen) Name() string { return f.name }

func (f *fakeGen) Generate(_ context.Context, gc GenerateContext) (Result, error) {
	if f.genErr != nil {
		return Result{}, f.genErr
	}
	return Result{PackageDir: gc.OutputDir, PackageName: gc.PackageName}, nil
}

func (f *fakeGen) Build(_ context.Context, _ Result) error { return f.buildErr }

func (f *fakeGen) ToolchainAvailable() (bool, string) {
	if f.toolMisss {
		return false, f.toolReason
	}
	return true, ""
}

func dummySpec(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "spec.yaml")
	require.NoError(t, os.WriteFile(p, []byte("openapi: 3.0.0\n"), 0o644))
	return p
}

func resultFor(rep VerifyReport, gen string) (GeneratorResult, bool) {
	for _, r := range rep.Results {
		if r.Generator == gen {
			return r, true
		}
	}
	return GeneratorResult{}, false
}

func TestVerify_AllPass(t *testing.T) {
	restore := resetForTesting()
	defer restore()
	Register(&fakeGen{name: "go"})
	Register(&fakeGen{name: "ts"})

	rep, err := Verify(context.Background(), VerifyOptions{
		SpecPath:   dummySpec(t),
		Generators: []string{"go", "ts"},
		WorkDir:    t.TempDir(),
	})
	require.NoError(t, err)
	assert.False(t, rep.Failed())
	assert.Equal(t, 2, rep.Ran())
	require.Len(t, rep.Results, 2)
	for _, r := range rep.Results {
		assert.True(t, r.OK, "%s should pass", r.Generator)
	}
}

func TestVerify_BuildFailureGates(t *testing.T) {
	restore := resetForTesting()
	defer restore()
	Register(&fakeGen{name: "go", buildErr: errors.New("Limit redeclared")})

	rep, err := Verify(context.Background(), VerifyOptions{
		SpecPath:   dummySpec(t),
		Generators: []string{"go"},
		WorkDir:    t.TempDir(),
	})
	require.NoError(t, err)
	assert.True(t, rep.Failed())
	res, ok := resultFor(rep, "go")
	require.True(t, ok)
	assert.False(t, res.OK)
	require.Error(t, res.Err)
	assert.Contains(t, res.Err.Error(), "build")
	assert.Contains(t, res.Err.Error(), "Limit redeclared")
}

func TestVerify_GenerateFailureGates(t *testing.T) {
	restore := resetForTesting()
	defer restore()
	Register(&fakeGen{name: "go", genErr: errors.New("oapi-codegen boom")})

	rep, err := Verify(context.Background(), VerifyOptions{
		SpecPath:   dummySpec(t),
		Generators: []string{"go"},
		WorkDir:    t.TempDir(),
	})
	require.NoError(t, err)
	assert.True(t, rep.Failed())
	res, _ := resultFor(rep, "go")
	assert.Contains(t, res.Err.Error(), "generate")
}

func TestVerify_SkipsWhenToolchainMissing(t *testing.T) {
	restore := resetForTesting()
	defer restore()
	Register(&fakeGen{name: "ts", toolMisss: true, toolReason: "npx not found on PATH"})
	Register(&fakeGen{name: "go"})

	rep, err := Verify(context.Background(), VerifyOptions{
		SpecPath:   dummySpec(t),
		Generators: []string{"ts", "go"},
		WorkDir:    t.TempDir(),
	})
	require.NoError(t, err)
	assert.False(t, rep.Failed(), "a skip is not a gate failure")
	assert.Equal(t, 1, rep.Ran(), "only go actually ran")

	ts, _ := resultFor(rep, "ts")
	assert.True(t, ts.Skipped)
	assert.Equal(t, "npx not found on PATH", ts.Reason)
	assert.False(t, ts.OK)

	goRes, _ := resultFor(rep, "go")
	assert.True(t, goRes.OK)
}

func TestVerify_UnknownGeneratorErrors(t *testing.T) {
	restore := resetForTesting()
	defer restore()
	Register(&fakeGen{name: "go"})

	_, err := Verify(context.Background(), VerifyOptions{
		SpecPath:   dummySpec(t),
		Generators: []string{"bogus"},
		WorkDir:    t.TempDir(),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown client generator")
}

func TestVerify_MissingSpecErrors(t *testing.T) {
	restore := resetForTesting()
	defer restore()
	Register(&fakeGen{name: "go"})

	_, err := Verify(context.Background(), VerifyOptions{
		SpecPath:   filepath.Join(t.TempDir(), "does-not-exist.yaml"),
		Generators: []string{"go"},
		WorkDir:    t.TempDir(),
	})
	require.Error(t, err)
}

func TestVerify_EmptySpecErrors(t *testing.T) {
	restore := resetForTesting()
	defer restore()
	_, err := Verify(context.Background(), VerifyOptions{Generators: []string{"go"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "spec path is required")
}

func TestVerify_DefaultGeneratorMatrix(t *testing.T) {
	restore := resetForTesting()
	defer restore()
	// Register fakes under the default matrix names so the empty-Generators path
	// resolves without a real toolchain.
	Register(&fakeGen{name: "go"})
	Register(&fakeGen{name: "typescript-angular"})

	rep, err := Verify(context.Background(), VerifyOptions{
		SpecPath: dummySpec(t),
		WorkDir:  t.TempDir(),
	})
	require.NoError(t, err)
	require.Len(t, rep.Results, 2)
	assert.Equal(t, []string{"go", "typescript-angular"}, DefaultVerifyGenerators)
}

// TestVerify_RealGeneratorsAreToolchainCheckers guards that the shipped go and
// typescript-angular generators implement ToolchainChecker, so verification can
// skip them gracefully when their toolchains are absent.
func TestVerify_RealGeneratorsAreToolchainCheckers(t *testing.T) {
	for _, name := range DefaultVerifyGenerators {
		g := Get(name)
		require.NotNil(t, g, "generator %q must be registered", name)
		_, ok := g.(ToolchainChecker)
		assert.True(t, ok, "generator %q must implement ToolchainChecker", name)
	}
}

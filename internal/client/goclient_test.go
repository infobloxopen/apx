package client

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/infobloxopen/apx/internal/publisher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGoGeneratorRegistered confirms the go generator self-registers in the real
// registry (via init()) and is listed for `apx client generate`.
func TestGoGeneratorRegistered(t *testing.T) {
	g := Get("go")
	require.NotNil(t, g)
	assert.Equal(t, "go", g.Name())
	assert.Contains(t, Names(), "go")
}

// TestBuilderPublisherWiring is the backward-compat guard (FR-7/AC-5/FM-4): the
// go generator implements Builder+Publisher (so the command layer routes to it),
// while typescript-angular implements NEITHER (so it keeps the npm build/publish
// path unchanged).
func TestBuilderPublisherWiring(t *testing.T) {
	goGen := Get("go")
	require.NotNil(t, goGen)
	_, isBuilder := goGen.(Builder)
	_, isPublisher := goGen.(Publisher)
	assert.True(t, isBuilder, "go generator must implement Builder")
	assert.True(t, isPublisher, "go generator must implement Publisher")

	ng := Get("typescript-angular")
	require.NotNil(t, ng)
	_, ngBuilder := ng.(Builder)
	_, ngPublisher := ng.(Publisher)
	assert.False(t, ngBuilder, "typescript-angular must NOT implement Builder (keeps npm build path)")
	assert.False(t, ngPublisher, "typescript-angular must NOT implement Publisher (keeps npm publish path)")
}

func TestGoPackageIdent(t *testing.T) {
	cases := map[string]string{
		"github.com/example/toy-client": "toyclient",
		"toy-client":                    "toyclient",
		"example.com/Foo_Bar":           "foobar",
		"":                              "client",
		"github.com/x/9lives":           "client9lives",
	}
	for in, want := range cases {
		assert.Equal(t, want, goPackageIdent(in), "goPackageIdent(%q)", in)
	}
}

func TestGoModContent(t *testing.T) {
	got := goModContent("github.com/example/toy-client")
	assert.Contains(t, got, "module github.com/example/toy-client")
	assert.Contains(t, got, "go 1.")
}

// TestGoGenerate_EmptyModulePath fails loud when no module path is given (FR-3).
func TestGoGenerate_EmptyModulePath(t *testing.T) {
	g := &goGenerator{}
	_, err := g.Generate(context.Background(), GenerateContext{
		SpecPath:    goClientFixture(t),
		OutputDir:   t.TempDir(),
		PackageName: "", // no module path
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "module path")
}

// TestGoGenerate_MissingSpec fails loud on an absent spec (FM-2).
func TestGoGenerate_MissingSpec(t *testing.T) {
	g := &goGenerator{}
	_, err := g.Generate(context.Background(), GenerateContext{
		SpecPath:    filepath.Join(t.TempDir(), "does-not-exist.yaml"),
		OutputDir:   t.TempDir(),
		PackageName: "github.com/example/toy-client",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestGoPublish_DryRunRecordsArtifact proves the go Publisher records a go-module
// artifact honoring DryRun, with no write token needed (AC-4). Pure — no network.
func TestGoPublish_DryRunRecordsArtifact(t *testing.T) {
	recPath := filepath.Join(t.TempDir(), "release-record.yaml")
	require.NoError(t, publisher.WriteReleaseRecord(&publisher.ReleaseRecord{
		SchemaVersion: "1",
		Kind:          "release-record",
		APIID:         "toy",
	}, recPath))

	g := &goGenerator{}
	err := g.Publish(context.Background(), Result{
		PackageName: "github.com/example/toy-client",
		PackageDir:  t.TempDir(),
	}, PublishOptions{DryRun: true, Version: "1.2.3", RecordPath: recPath})
	require.NoError(t, err)

	rec, err := publisher.ReadReleaseRecord(recPath)
	require.NoError(t, err)
	require.Len(t, rec.Artifacts, 1)
	a := rec.Artifacts[0]
	assert.Equal(t, "go-module", a.Type)
	assert.Equal(t, "github.com/example/toy-client", a.Name)
	assert.Equal(t, "1.2.3", a.Version)
	assert.Equal(t, "dry-run", a.Status)
}

// TestGoGenerator_E2E generates a Go client from the enriched toy OpenAPI (the
// devedge-sdk 044 golden), compile-verifies it via the Builder, and asserts the
// enriched contract projected into Go: a typed client method, an enum type
// (allowed_values), a required field (explicit field_behavior=REQUIRED), and a
// not_null-but-optional field (proving not_null↛REQUIRED). Runs the real
// oapi-codegen + go build, so it needs network for the first tool fetch; skipped
// under -short and skipped (not failed) if module download is unavailable.
func TestGoGenerator_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network/tool e2e in -short mode")
	}
	out := t.TempDir()
	g := &goGenerator{}
	res, err := g.Generate(context.Background(), GenerateContext{
		SpecPath:    goClientFixture(t),
		OutputDir:   out,
		PackageName: "github.com/example/toy-client",
	})
	if err != nil && looksLikeNetworkErr(err) {
		t.Skipf("skipping e2e: tool/module fetch unavailable: %v", err)
	}
	require.NoError(t, err)

	if err := g.Build(context.Background(), res); err != nil {
		if looksLikeNetworkErr(err) {
			t.Skipf("skipping e2e build: module fetch unavailable: %v", err)
		}
		t.Fatalf("generated client failed to build: %v", err)
	}

	src, err := os.ReadFile(filepath.Join(out, "toyclient.gen.go"))
	require.NoError(t, err)
	code := string(src)
	// Unknown x-aip-* extensions were ignored (it compiled). Now the projections:
	assert.Contains(t, code, "type V1WidgetCategory string", "enum type from allowed_values (AC-4)")
	assert.Regexp(t, `DisplayName\s+string`, code, "REQUIRED field is non-pointer (AC-2)")
	assert.Regexp(t, `Sku\s+\*string`, code, "not_null-but-not-required field stays optional pointer (FM-4)")
	assert.Contains(t, code, "type Client struct", "typed client generated (AC-2)")
}

// goClientFixture returns the path to the enriched toy OpenAPI fixture (copied
// from the devedge-sdk 044 golden), relative to this package dir.
func goClientFixture(t *testing.T) string {
	t.Helper()
	p := filepath.Join("..", "..", "testdata", "goclient", "toy.openapi.yaml")
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("fixture missing at %s: %v", p, err)
	}
	return p
}

func looksLikeNetworkErr(err error) bool {
	s := strings.ToLower(err.Error())
	for _, hint := range []string{"dial tcp", "no such host", "network is unreachable", "timeout", "connection refused", "proxy.golang.org", "could not resolve", "i/o timeout"} {
		if strings.Contains(s, hint) {
			return true
		}
	}
	return false
}

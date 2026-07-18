package client

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVerify_E2E_RejectsBrokenSpec is the regression guard: a spec that passes
// lint/breaking but whose generated Go client does not compile MUST fail
// verification. It runs the real oapi-codegen + go build via the go generator,
// so it is skipped under -short and skipped (not failed) when tool/module fetch
// is unavailable — matching TestGoGenerator_E2E.
func TestVerify_E2E_RejectsBrokenSpec(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network/tool e2e in -short mode")
	}
	report, err := Verify(context.Background(), VerifyOptions{
		SpecPath:   brokenClientFixture(t),
		Generators: []string{"go"},
		WorkDir:    t.TempDir(),
	})
	// Verify itself succeeds — the per-generator failure is carried in the report.
	require.NoError(t, err)
	require.Len(t, report.Results, 1)

	res := report.Results[0]
	if res.Err != nil && looksLikeNetworkErr(res.Err) {
		t.Skipf("skipping e2e: tool/module fetch unavailable: %v", res.Err)
	}
	assert.True(t, report.Failed(), "broken spec must fail client verification")
	assert.False(t, res.OK)
	require.Error(t, res.Err)
	// The client is generated but does not compile: the failure is at build.
	assert.Contains(t, res.Err.Error(), "build")
}

// TestVerify_E2E_AcceptsGoodSpec proves the gate passes a spec whose generated
// Go client compiles cleanly (the enriched toy fixture).
func TestVerify_E2E_AcceptsGoodSpec(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network/tool e2e in -short mode")
	}
	report, err := Verify(context.Background(), VerifyOptions{
		SpecPath:    goClientFixture(t),
		Generators:  []string{"go"},
		PackageName: "github.com/example/toy-client",
		WorkDir:     t.TempDir(),
	})
	require.NoError(t, err)
	require.Len(t, report.Results, 1)

	res := report.Results[0]
	if res.Err != nil && looksLikeNetworkErr(res.Err) {
		t.Skipf("skipping e2e: tool/module fetch unavailable: %v", res.Err)
	}
	assert.False(t, report.Failed(), "a good spec must pass verification")
	assert.True(t, res.OK)
}

// brokenClientFixture returns the path to the CICD regression spec that produces
// a non-compiling Go client (patterns A and B).
func brokenClientFixture(t *testing.T) string {
	t.Helper()
	p := filepath.Join("..", "..", "testdata", "goclient", "collision.openapi.yaml")
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("fixture missing at %s: %v", p, err)
	}
	return p
}

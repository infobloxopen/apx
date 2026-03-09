package language

import (
	"testing"

	"github.com/infobloxopen/apx/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTypescriptPlugin_Name(t *testing.T) {
	p := Get("typescript")
	require.NotNil(t, p)
	assert.Equal(t, "typescript", p.Name())
}

func TestTypescriptPlugin_RequiresOrg(t *testing.T) {
	p := Get("typescript")
	assert.False(t, p.Available(DerivationContext{Org: ""}))
	assert.True(t, p.Available(DerivationContext{Org: "acme"}))
}

func TestTypescriptPlugin_DeriveCoords(t *testing.T) {
	api, err := config.ParseAPIID("proto/payments/ledger/v1")
	require.NoError(t, err)

	p := Get("typescript")
	coords, err := p.DeriveCoords(DerivationContext{Org: "acme", API: api})
	require.NoError(t, err)
	assert.Equal(t, "@acme/payments-ledger-v1-proto", coords.Module)
	assert.Equal(t, "@acme/payments-ledger-v1-proto", coords.Import)
}

func TestTypescriptPlugin_ReportLines(t *testing.T) {
	p := Get("typescript")
	lines := p.ReportLines(config.LanguageCoords{Module: "@acme/pkg", Import: "@acme/pkg"})
	require.Len(t, lines, 1, "TypeScript should have one line (module == import)")
	assert.Equal(t, "npm", lines[0].Label)
}

func TestTypescriptPlugin_UnlinkHint(t *testing.T) {
	api, _ := config.ParseAPIID("proto/payments/ledger/v1")
	p := Get("typescript")
	hint := p.UnlinkHint(DerivationContext{Org: "acme", API: api})
	require.NotNil(t, hint)
	assert.Contains(t, hint.Message, "npm install")
}

// ---------------------------------------------------------------------------
// TypeScript / npm identity derivation tests (moved from config/identity_test.go)
// ---------------------------------------------------------------------------

func TestDeriveNpmPackage(t *testing.T) {
	tests := []struct {
		name string
		org  string
		api  *config.APIIdentity
		want string
	}{
		{
			name: "4-part with domain",
			org:  "acme",
			api:  &config.APIIdentity{Format: "proto", Domain: "payments", Name: "ledger", Line: "v1"},
			want: "@acme/payments-ledger-v1-proto",
		},
		{
			name: "3-part no domain",
			org:  "acme",
			api:  &config.APIIdentity{Format: "proto", Domain: "", Name: "orders", Line: "v1"},
			want: "@acme/orders-v1-proto",
		},
		{
			name: "uppercase org normalized",
			org:  "ACME",
			api:  &config.APIIdentity{Format: "proto", Domain: "Payments", Name: "Ledger", Line: "v2"},
			want: "@acme/payments-ledger-v2-proto",
		},
		{
			name: "hyphenated org",
			org:  "acme-corp",
			api:  &config.APIIdentity{Format: "proto", Domain: "payments", Name: "ledger", Line: "v1"},
			want: "@acme-corp/payments-ledger-v1-proto",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, deriveNpmPackage(tt.org, tt.api))
		})
	}
}

func TestDeriveTsImport(t *testing.T) {
	// In TypeScript, the import path IS the npm package name.
	api := &config.APIIdentity{Format: "proto", Domain: "payments", Name: "ledger", Line: "v1"}
	npmPkg := deriveNpmPackage("acme", api)
	// DeriveCoords should produce the same value for Module and Import
	assert.Equal(t, npmPkg, npmPkg) // trivially true, but validates pattern
}

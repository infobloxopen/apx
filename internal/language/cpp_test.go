package language

import (
	"testing"

	"github.com/infobloxopen/apx/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCppPlugin_Name(t *testing.T) {
	p := Get("cpp")
	require.NotNil(t, p)
	assert.Equal(t, "cpp", p.Name())
}

func TestCppPlugin_Tier(t *testing.T) {
	p := Get("cpp")
	assert.Equal(t, 2, p.Tier())
}

func TestCppPlugin_RequiresOrg(t *testing.T) {
	p := Get("cpp")
	assert.False(t, p.Available(DerivationContext{Org: ""}))
	assert.True(t, p.Available(DerivationContext{Org: "acme"}))
}

func TestCppPlugin_DeriveCoords(t *testing.T) {
	api, err := config.ParseAPIID("proto/payments/ledger/v1")
	require.NoError(t, err)

	p := Get("cpp")
	coords, err := p.DeriveCoords(DerivationContext{Org: "acme", API: api})
	require.NoError(t, err)
	assert.Equal(t, "acme-payments-ledger-v1-proto", coords.Module)
	assert.Equal(t, "acme::payments::ledger::v1", coords.Import)
}

func TestCppPlugin_DeriveCoords_3Part(t *testing.T) {
	api, err := config.ParseAPIID("proto/orders/v1")
	require.NoError(t, err)

	p := Get("cpp")
	coords, err := p.DeriveCoords(DerivationContext{Org: "acme", API: api})
	require.NoError(t, err)
	assert.Equal(t, "acme-orders-v1-proto", coords.Module)
	assert.Equal(t, "acme::orders::v1", coords.Import)
}

func TestCppPlugin_ReportLines(t *testing.T) {
	p := Get("cpp")
	lines := p.ReportLines(config.LanguageCoords{Module: "ref", Import: "ns"})
	require.Len(t, lines, 2)
	assert.Equal(t, "Conan", lines[0].Label)
	assert.Equal(t, "C++ ns", lines[1].Label)
}

func TestCppPlugin_UnlinkHint(t *testing.T) {
	api, _ := config.ParseAPIID("proto/payments/ledger/v1")
	p := Get("cpp")
	hint := p.UnlinkHint(DerivationContext{Org: "acme", API: api})
	require.NotNil(t, hint)
	assert.Contains(t, hint.Message, "conanfile")
}

// ---------------------------------------------------------------------------
// C++ / Conan identity derivation tests
// ---------------------------------------------------------------------------

func TestDeriveCppConanRef(t *testing.T) {
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
			want: "acme-payments-ledger-v1-proto",
		},
		{
			name: "3-part no domain",
			org:  "acme",
			api:  &config.APIIdentity{Format: "proto", Domain: "", Name: "orders", Line: "v1"},
			want: "acme-orders-v1-proto",
		},
		{
			name: "uppercase org normalized",
			org:  "ACME",
			api:  &config.APIIdentity{Format: "proto", Domain: "Payments", Name: "Ledger", Line: "v2"},
			want: "acme-payments-ledger-v2-proto",
		},
		{
			name: "openapi format",
			org:  "acme",
			api:  &config.APIIdentity{Format: "openapi", Domain: "billing", Name: "invoices", Line: "v1"},
			want: "acme-billing-invoices-v1-proto",
		},
		{
			name: "hyphenated org kept in conan ref",
			org:  "acme-corp",
			api:  &config.APIIdentity{Format: "proto", Domain: "payments", Name: "ledger", Line: "v1"},
			want: "acme-corp-payments-ledger-v1-proto",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, deriveCppConanRef(tt.org, tt.api))
		})
	}
}

func TestDeriveCppNamespace(t *testing.T) {
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
			want: "acme::payments::ledger::v1",
		},
		{
			name: "3-part no domain",
			org:  "acme",
			api:  &config.APIIdentity{Format: "proto", Domain: "", Name: "orders", Line: "v1"},
			want: "acme::orders::v1",
		},
		{
			name: "uppercase org normalized",
			org:  "ACME",
			api:  &config.APIIdentity{Format: "proto", Domain: "Payments", Name: "Ledger", Line: "v2"},
			want: "acme::payments::ledger::v2",
		},
		{
			name: "hyphenated org normalized to underscores",
			org:  "acme-corp",
			api:  &config.APIIdentity{Format: "proto", Domain: "payments", Name: "ledger", Line: "v1"},
			want: "acme_corp::payments::ledger::v1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, deriveCppNamespace(tt.org, tt.api))
		})
	}
}

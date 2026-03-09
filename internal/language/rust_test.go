package language

import (
	"testing"

	"github.com/infobloxopen/apx/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRustPlugin_Name(t *testing.T) {
	p := Get("rust")
	require.NotNil(t, p)
	assert.Equal(t, "rust", p.Name())
}

func TestRustPlugin_Tier(t *testing.T) {
	p := Get("rust")
	assert.Equal(t, 2, p.Tier())
}

func TestRustPlugin_RequiresOrg(t *testing.T) {
	p := Get("rust")
	assert.False(t, p.Available(DerivationContext{Org: ""}))
	assert.True(t, p.Available(DerivationContext{Org: "acme"}))
}

func TestRustPlugin_DeriveCoords(t *testing.T) {
	api, err := config.ParseAPIID("proto/payments/ledger/v1")
	require.NoError(t, err)

	p := Get("rust")
	coords, err := p.DeriveCoords(DerivationContext{Org: "acme", API: api})
	require.NoError(t, err)
	assert.Equal(t, "acme-payments-ledger-v1-proto", coords.Module)
	assert.Equal(t, "acme_payments::ledger::v1", coords.Import)
}

func TestRustPlugin_DeriveCoords_3Part(t *testing.T) {
	api, err := config.ParseAPIID("proto/orders/v1")
	require.NoError(t, err)

	p := Get("rust")
	coords, err := p.DeriveCoords(DerivationContext{Org: "acme", API: api})
	require.NoError(t, err)
	assert.Equal(t, "acme-orders-v1-proto", coords.Module)
	assert.Equal(t, "acme_orders::v1", coords.Import)
}

func TestRustPlugin_ReportLines(t *testing.T) {
	p := Get("rust")
	lines := p.ReportLines(config.LanguageCoords{Module: "crate", Import: "mod"})
	require.Len(t, lines, 2)
	assert.Equal(t, "Crate", lines[0].Label)
	assert.Equal(t, "Rust mod", lines[1].Label)
}

func TestRustPlugin_UnlinkHint(t *testing.T) {
	api, _ := config.ParseAPIID("proto/payments/ledger/v1")
	p := Get("rust")
	hint := p.UnlinkHint(DerivationContext{Org: "acme", API: api})
	require.NotNil(t, hint)
	assert.Contains(t, hint.Message, "Cargo.toml")
}

// ---------------------------------------------------------------------------
// Rust / Cargo identity derivation tests
// ---------------------------------------------------------------------------

func TestDeriveRustCrate(t *testing.T) {
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, deriveRustCrate(tt.org, tt.api))
		})
	}
}

func TestDeriveRustModule(t *testing.T) {
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
			want: "acme_payments::ledger::v1",
		},
		{
			name: "3-part no domain",
			org:  "acme",
			api:  &config.APIIdentity{Format: "proto", Domain: "", Name: "orders", Line: "v1"},
			want: "acme_orders::v1",
		},
		{
			name: "uppercase org normalized",
			org:  "ACME",
			api:  &config.APIIdentity{Format: "proto", Domain: "Payments", Name: "Ledger", Line: "v2"},
			want: "acme_payments::ledger::v2",
		},
		{
			name: "v0 line",
			org:  "myorg",
			api:  &config.APIIdentity{Format: "proto", Domain: "events", Name: "click", Line: "v0"},
			want: "myorg_events::click::v0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, deriveRustModule(tt.org, tt.api))
		})
	}
}

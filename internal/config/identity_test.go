package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAPIID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *APIIdentity
		wantErr string
	}{
		{
			name:  "valid proto v1",
			input: "proto/payments/ledger/v1",
			want: &APIIdentity{
				ID: "proto/payments/ledger/v1", Format: "proto",
				Domain: "payments", Name: "ledger", Line: "v1",
			},
		},
		{
			name:  "valid openapi v2",
			input: "openapi/billing/invoices/v2",
			want: &APIIdentity{
				ID: "openapi/billing/invoices/v2", Format: "openapi",
				Domain: "billing", Name: "invoices", Line: "v2",
			},
		},
		{
			name:  "valid avro v3",
			input: "avro/events/click/v3",
			want: &APIIdentity{
				ID: "avro/events/click/v3", Format: "avro",
				Domain: "events", Name: "click", Line: "v3",
			},
		},
		{
			name:    "too few parts (no line)",
			input:   "proto/payments/ledger",
			wantErr: "invalid API line", // parsed as 3-part; "ledger" is not a valid v<N> line
		},
		{
			name:    "too many parts",
			input:   "proto/payments/ledger/v1/extra",
			wantErr: "expected format/<name>/<line> or format/<domain>/<name>/<line>",
		},
		{
			name:  "valid 3-part (no domain)",
			input: "proto/orders/v1",
			want: &APIIdentity{
				ID: "proto/orders/v1", Format: "proto",
				Domain: "", Name: "orders", Line: "v1",
			},
		},
		{
			name:    "invalid format",
			input:   "graphql/payments/ledger/v1",
			wantErr: "invalid API format",
		},
		{
			name:    "invalid line missing v prefix",
			input:   "proto/payments/ledger/1",
			wantErr: "invalid API line",
		},
		{
			name:  "valid v0 line",
			input: "proto/payments/ledger/v0",
			want: &APIIdentity{
				ID: "proto/payments/ledger/v0", Format: "proto",
				Domain: "payments", Name: "ledger", Line: "v0",
			},
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: "expected format/<name>/<line> or format/<domain>/<name>/<line>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseAPIID(tt.input)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want.ID, got.ID)
			assert.Equal(t, tt.want.Format, got.Format)
			assert.Equal(t, tt.want.Domain, got.Domain)
			assert.Equal(t, tt.want.Name, got.Name)
			assert.Equal(t, tt.want.Line, got.Line)
		})
	}
}

func TestLineMajor(t *testing.T) {
	tests := []struct {
		line    string
		want    int
		wantErr bool
	}{
		{"v0", 0, false},
		{"v1", 1, false},
		{"v2", 2, false},
		{"v10", 10, false},
		{"1", 0, true},
		{"vx", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			got, err := LineMajor(tt.line)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDeriveTag(t *testing.T) {
	tests := []struct {
		apiID   string
		version string
		want    string
	}{
		{"proto/payments/ledger/v1", "v1.0.0-alpha.1", "proto/payments/ledger/v1/v1.0.0-alpha.1"},
		{"proto/payments/ledger/v1", "1.0.0-beta.1", "proto/payments/ledger/v1/v1.0.0-beta.1"},
		{"proto/payments/ledger/v2", "v2.0.0", "proto/payments/ledger/v2/v2.0.0"},
	}
	for _, tt := range tests {
		t.Run(tt.apiID+"@"+tt.version, func(t *testing.T) {
			assert.Equal(t, tt.want, DeriveTag(tt.apiID, tt.version))
		})
	}
}

func TestBuildIdentityBlock(t *testing.T) {
	api, source, release, err := BuildIdentityBlock(
		"proto/payments/ledger/v1",
		"github.com/acme/apis",
		"beta",
		"v1.0.0-beta.1",
	)
	require.NoError(t, err)

	assert.Equal(t, "proto/payments/ledger/v1", api.ID)
	assert.Equal(t, "beta", api.Lifecycle)
	assert.Equal(t, "github.com/acme/apis", source.Repo)
	assert.Equal(t, "proto/payments/ledger/v1", source.Path)
	assert.Equal(t, "v1.0.0-beta.1", release.Current)
}

func TestBuildIdentityBlockV2(t *testing.T) {
	api, source, release, err := BuildIdentityBlock(
		"proto/payments/ledger/v2",
		"github.com/acme/apis",
		"experimental",
		"v2.0.0-alpha.1",
	)
	require.NoError(t, err)

	assert.Equal(t, "proto/payments/ledger/v2", api.ID)
	assert.Equal(t, "experimental", api.Lifecycle)
	assert.Equal(t, "proto/payments/ledger/v2", source.Path)
	assert.Equal(t, "v2.0.0-alpha.1", release.Current)
}

func TestBuildIdentityBlockNoRelease(t *testing.T) {
	api, source, release, err := BuildIdentityBlock(
		"openapi/billing/invoices/v1",
		"github.com/acme/apis",
		"",
		"",
	)
	require.NoError(t, err)

	assert.Equal(t, "openapi/billing/invoices/v1", api.ID)
	assert.Equal(t, "", api.Lifecycle)
	assert.Equal(t, "github.com/acme/apis", source.Repo)
	assert.Nil(t, release)
}

func TestValidateLifecycle(t *testing.T) {
	for _, valid := range []string{"experimental", "preview", "beta", "stable", "deprecated", "sunset"} {
		assert.NoError(t, ValidateLifecycle(valid))
	}
	assert.Error(t, ValidateLifecycle("alpha"))
	assert.Error(t, ValidateLifecycle(""))
	assert.Error(t, ValidateLifecycle("ga"))
}

func TestBuildIdentityBlockNoLifecycle(t *testing.T) {
	api, _, _, err := BuildIdentityBlock(
		"proto/payments/ledger/v1",
		"github.com/acme/apis",
		"",
		"",
	)
	require.NoError(t, err)
	assert.Equal(t, "", api.Lifecycle)
}

func TestValidateGoPackage(t *testing.T) {
	tests := []struct {
		name           string
		goPackage      string
		expectedImport string
		wantErr        string
	}{
		{
			name:           "exact match",
			goPackage:      "github.com/acme/apis/proto/payments/ledger/v1",
			expectedImport: "github.com/acme/apis/proto/payments/ledger/v1",
		},
		{
			name:           "match with alias stripped",
			goPackage:      "github.com/acme/apis/proto/payments/ledger/v1;ledgerpb",
			expectedImport: "github.com/acme/apis/proto/payments/ledger/v1",
		},
		{
			name:           "mismatch",
			goPackage:      "github.com/wrong/path/v1",
			expectedImport: "github.com/acme/apis/proto/payments/ledger/v1",
			wantErr:        "go_package mismatch",
		},
		{
			name:           "mismatch with alias",
			goPackage:      "github.com/wrong/path/v1;pb",
			expectedImport: "github.com/acme/apis/proto/payments/ledger/v1",
			wantErr:        "go_package mismatch",
		},
		{
			name:           "empty go_package skipped",
			goPackage:      "",
			expectedImport: "github.com/acme/apis/proto/payments/ledger/v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGoPackage(tt.goPackage, tt.expectedImport)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestDeriveGoModDir(t *testing.T) {
	tests := []struct {
		name string
		api  *APIIdentity
		want string
	}{
		{
			name: "v0 no version suffix",
			api:  &APIIdentity{Format: "proto", Domain: "payments", Name: "ledger", Line: "v0"},
			want: "proto/payments/ledger",
		},
		{
			name: "v1 no version suffix",
			api:  &APIIdentity{Format: "proto", Domain: "payments", Name: "ledger", Line: "v1"},
			want: "proto/payments/ledger",
		},
		{
			name: "v2 includes version suffix",
			api:  &APIIdentity{Format: "proto", Domain: "payments", Name: "ledger", Line: "v2"},
			want: "proto/payments/ledger/v2",
		},
		{
			name: "v3 includes version suffix",
			api:  &APIIdentity{Format: "openapi", Domain: "billing", Name: "invoices", Line: "v3"},
			want: "openapi/billing/invoices/v3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeriveGoModDir(tt.api)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// v0 line support
// ---------------------------------------------------------------------------

func TestIsV0Line(t *testing.T) {
	assert.True(t, IsV0Line("v0"))
	assert.False(t, IsV0Line("v1"))
	assert.False(t, IsV0Line("v2"))
	assert.False(t, IsV0Line("invalid"))
}

func TestBuildIdentityBlockV0(t *testing.T) {
	api, source, release, err := BuildIdentityBlock(
		"proto/payments/ledger/v0",
		"github.com/acme/apis",
		"experimental",
		"v0.1.0-alpha.1",
	)
	require.NoError(t, err)

	assert.Equal(t, "proto/payments/ledger/v0", api.ID)
	assert.Equal(t, "v0", api.Line)
	assert.Equal(t, "experimental", api.Lifecycle)
	assert.Equal(t, "github.com/acme/apis", source.Repo)
	assert.Equal(t, "v0.1.0-alpha.1", release.Current)
}

// ---------------------------------------------------------------------------
// Import-root decoupling (APX-112)
// ---------------------------------------------------------------------------

func TestEffectiveGoRoot(t *testing.T) {
	tests := []struct {
		name       string
		sourceRepo string
		importRoot string
		want       string
	}{
		{
			name:       "import root empty falls back to source repo",
			sourceRepo: "github.com/acme/apis",
			importRoot: "",
			want:       "github.com/acme/apis",
		},
		{
			name:       "import root set overrides source repo",
			sourceRepo: "github.com/acme/apis",
			importRoot: "go.acme.dev/apis",
			want:       "go.acme.dev/apis",
		},
		{
			name:       "import root with different host",
			sourceRepo: "github.com/acme/apis",
			importRoot: "buf.build/gen/go/acme/apis",
			want:       "buf.build/gen/go/acme/apis",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, EffectiveGoRoot(tt.sourceRepo, tt.importRoot))
		})
	}
}

// Note: Language coordinate derivation tests have moved to internal/language/
// package where the plugin system lives. Each plugin owns its own derivation
// logic and tests.

package config

import (
	"strings"
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
			name:    "too few parts",
			input:   "proto/payments/ledger",
			wantErr: "expected format/<domain>/<name>/<line>",
		},
		{
			name:    "too many parts",
			input:   "proto/payments/ledger/v1/extra",
			wantErr: "expected format/<domain>/<name>/<line>",
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
			wantErr: "expected format/<domain>/<name>/<line>",
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

func TestDeriveGoModule(t *testing.T) {
	tests := []struct {
		name       string
		sourceRepo string
		api        *APIIdentity
		want       string
	}{
		{
			name:       "v1 module has no version suffix",
			sourceRepo: "github.com/acme/apis",
			api:        &APIIdentity{Format: "proto", Domain: "payments", Name: "ledger", Line: "v1"},
			want:       "github.com/acme/apis/proto/payments/ledger",
		},
		{
			name:       "v2 module has version suffix",
			sourceRepo: "github.com/acme/apis",
			api:        &APIIdentity{Format: "proto", Domain: "payments", Name: "ledger", Line: "v2"},
			want:       "github.com/acme/apis/proto/payments/ledger/v2",
		},
		{
			name:       "v3 module has version suffix",
			sourceRepo: "github.com/acme/apis",
			api:        &APIIdentity{Format: "openapi", Domain: "billing", Name: "invoices", Line: "v3"},
			want:       "github.com/acme/apis/openapi/billing/invoices/v3",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DeriveGoModule(tt.sourceRepo, tt.api)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDeriveGoImport(t *testing.T) {
	tests := []struct {
		name       string
		sourceRepo string
		api        *APIIdentity
		want       string
	}{
		{
			name:       "v1 import includes v1",
			sourceRepo: "github.com/acme/apis",
			api:        &APIIdentity{Format: "proto", Domain: "payments", Name: "ledger", Line: "v1"},
			want:       "github.com/acme/apis/proto/payments/ledger/v1",
		},
		{
			name:       "v2 import includes v2",
			sourceRepo: "github.com/acme/apis",
			api:        &APIIdentity{Format: "proto", Domain: "payments", Name: "ledger", Line: "v2"},
			want:       "github.com/acme/apis/proto/payments/ledger/v2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DeriveGoImport(tt.sourceRepo, tt.api)
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

func TestDeriveLanguageCoords(t *testing.T) {
	api := &APIIdentity{Format: "proto", Domain: "payments", Name: "ledger", Line: "v1"}
	coords, err := DeriveLanguageCoords("github.com/acme/apis", api)
	require.NoError(t, err)

	goCoords, ok := coords["go"]
	require.True(t, ok)
	assert.Equal(t, "github.com/acme/apis/proto/payments/ledger", goCoords.Module)
	assert.Equal(t, "github.com/acme/apis/proto/payments/ledger/v1", goCoords.Import)
}

func TestBuildIdentityBlock(t *testing.T) {
	api, source, release, langs, err := BuildIdentityBlock(
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
	assert.Equal(t, "github.com/acme/apis/proto/payments/ledger", langs["go"].Module)
	assert.Equal(t, "github.com/acme/apis/proto/payments/ledger/v1", langs["go"].Import)
}

func TestBuildIdentityBlockV2(t *testing.T) {
	api, source, release, langs, err := BuildIdentityBlock(
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
	// v2 module path includes /v2 suffix
	assert.Equal(t, "github.com/acme/apis/proto/payments/ledger/v2", langs["go"].Module)
	assert.Equal(t, "github.com/acme/apis/proto/payments/ledger/v2", langs["go"].Import)
}

func TestBuildIdentityBlockNoRelease(t *testing.T) {
	api, source, release, langs, err := BuildIdentityBlock(
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
	assert.NotNil(t, langs["go"])
}

func TestFormatIdentityReport(t *testing.T) {
	api, source, release, langs, err := BuildIdentityBlock(
		"proto/payments/ledger/v1",
		"github.com/acme/apis",
		"beta",
		"v1.0.0-beta.1",
	)
	require.NoError(t, err)

	report := FormatIdentityReport(api, source, release, langs)
	assert.Contains(t, report, "API:        proto/payments/ledger/v1")
	assert.Contains(t, report, "Format:     proto")
	assert.Contains(t, report, "Lifecycle:  beta")
	assert.Contains(t, report, "Release:    v1.0.0-beta.1")
	assert.Contains(t, report, "Tag:        proto/payments/ledger/v1/v1.0.0-beta.1")
	assert.Contains(t, report, "Go module:  github.com/acme/apis/proto/payments/ledger")
	assert.Contains(t, report, "Go import:  github.com/acme/apis/proto/payments/ledger/v1")
}

func TestValidateLifecycle(t *testing.T) {
	for _, valid := range []string{"experimental", "preview", "beta", "stable", "deprecated", "sunset"} {
		assert.NoError(t, ValidateLifecycle(valid))
	}
	assert.Error(t, ValidateLifecycle("alpha"))
	assert.Error(t, ValidateLifecycle(""))
	assert.Error(t, ValidateLifecycle("ga"))
}

func TestFormatIdentityReportNoLifecycle(t *testing.T) {
	api := &APIIdentity{ID: "proto/payments/ledger/v1", Format: "proto",
		Domain: "payments", Name: "ledger", Line: "v1"}
	report := FormatIdentityReport(api, nil, nil, nil)
	assert.Contains(t, report, "API:        proto/payments/ledger/v1")
	assert.False(t, strings.Contains(report, "Lifecycle:"))
	assert.False(t, strings.Contains(report, "Release:"))
	assert.False(t, strings.Contains(report, "Go module:"))
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
	api, source, release, langs, err := BuildIdentityBlock(
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
	// v0 module has no version suffix (like v1)
	assert.Equal(t, "github.com/acme/apis/proto/payments/ledger", langs["go"].Module)
	assert.Equal(t, "github.com/acme/apis/proto/payments/ledger/v0", langs["go"].Import)
}

func TestDeriveGoModuleV0(t *testing.T) {
	api := &APIIdentity{Format: "proto", Domain: "payments", Name: "ledger", Line: "v0"}
	got, err := DeriveGoModule("github.com/acme/apis", api)
	require.NoError(t, err)
	assert.Equal(t, "github.com/acme/apis/proto/payments/ledger", got)
}

func TestDeriveGoImportV0(t *testing.T) {
	api := &APIIdentity{Format: "proto", Domain: "payments", Name: "ledger", Line: "v0"}
	got, err := DeriveGoImport("github.com/acme/apis", api)
	require.NoError(t, err)
	assert.Equal(t, "github.com/acme/apis/proto/payments/ledger/v0", got)
}

package publisher

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateGoMod(t *testing.T) {
	tests := []struct {
		name       string
		modulePath string
		goVersion  string
		want       string
		wantErr    bool
	}{
		{
			name:       "basic v1 module",
			modulePath: "github.com/acme/apis/proto/payments/ledger",
			goVersion:  "1.21",
			want:       "module github.com/acme/apis/proto/payments/ledger\n\ngo 1.21\n",
		},
		{
			name:       "v2 module",
			modulePath: "github.com/acme/apis/proto/payments/ledger/v2",
			goVersion:  "1.22",
			want:       "module github.com/acme/apis/proto/payments/ledger/v2\n\ngo 1.22\n",
		},
		{
			name:       "go prefix stripped",
			modulePath: "github.com/acme/apis/proto/payments/ledger",
			goVersion:  "go1.21",
			want:       "module github.com/acme/apis/proto/payments/ledger\n\ngo 1.21\n",
		},
		{
			name:       "default go version",
			modulePath: "github.com/acme/apis/proto/payments/ledger",
			goVersion:  "",
			want:       "module github.com/acme/apis/proto/payments/ledger\n\ngo 1.21\n",
		},
		{
			name:       "empty module path",
			modulePath: "",
			goVersion:  "1.21",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateGoMod(tt.modulePath, tt.goVersion)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(got))
		})
	}
}

func TestParseGoModModule(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
		wantErr bool
	}{
		{
			name:    "standard go.mod",
			content: "module github.com/acme/apis/proto/payments/ledger\n\ngo 1.21\n",
			want:    "github.com/acme/apis/proto/payments/ledger",
		},
		{
			name:    "with require block",
			content: "module github.com/acme/apis/proto/payments/ledger/v2\n\ngo 1.22\n\nrequire (\n\tgoogle.golang.org/protobuf v1.31.0\n)\n",
			want:    "github.com/acme/apis/proto/payments/ledger/v2",
		},
		{
			name:    "no module directive",
			content: "go 1.21\n",
			wantErr: true,
		},
		{
			name:    "empty content",
			content: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseGoModModule([]byte(tt.content))
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

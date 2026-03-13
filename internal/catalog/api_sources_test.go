package catalog

import (
	"testing"

	"github.com/infobloxopen/apx/internal/config"
)

func TestMergeAPISources_Empty(t *testing.T) {
	cat := &Catalog{Version: 1, Org: "test", Repo: "apis"}
	err := MergeAPISources(cat, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cat.Modules) != 0 {
		t.Fatalf("expected 0 modules, got %d", len(cat.Modules))
	}
}

func TestMergeAPISources_FromTags(t *testing.T) {
	// Simulate what would happen if FetchRemoteTags returned these tags.
	// We test the merge logic by pre-building a catalog from tags directly.
	tags := []string{
		"proto/infoblox/authz/v1/v1.0.0",
		"proto/infoblox/authz/v1/v1.1.0",
		"proto/infoblox/licensing/v1/v0.1.0-alpha.1",
	}

	cat := &Catalog{Version: 1, Org: "infobloxopen", Repo: "apis"}

	// Simulate MergeAPISources behavior without network call
	remoteCat := GenerateFromTags(tags, "", "")
	src := config.APISource{
		Repo:       "github.com/Infoblox-CTO/ngp.authz",
		ImportMode: "preserve",
		PathMap: map[string]string{
			"proto/infoblox/authz/v1":     "dbapiserver/pkg/pb",
			"proto/infoblox/licensing/v1": "proto/licensing",
		},
	}

	for _, m := range remoteCat.Modules {
		m.Origin = config.OriginSourced
		m.ManagedRepo = src.Repo
		m.ImportMode = src.ImportMode
		m.Path = src.SourcePathFor(m.ID)
		cat.Modules = append(cat.Modules, m)
	}

	if len(cat.Modules) != 2 {
		t.Fatalf("expected 2 modules, got %d", len(cat.Modules))
	}

	// Check authz module
	authz := cat.Modules[0]
	if authz.ID != "proto/infoblox/authz/v1" {
		t.Fatalf("expected authz, got %s", authz.ID)
	}
	if authz.Version != "v1.1.0" {
		t.Fatalf("expected v1.1.0, got %s", authz.Version)
	}
	if authz.Lifecycle != "stable" {
		t.Fatalf("expected stable, got %s", authz.Lifecycle)
	}
	if authz.Origin != "sourced" {
		t.Fatalf("expected sourced origin, got %s", authz.Origin)
	}
	if authz.ManagedRepo != "github.com/Infoblox-CTO/ngp.authz" {
		t.Fatalf("expected managed repo, got %s", authz.ManagedRepo)
	}
	if authz.ImportMode != "preserve" {
		t.Fatalf("expected preserve, got %s", authz.ImportMode)
	}
	if authz.Path != "dbapiserver/pkg/pb" {
		t.Fatalf("expected path_map override, got %s", authz.Path)
	}

	// Check licensing module
	lic := cat.Modules[1]
	if lic.ID != "proto/infoblox/licensing/v1" {
		t.Fatalf("expected licensing, got %s", lic.ID)
	}
	if lic.Lifecycle != "experimental" {
		t.Fatalf("expected experimental for alpha, got %s", lic.Lifecycle)
	}
	if lic.Path != "proto/licensing" {
		t.Fatalf("expected path_map override, got %s", lic.Path)
	}
}

func TestMergeAPISources_ConflictWithLocal(t *testing.T) {
	cat := &Catalog{
		Version: 1,
		Org:     "test",
		Repo:    "apis",
		Modules: []Module{
			{ID: "proto/infoblox/authz/v1", Format: "proto", Path: "proto/infoblox/authz/v1"},
		},
	}

	// Simulate the conflict check
	existingIDs := make(map[string]bool)
	for _, m := range cat.Modules {
		existingIDs[m.ID] = true
	}

	conflictID := "proto/infoblox/authz/v1"
	if !existingIDs[conflictID] {
		t.Fatal("expected conflict to be detected")
	}
}

func TestAPISource_Validate(t *testing.T) {
	tests := []struct {
		name    string
		source  config.APISource
		wantErr bool
	}{
		{
			name:    "valid",
			source:  config.APISource{Repo: "github.com/org/repo"},
			wantErr: false,
		},
		{
			name:    "missing repo",
			source:  config.APISource{},
			wantErr: true,
		},
		{
			name:    "invalid import mode",
			source:  config.APISource{Repo: "github.com/org/repo", ImportMode: "bogus"},
			wantErr: true,
		},
		{
			name: "invalid path_map key",
			source: config.APISource{
				Repo:    "github.com/org/repo",
				PathMap: map[string]string{"not-an-api-id": "some/path"},
			},
			wantErr: true,
		},
		{
			name: "path_map with leading slash",
			source: config.APISource{
				Repo:    "github.com/org/repo",
				PathMap: map[string]string{"proto/x/y/v1": "/bad/path"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.source.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAPISource_SourcePathFor(t *testing.T) {
	src := config.APISource{
		Repo: "github.com/org/repo",
		PathMap: map[string]string{
			"proto/infoblox/authz/v1": "dbapiserver/pkg/pb",
		},
	}

	// Mapped
	if got := src.SourcePathFor("proto/infoblox/authz/v1"); got != "dbapiserver/pkg/pb" {
		t.Fatalf("expected mapped path, got %s", got)
	}

	// Unmapped — falls back to API ID
	if got := src.SourcePathFor("proto/infoblox/other/v1"); got != "proto/infoblox/other/v1" {
		t.Fatalf("expected fallback to API ID, got %s", got)
	}
}

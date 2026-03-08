package catalog

import (
	"testing"

	"github.com/infobloxopen/apx/internal/config"
)

func TestMergeExternalAPIs(t *testing.T) {
	t.Run("merge external into empty catalog", func(t *testing.T) {
		cat := &Catalog{
			Version: 1,
			Org:     "testorg",
			Repo:    "apis",
			Modules: []Module{},
		}
		externals := []config.ExternalRegistration{
			{
				ID:           "proto/google/pubsub/v1",
				ManagedRepo:  "github.com/Infoblox-CTO/apis-contrib-google",
				ManagedPath:  "google/pubsub/v1",
				UpstreamRepo: "github.com/googleapis/googleapis",
				UpstreamPath: "google/pubsub/v1",
				ImportMode:   config.ImportModePreserve,
				Origin:       config.OriginExternal,
				Description:  "Google Pub/Sub API",
				Lifecycle:    "stable",
				Version:      "v1.0.0",
				Owners:       []string{"platform-team"},
				Tags:         []string{"google", "messaging"},
			},
		}

		err := MergeExternalAPIs(cat, externals)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cat.Modules) != 1 {
			t.Fatalf("expected 1 module, got %d", len(cat.Modules))
		}

		m := cat.Modules[0]
		if m.ID != "proto/google/pubsub/v1" {
			t.Errorf("unexpected ID: %s", m.ID)
		}
		if m.Origin != config.OriginExternal {
			t.Errorf("expected origin %q, got %q", config.OriginExternal, m.Origin)
		}
		if m.ManagedRepo != "github.com/Infoblox-CTO/apis-contrib-google" {
			t.Errorf("unexpected managed_repo: %s", m.ManagedRepo)
		}
		if m.UpstreamRepo != "github.com/googleapis/googleapis" {
			t.Errorf("unexpected upstream_repo: %s", m.UpstreamRepo)
		}
		if m.ImportMode != config.ImportModePreserve {
			t.Errorf("expected import_mode %q, got %q", config.ImportModePreserve, m.ImportMode)
		}
		if m.Format != "proto" {
			t.Errorf("expected format %q, got %q", "proto", m.Format)
		}
		if m.Domain != "google" {
			t.Errorf("expected domain %q, got %q", "google", m.Domain)
		}
		if m.APILine != "v1" {
			t.Errorf("expected api_line %q, got %q", "v1", m.APILine)
		}
		if m.Path != "google/pubsub/v1" {
			t.Errorf("expected path %q, got %q", "google/pubsub/v1", m.Path)
		}
		if m.Description != "Google Pub/Sub API" {
			t.Errorf("unexpected description: %s", m.Description)
		}
		if m.Version != "v1.0.0" {
			t.Errorf("unexpected version: %s", m.Version)
		}
		if m.LatestStable != "v1.0.0" {
			t.Errorf("expected latest_stable %q, got %q", "v1.0.0", m.LatestStable)
		}
	})

	t.Run("merge into catalog with first-party modules", func(t *testing.T) {
		cat := &Catalog{
			Version: 1,
			Org:     "testorg",
			Repo:    "apis",
			Modules: []Module{
				{
					ID:     "proto/payments/ledger/v1",
					Format: "proto",
					Domain: "payments",
					Path:   "proto/payments/ledger/v1",
				},
			},
		}
		externals := []config.ExternalRegistration{
			{
				ID:           "proto/google/pubsub/v1",
				ManagedRepo:  "github.com/Infoblox-CTO/apis-contrib-google",
				ManagedPath:  "google/pubsub/v1",
				UpstreamRepo: "github.com/googleapis/googleapis",
				UpstreamPath: "google/pubsub/v1",
				ImportMode:   config.ImportModePreserve,
				Origin:       config.OriginExternal,
			},
		}

		err := MergeExternalAPIs(cat, externals)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cat.Modules) != 2 {
			t.Fatalf("expected 2 modules, got %d", len(cat.Modules))
		}
	})

	t.Run("duplicate external ID with first-party", func(t *testing.T) {
		cat := &Catalog{
			Version: 1,
			Modules: []Module{
				{ID: "proto/google/pubsub/v1", Format: "proto", Path: "proto/google/pubsub/v1"},
			},
		}
		externals := []config.ExternalRegistration{
			{
				ID:           "proto/google/pubsub/v1",
				ManagedRepo:  "github.com/Infoblox-CTO/apis-contrib-google",
				ManagedPath:  "google/pubsub/v1",
				UpstreamRepo: "github.com/googleapis/googleapis",
				UpstreamPath: "google/pubsub/v1",
				Origin:       config.OriginExternal,
				ImportMode:   config.ImportModePreserve,
			},
		}

		err := MergeExternalAPIs(cat, externals)
		if err == nil {
			t.Error("expected error for duplicate ID conflict with first-party")
		}
	})

	t.Run("path conflict detection", func(t *testing.T) {
		cat := &Catalog{
			Version: 1,
			Modules: []Module{
				{ID: "proto/payments/ledger/v1", Format: "proto", Path: "google/pubsub/v1"},
			},
		}
		externals := []config.ExternalRegistration{
			{
				ID:           "proto/google/pubsub/v1",
				ManagedRepo:  "github.com/Infoblox-CTO/apis-contrib-google",
				ManagedPath:  "google/pubsub/v1",
				UpstreamRepo: "github.com/googleapis/googleapis",
				UpstreamPath: "google/pubsub/v1",
				Origin:       config.OriginExternal,
				ImportMode:   config.ImportModePreserve,
			},
		}

		err := MergeExternalAPIs(cat, externals)
		if err == nil {
			t.Error("expected error for path conflict")
		}
	})

	t.Run("replaces existing external entries on re-merge", func(t *testing.T) {
		cat := &Catalog{
			Version: 1,
			Modules: []Module{
				{ID: "proto/payments/ledger/v1", Format: "proto", Path: "proto/payments/ledger/v1"},
				{ID: "proto/google/pubsub/v1", Format: "proto", Path: "google/pubsub/v1", Origin: config.OriginExternal, Version: "v0.9.0"},
			},
		}
		externals := []config.ExternalRegistration{
			{
				ID:           "proto/google/pubsub/v1",
				ManagedRepo:  "github.com/Infoblox-CTO/apis-contrib-google",
				ManagedPath:  "google/pubsub/v1",
				UpstreamRepo: "github.com/googleapis/googleapis",
				UpstreamPath: "google/pubsub/v1",
				Origin:       config.OriginExternal,
				ImportMode:   config.ImportModePreserve,
				Version:      "v1.0.0",
			},
		}

		err := MergeExternalAPIs(cat, externals)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cat.Modules) != 2 {
			t.Fatalf("expected 2 modules, got %d", len(cat.Modules))
		}
		// The external module should be updated
		for _, m := range cat.Modules {
			if m.ID == "proto/google/pubsub/v1" {
				if m.Version != "v1.0.0" {
					t.Errorf("expected version v1.0.0, got %s", m.Version)
				}
				return
			}
		}
		t.Error("external module not found after re-merge")
	})

	t.Run("empty externals is no-op", func(t *testing.T) {
		cat := &Catalog{
			Version: 1,
			Modules: []Module{
				{ID: "proto/payments/ledger/v1"},
			},
		}
		err := MergeExternalAPIs(cat, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cat.Modules) != 1 {
			t.Errorf("expected 1 module unchanged, got %d", len(cat.Modules))
		}
	})

	t.Run("prerelease version sets latest_prerelease", func(t *testing.T) {
		cat := &Catalog{Version: 1, Modules: []Module{}}
		externals := []config.ExternalRegistration{
			{
				ID:           "proto/google/pubsub/v1",
				ManagedRepo:  "github.com/Infoblox-CTO/apis-contrib-google",
				ManagedPath:  "google/pubsub/v1",
				UpstreamRepo: "github.com/googleapis/googleapis",
				UpstreamPath: "google/pubsub/v1",
				Origin:       config.OriginExternal,
				ImportMode:   config.ImportModePreserve,
				Version:      "v1.0.0-beta.1",
				Lifecycle:    "beta",
			},
		}
		err := MergeExternalAPIs(cat, externals)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		m := cat.Modules[0]
		if m.LatestPrerelease != "v1.0.0-beta.1" {
			t.Errorf("expected latest_prerelease %q, got %q", "v1.0.0-beta.1", m.LatestPrerelease)
		}
		if m.LatestStable != "" {
			t.Errorf("expected empty latest_stable for prerelease, got %q", m.LatestStable)
		}
	})

	t.Run("forked origin preserved", func(t *testing.T) {
		cat := &Catalog{Version: 1, Modules: []Module{}}
		externals := []config.ExternalRegistration{
			{
				ID:           "proto/google/pubsub/v1",
				ManagedRepo:  "github.com/Infoblox-CTO/apis-contrib-google",
				ManagedPath:  "google/pubsub/v1",
				UpstreamRepo: "github.com/googleapis/googleapis",
				UpstreamPath: "google/pubsub/v1",
				Origin:       config.OriginForked,
				ImportMode:   config.ImportModeRewrite,
			},
		}
		err := MergeExternalAPIs(cat, externals)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		m := cat.Modules[0]
		if m.Origin != config.OriginForked {
			t.Errorf("expected origin %q, got %q", config.OriginForked, m.Origin)
		}
		if m.ImportMode != config.ImportModeRewrite {
			t.Errorf("expected import_mode %q, got %q", config.ImportModeRewrite, m.ImportMode)
		}
	})
}

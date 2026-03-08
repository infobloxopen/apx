package config

import (
	"errors"
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestExternalRegistration_Validate(t *testing.T) {
	validReg := func() *ExternalRegistration {
		return &ExternalRegistration{
			ID:           "proto/google/pubsub/v1",
			ManagedRepo:  "github.com/Infoblox-CTO/apis-contrib-google",
			ManagedPath:  "google/pubsub/v1",
			UpstreamRepo: "github.com/googleapis/googleapis",
			UpstreamPath: "google/pubsub/v1",
		}
	}

	t.Run("valid registration applies defaults", func(t *testing.T) {
		reg := validReg()
		if err := reg.Validate(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if reg.ImportMode != ImportModePreserve {
			t.Errorf("expected import_mode %q, got %q", ImportModePreserve, reg.ImportMode)
		}
		if reg.Origin != OriginExternal {
			t.Errorf("expected origin %q, got %q", OriginExternal, reg.Origin)
		}
	})

	t.Run("valid registration with explicit values", func(t *testing.T) {
		reg := validReg()
		reg.ImportMode = ImportModeRewrite
		reg.Origin = OriginForked
		reg.Lifecycle = "stable"
		reg.Description = "Google Pub/Sub"
		reg.Version = "v1.0.0"
		reg.Owners = []string{"platform-team"}
		reg.Tags = []string{"google", "messaging"}
		if err := reg.Validate(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("missing API ID", func(t *testing.T) {
		reg := validReg()
		reg.ID = ""
		if err := reg.Validate(); err == nil {
			t.Error("expected error for empty ID")
		}
	})

	t.Run("invalid API ID format", func(t *testing.T) {
		reg := validReg()
		reg.ID = "not-valid"
		if err := reg.Validate(); err == nil {
			t.Error("expected error for invalid API ID")
		}
	})

	t.Run("invalid API ID bad format segment", func(t *testing.T) {
		reg := validReg()
		reg.ID = "badformat/google/pubsub/v1"
		if err := reg.Validate(); err == nil {
			t.Error("expected error for bad format segment")
		}
	})

	t.Run("missing managed_repo", func(t *testing.T) {
		reg := validReg()
		reg.ManagedRepo = ""
		if err := reg.Validate(); err == nil {
			t.Error("expected error for empty managed_repo")
		}
	})

	t.Run("missing managed_path", func(t *testing.T) {
		reg := validReg()
		reg.ManagedPath = ""
		if err := reg.Validate(); err == nil {
			t.Error("expected error for empty managed_path")
		}
	})

	t.Run("missing upstream_repo", func(t *testing.T) {
		reg := validReg()
		reg.UpstreamRepo = ""
		if err := reg.Validate(); err == nil {
			t.Error("expected error for empty upstream_repo")
		}
	})

	t.Run("missing upstream_path", func(t *testing.T) {
		reg := validReg()
		reg.UpstreamPath = ""
		if err := reg.Validate(); err == nil {
			t.Error("expected error for empty upstream_path")
		}
	})

	t.Run("managed_path with leading slash", func(t *testing.T) {
		reg := validReg()
		reg.ManagedPath = "/google/pubsub/v1"
		if err := reg.Validate(); err == nil {
			t.Error("expected error for leading slash in managed_path")
		}
	})

	t.Run("managed_path with trailing slash", func(t *testing.T) {
		reg := validReg()
		reg.ManagedPath = "google/pubsub/v1/"
		if err := reg.Validate(); err == nil {
			t.Error("expected error for trailing slash in managed_path")
		}
	})

	t.Run("managed_path with dot-dot traversal", func(t *testing.T) {
		reg := validReg()
		reg.ManagedPath = "google/../evil"
		if err := reg.Validate(); err == nil {
			t.Error("expected error for .. in managed_path")
		}
	})

	t.Run("invalid import_mode", func(t *testing.T) {
		reg := validReg()
		reg.ImportMode = "invalid"
		err := reg.Validate()
		if err == nil {
			t.Error("expected error for invalid import_mode")
		}
		if !errors.Is(err, ErrExternalInvalidMode) {
			t.Errorf("expected ErrExternalInvalidMode, got %v", err)
		}
	})

	t.Run("invalid origin", func(t *testing.T) {
		reg := validReg()
		reg.Origin = "invalid"
		err := reg.Validate()
		if err == nil {
			t.Error("expected error for invalid origin")
		}
		if !errors.Is(err, ErrExternalInvalidOrigin) {
			t.Errorf("expected ErrExternalInvalidOrigin, got %v", err)
		}
	})

	t.Run("invalid lifecycle", func(t *testing.T) {
		reg := validReg()
		reg.Lifecycle = "invalid"
		if err := reg.Validate(); err == nil {
			t.Error("expected error for invalid lifecycle")
		}
	})

	t.Run("malformed managed_repo URL", func(t *testing.T) {
		reg := validReg()
		reg.ManagedRepo = "nope"
		if err := reg.Validate(); err == nil {
			t.Error("expected error for malformed managed_repo URL")
		}
	})

	t.Run("malformed upstream_repo URL", func(t *testing.T) {
		reg := validReg()
		reg.UpstreamRepo = "nope"
		if err := reg.Validate(); err == nil {
			t.Error("expected error for malformed upstream_repo URL")
		}
	})

	t.Run("full URL form for repos", func(t *testing.T) {
		reg := validReg()
		reg.ManagedRepo = "https://github.com/Infoblox-CTO/apis-contrib-google"
		reg.UpstreamRepo = "https://github.com/googleapis/googleapis"
		if err := reg.Validate(); err != nil {
			t.Fatalf("unexpected error with full URLs: %v", err)
		}
	})
}

func TestAddExternal(t *testing.T) {
	t.Run("add valid registration", func(t *testing.T) {
		cfg := &Config{}
		reg := &ExternalRegistration{
			ID:           "proto/google/pubsub/v1",
			ManagedRepo:  "github.com/Infoblox-CTO/apis-contrib-google",
			ManagedPath:  "google/pubsub/v1",
			UpstreamRepo: "github.com/googleapis/googleapis",
			UpstreamPath: "google/pubsub/v1",
		}
		if err := AddExternal(cfg, reg, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cfg.ExternalAPIs) != 1 {
			t.Fatalf("expected 1 external API, got %d", len(cfg.ExternalAPIs))
		}
		if cfg.ExternalAPIs[0].ID != "proto/google/pubsub/v1" {
			t.Errorf("unexpected ID: %s", cfg.ExternalAPIs[0].ID)
		}
	})

	t.Run("duplicate ID rejected", func(t *testing.T) {
		cfg := &Config{
			ExternalAPIs: []ExternalRegistration{
				{ID: "proto/google/pubsub/v1", ManagedPath: "google/pubsub/v1"},
			},
		}
		reg := &ExternalRegistration{
			ID:           "proto/google/pubsub/v1",
			ManagedRepo:  "github.com/Infoblox-CTO/apis-contrib-google",
			ManagedPath:  "google/pubsub/v1-copy",
			UpstreamRepo: "github.com/googleapis/googleapis",
			UpstreamPath: "google/pubsub/v1",
		}
		err := AddExternal(cfg, reg, nil)
		if err == nil {
			t.Error("expected error for duplicate ID")
		}
		if !errors.Is(err, ErrExternalDuplicateID) {
			t.Errorf("expected ErrExternalDuplicateID, got %v", err)
		}
	})

	t.Run("path conflict rejected", func(t *testing.T) {
		cfg := &Config{}
		reg := &ExternalRegistration{
			ID:           "proto/google/pubsub/v1",
			ManagedRepo:  "github.com/Infoblox-CTO/apis-contrib-google",
			ManagedPath:  "google/pubsub/v1",
			UpstreamRepo: "github.com/googleapis/googleapis",
			UpstreamPath: "google/pubsub/v1",
		}
		existingPaths := []string{"google/pubsub/v1"}
		err := AddExternal(cfg, reg, existingPaths)
		if err == nil {
			t.Error("expected error for path conflict")
		}
		if !errors.Is(err, ErrExternalPathConflict) {
			t.Errorf("expected ErrExternalPathConflict, got %v", err)
		}
	})

	t.Run("invalid registration rejected", func(t *testing.T) {
		cfg := &Config{}
		reg := &ExternalRegistration{
			ID: "invalid",
		}
		if err := AddExternal(cfg, reg, nil); err == nil {
			t.Error("expected error for invalid registration")
		}
	})
}

func TestRemoveExternal(t *testing.T) {
	t.Run("remove existing", func(t *testing.T) {
		cfg := &Config{
			ExternalAPIs: []ExternalRegistration{
				{ID: "proto/google/pubsub/v1"},
				{ID: "proto/google/api/v1"},
			},
		}
		if err := RemoveExternal(cfg, "proto/google/pubsub/v1"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cfg.ExternalAPIs) != 1 {
			t.Fatalf("expected 1 external API, got %d", len(cfg.ExternalAPIs))
		}
		if cfg.ExternalAPIs[0].ID != "proto/google/api/v1" {
			t.Errorf("wrong API remained: %s", cfg.ExternalAPIs[0].ID)
		}
	})

	t.Run("remove non-existent", func(t *testing.T) {
		cfg := &Config{}
		err := RemoveExternal(cfg, "proto/google/missing/v1")
		if !errors.Is(err, ErrExternalNotFound) {
			t.Errorf("expected ErrExternalNotFound, got %v", err)
		}
	})
}

func TestFindExternalByID(t *testing.T) {
	cfg := &Config{
		ExternalAPIs: []ExternalRegistration{
			{ID: "proto/google/pubsub/v1", Description: "Pub/Sub"},
		},
	}

	t.Run("found", func(t *testing.T) {
		reg, err := FindExternalByID(cfg, "proto/google/pubsub/v1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if reg.Description != "Pub/Sub" {
			t.Errorf("unexpected description: %s", reg.Description)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := FindExternalByID(cfg, "proto/google/missing/v1")
		if !errors.Is(err, ErrExternalNotFound) {
			t.Errorf("expected ErrExternalNotFound, got %v", err)
		}
	})
}

func TestListExternals(t *testing.T) {
	cfg := &Config{
		ExternalAPIs: []ExternalRegistration{
			{ID: "proto/google/pubsub/v1", Origin: OriginExternal},
			{ID: "proto/google/api/v1", Origin: OriginForked},
			{ID: "proto/google/maps/v1", Origin: OriginExternal},
		},
	}

	t.Run("all", func(t *testing.T) {
		result := ListExternals(cfg, "")
		if len(result) != 3 {
			t.Errorf("expected 3, got %d", len(result))
		}
	})

	t.Run("external only", func(t *testing.T) {
		result := ListExternals(cfg, OriginExternal)
		if len(result) != 2 {
			t.Errorf("expected 2 external, got %d", len(result))
		}
	})

	t.Run("forked only", func(t *testing.T) {
		result := ListExternals(cfg, OriginForked)
		if len(result) != 1 {
			t.Errorf("expected 1 forked, got %d", len(result))
		}
	})
}

func TestTransitionExternal(t *testing.T) {
	t.Run("external to forked", func(t *testing.T) {
		cfg := &Config{
			ExternalAPIs: []ExternalRegistration{
				{
					ID:         "proto/google/pubsub/v1",
					Origin:     OriginExternal,
					ImportMode: ImportModePreserve,
				},
			},
		}
		if err := TransitionExternal(cfg, "proto/google/pubsub/v1", OriginForked); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		reg, _ := FindExternalByID(cfg, "proto/google/pubsub/v1")
		if reg.Origin != OriginForked {
			t.Errorf("expected origin %q, got %q", OriginForked, reg.Origin)
		}
		if reg.ImportMode != ImportModeRewrite {
			t.Errorf("expected import_mode %q, got %q", ImportModeRewrite, reg.ImportMode)
		}
	})

	t.Run("forked to external", func(t *testing.T) {
		cfg := &Config{
			ExternalAPIs: []ExternalRegistration{
				{
					ID:         "proto/google/pubsub/v1",
					Origin:     OriginForked,
					ImportMode: ImportModeRewrite,
				},
			},
		}
		if err := TransitionExternal(cfg, "proto/google/pubsub/v1", OriginExternal); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		reg, _ := FindExternalByID(cfg, "proto/google/pubsub/v1")
		if reg.Origin != OriginExternal {
			t.Errorf("expected origin %q, got %q", OriginExternal, reg.Origin)
		}
		if reg.ImportMode != ImportModePreserve {
			t.Errorf("expected import_mode %q, got %q", ImportModePreserve, reg.ImportMode)
		}
	})

	t.Run("already at target", func(t *testing.T) {
		cfg := &Config{
			ExternalAPIs: []ExternalRegistration{
				{ID: "proto/google/pubsub/v1", Origin: OriginExternal},
			},
		}
		err := TransitionExternal(cfg, "proto/google/pubsub/v1", OriginExternal)
		if !errors.Is(err, ErrExternalAlreadyTarget) {
			t.Errorf("expected ErrExternalAlreadyTarget, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		cfg := &Config{}
		err := TransitionExternal(cfg, "proto/google/missing/v1", OriginForked)
		if !errors.Is(err, ErrExternalNotFound) {
			t.Errorf("expected ErrExternalNotFound, got %v", err)
		}
	})

	t.Run("invalid target origin", func(t *testing.T) {
		cfg := &Config{}
		err := TransitionExternal(cfg, "proto/google/pubsub/v1", "invalid")
		if !errors.Is(err, ErrExternalInvalidOrigin) {
			t.Errorf("expected ErrExternalInvalidOrigin, got %v", err)
		}
	})
}

func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := tmpDir + "/apx.yaml"

	cfg := &Config{
		Version:    1,
		Org:        "testorg",
		Repo:       "apis",
		Publishing: Publishing{TagFormat: "{api}/{version}"},
		Execution:  Execution{Mode: "local"},
		ExternalAPIs: []ExternalRegistration{
			{
				ID:           "proto/google/pubsub/v1",
				ManagedRepo:  "github.com/Infoblox-CTO/apis-contrib-google",
				ManagedPath:  "google/pubsub/v1",
				UpstreamRepo: "github.com/googleapis/googleapis",
				UpstreamPath: "google/pubsub/v1",
				ImportMode:   ImportModePreserve,
				Origin:       OriginExternal,
			},
		},
	}

	if err := SaveConfig(cfg, configPath); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Read back the raw YAML and verify it round-trips
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read saved config: %v", err)
	}

	var loaded Config
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("failed to parse saved config: %v", err)
	}

	if len(loaded.ExternalAPIs) != 1 {
		t.Fatalf("expected 1 external API after reload, got %d", len(loaded.ExternalAPIs))
	}
	if loaded.ExternalAPIs[0].ID != "proto/google/pubsub/v1" {
		t.Errorf("unexpected ID after reload: %s", loaded.ExternalAPIs[0].ID)
	}
	if loaded.ExternalAPIs[0].ImportMode != ImportModePreserve {
		t.Errorf("unexpected import_mode after reload: %s", loaded.ExternalAPIs[0].ImportMode)
	}
}

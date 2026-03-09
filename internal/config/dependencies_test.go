package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAddDependency(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "apx.yaml")
	lockPath := filepath.Join(tmpDir, "apx.lock")

	// Create initial apx.yaml and apx.lock files
	initialConfig := `dependencies: []
`
	initialLock := `dependencies: {}
`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("failed to create initial apx.yaml: %v", err)
	}
	if err := os.WriteFile(lockPath, []byte(initialLock), 0644); err != nil {
		t.Fatalf("failed to create initial apx.lock: %v", err)
	}

	tests := []struct {
		name        string
		modulePath  string
		version     string
		expectError bool
	}{
		{
			name:        "add valid dependency",
			modulePath:  "proto/payments/ledger/v1",
			version:     "v1.2.3",
			expectError: false,
		},
		{
			name:        "add without version",
			modulePath:  "proto/payments/wallet/v1",
			version:     "",
			expectError: false, // Should fetch latest
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := NewDependencyManager(configPath, lockPath, "github.com/acme/apis")
			err := mgr.Add(tt.modulePath, tt.version)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectError {
				// Verify dependency added
				deps, err := mgr.List()
				if err != nil {
					t.Fatalf("failed to list dependencies: %v", err)
				}
				found := false
				for _, dep := range deps {
					if dep.ModulePath == tt.modulePath {
						found = true
						if tt.version != "" && dep.Version != tt.version {
							t.Errorf("expected version %s, got %s", tt.version, dep.Version)
						}
					}
				}
				if !found {
					t.Errorf("dependency %s not found", tt.modulePath)
				}
			}
		})
	}
}

func TestAddDependency_SourceRepoInLock(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "apx.yaml")
	lockPath := filepath.Join(tmpDir, "apx.lock")

	os.WriteFile(configPath, []byte("dependencies: []\n"), 0644)
	os.WriteFile(lockPath, []byte("dependencies: {}\n"), 0644)

	mgr := NewDependencyManager(configPath, lockPath, "github.com/acme/apis")
	if err := mgr.Add("proto/payments/ledger/v1", "v1.2.3"); err != nil {
		t.Fatalf("failed to add: %v", err)
	}

	// Read lock file and verify repo field
	lockFile, err := mgr.loadLock()
	if err != nil {
		t.Fatalf("failed to load lock: %v", err)
	}
	dep, ok := lockFile.Dependencies["proto/payments/ledger/v1"]
	if !ok {
		t.Fatal("dependency not found in lock")
	}
	if dep.Repo != "github.com/acme/apis" {
		t.Errorf("expected repo github.com/acme/apis, got %s", dep.Repo)
	}

	// Verify empty sourceRepo falls back to placeholder
	mgr2 := NewDependencyManager(configPath, lockPath, "")
	if err := mgr2.Add("proto/orders/v1", "v1.0.0"); err != nil {
		t.Fatalf("failed to add: %v", err)
	}
	lockFile2, _ := mgr2.loadLock()
	dep2 := lockFile2.Dependencies["proto/orders/v1"]
	if dep2.Repo != "github.com/<org>/<repo>" {
		t.Errorf("expected placeholder repo, got %s", dep2.Repo)
	}
}

func TestRemoveDependency(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "apx.yaml")
	lockPath := filepath.Join(tmpDir, "apx.lock")

	// Create initial apx.yaml and apx.lock files
	initialConfig := `dependencies: []
`
	initialLock := `dependencies: {}
`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("failed to create initial apx.yaml: %v", err)
	}
	if err := os.WriteFile(lockPath, []byte(initialLock), 0644); err != nil {
		t.Fatalf("failed to create initial apx.lock: %v", err)
	}

	mgr := NewDependencyManager(configPath, lockPath, "github.com/acme/apis")

	// Add a dependency
	if err := mgr.Add("proto/payments/ledger/v1", "v1.2.3"); err != nil {
		t.Fatalf("failed to add dependency: %v", err)
	}

	// Remove it
	if err := mgr.Remove("proto/payments/ledger/v1"); err != nil {
		t.Fatalf("failed to remove dependency: %v", err)
	}

	// Verify it's gone
	deps, err := mgr.List()
	if err != nil {
		t.Fatalf("failed to list dependencies: %v", err)
	}
	for _, dep := range deps {
		if dep.ModulePath == "proto/payments/ledger/v1" {
			t.Error("dependency should have been removed")
		}
	}
}

func TestUpdateDependencyVersion(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "apx.yaml")
	lockPath := filepath.Join(tmpDir, "apx.lock")

	// Create initial apx.yaml and apx.lock files
	initialConfig := `dependencies: []
`
	initialLock := `dependencies: {}
`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("failed to create initial apx.yaml: %v", err)
	}
	if err := os.WriteFile(lockPath, []byte(initialLock), 0644); err != nil {
		t.Fatalf("failed to create initial apx.lock: %v", err)
	}

	mgr := NewDependencyManager(configPath, lockPath, "github.com/acme/apis")

	// Add dependency
	if err := mgr.Add("proto/payments/ledger/v1", "v1.2.3"); err != nil {
		t.Fatalf("failed to add dependency: %v", err)
	}

	// Update version
	if err := mgr.Add("proto/payments/ledger/v1", "v1.3.0"); err != nil {
		t.Fatalf("failed to update dependency: %v", err)
	}

	// Verify version updated
	deps, err := mgr.List()
	if err != nil {
		t.Fatalf("failed to list dependencies: %v", err)
	}
	found := false
	for _, dep := range deps {
		if dep.ModulePath == "proto/payments/ledger/v1" {
			found = true
			if dep.Version != "v1.3.0" {
				t.Errorf("expected version v1.3.0, got %s", dep.Version)
			}
		}
	}
	if !found {
		t.Error("dependency not found after update")
	}
}

func TestListDependencies(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "apx.yaml")
	lockPath := filepath.Join(tmpDir, "apx.lock")

	// Create initial apx.yaml and apx.lock files
	initialConfig := `dependencies: []
`
	initialLock := `dependencies: {}
`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("failed to create initial apx.yaml: %v", err)
	}
	if err := os.WriteFile(lockPath, []byte(initialLock), 0644); err != nil {
		t.Fatalf("failed to create initial apx.lock: %v", err)
	}

	mgr := NewDependencyManager(configPath, lockPath, "github.com/acme/apis")

	// Add multiple dependencies
	deps := []struct {
		path    string
		version string
	}{
		{"proto/payments/ledger/v1", "v1.2.3"},
		{"proto/payments/wallet/v1", "v1.0.0"},
		{"openapi/customer/accounts/v2", "v2.0.0"},
	}

	for _, dep := range deps {
		if err := mgr.Add(dep.path, dep.version); err != nil {
			t.Fatalf("failed to add dependency %s: %v", dep.path, err)
		}
	}

	// List all
	list, err := mgr.List()
	if err != nil {
		t.Fatalf("failed to list dependencies: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("expected 3 dependencies, got %d", len(list))
	}

	// Verify all are present
	found := make(map[string]bool)
	for _, dep := range list {
		found[dep.ModulePath] = true
	}
	for _, expected := range deps {
		if !found[expected.path] {
			t.Errorf("expected dependency %s not found", expected.path)
		}
	}
}

package catalog

import (
	"path/filepath"
	"testing"
)

func TestSearch(t *testing.T) {
	// Create temp catalog file
	tmpDir := t.TempDir()
	catalogPath := filepath.Join(tmpDir, "catalog.yaml")

	// Create test catalog
	catalog := &Catalog{
		Version: 1,
		Org:     "testorg",
		Repo:    "apis",
		Modules: []Module{
			{
				Name:        "proto/payments/ledger/v1",
				Format:      "proto",
				Description: "Payment ledger API",
				Version:     "v1.2.3",
				Path:        "proto/payments/ledger/v1",
				Owners:      []string{"@platform/payments"},
			},
			{
				Name:        "proto/payments/wallet/v1",
				Format:      "proto",
				Description: "Digital wallet API",
				Version:     "v1.0.0",
				Path:        "proto/payments/wallet/v1",
				Owners:      []string{"@platform/payments"},
			},
			{
				Name:        "openapi/customer/accounts/v2",
				Format:      "openapi",
				Description: "Customer accounts API",
				Version:     "v2.0.0",
				Path:        "openapi/customer/accounts/v2",
				Owners:      []string{"@platform/customer"},
			},
		},
	}

	gen := NewGenerator(catalogPath)
	if err := gen.Save(catalog); err != nil {
		t.Fatalf("failed to save test catalog: %v", err)
	}

	tests := []struct {
		name          string
		query         string
		format        string
		expectedCount int
		expectedNames []string
	}{
		{
			name:          "search all",
			query:         "",
			format:        "",
			expectedCount: 3,
			expectedNames: []string{"proto/payments/ledger/v1", "proto/payments/wallet/v1", "openapi/customer/accounts/v2"},
		},
		{
			name:          "search by name",
			query:         "ledger",
			format:        "",
			expectedCount: 1,
			expectedNames: []string{"proto/payments/ledger/v1"},
		},
		{
			name:          "search by format proto",
			query:         "",
			format:        "proto",
			expectedCount: 2,
			expectedNames: []string{"proto/payments/ledger/v1", "proto/payments/wallet/v1"},
		},
		{
			name:          "search by format openapi",
			query:         "",
			format:        "openapi",
			expectedCount: 1,
			expectedNames: []string{"openapi/customer/accounts/v2"},
		},
		{
			name:          "search by description",
			query:         "ledger",
			format:        "",
			expectedCount: 1,
			expectedNames: []string{"proto/payments/ledger/v1"},
		},
		{
			name:          "no matches",
			query:         "nonexistent",
			format:        "",
			expectedCount: 0,
			expectedNames: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := SearchModules(gen, tt.query, tt.format)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(results) != tt.expectedCount {
				t.Errorf("expected %d results, got %d", tt.expectedCount, len(results))
			}

			foundNames := make(map[string]bool)
			for _, result := range results {
				foundNames[result.Name] = true
			}

			for _, expectedName := range tt.expectedNames {
				if !foundNames[expectedName] {
					t.Errorf("expected module %s not found in results", expectedName)
				}
			}
		})
	}
}

func TestLoadCatalog(t *testing.T) {
	t.Run("load existing catalog", func(t *testing.T) {
		tmpDir := t.TempDir()
		catalogPath := filepath.Join(tmpDir, "catalog.yaml")

		// Create test catalog
		catalog := &Catalog{
			Version: 1,
			Org:     "testorg",
			Repo:    "apis",
			Modules: []Module{
				{
					Name:   "test-api",
					Format: "proto",
				},
			},
		}

		gen := NewGenerator(catalogPath)
		if err := gen.Save(catalog); err != nil {
			t.Fatalf("failed to save catalog: %v", err)
		}

		// Load it back
		loaded, err := gen.Load()
		if err != nil {
			t.Fatalf("failed to load catalog: %v", err)
		}

		if loaded.Org != "testorg" {
			t.Errorf("expected org 'testorg', got %s", loaded.Org)
		}

		if len(loaded.Modules) != 1 {
			t.Errorf("expected 1 module, got %d", len(loaded.Modules))
		}
	})

	t.Run("load non-existent catalog", func(t *testing.T) {
		gen := NewGenerator("/nonexistent/catalog.yaml")
		catalog, err := gen.Load()
		if err != nil {
			t.Fatalf("should return empty catalog, not error: %v", err)
		}

		if len(catalog.Modules) != 0 {
			t.Errorf("expected empty catalog, got %d modules", len(catalog.Modules))
		}
	})
}

func TestAddModule(t *testing.T) {
	tmpDir := t.TempDir()
	catalogPath := filepath.Join(tmpDir, "catalog.yaml")

	gen := NewGenerator(catalogPath)

	module := Module{
		Name:        "proto/test/api/v1",
		Format:      "proto",
		Description: "Test API",
		Version:     "v1.0.0",
	}

	if err := gen.AddModule(module); err != nil {
		t.Fatalf("failed to add module: %v", err)
	}

	catalog, err := gen.Load()
	if err != nil {
		t.Fatalf("failed to load catalog: %v", err)
	}

	if len(catalog.Modules) != 1 {
		t.Errorf("expected 1 module, got %d", len(catalog.Modules))
	}

	if catalog.Modules[0].Name != "proto/test/api/v1" {
		t.Errorf("expected module name 'proto/test/api/v1', got %s", catalog.Modules[0].Name)
	}
}

func TestRemoveModule(t *testing.T) {
	tmpDir := t.TempDir()
	catalogPath := filepath.Join(tmpDir, "catalog.yaml")

	gen := NewGenerator(catalogPath)

	// Add module
	module := Module{
		Name:   "proto/test/api/v1",
		Format: "proto",
	}
	if err := gen.AddModule(module); err != nil {
		t.Fatalf("failed to add module: %v", err)
	}

	// Remove it
	if err := gen.RemoveModule("proto/test/api/v1"); err != nil {
		t.Fatalf("failed to remove module: %v", err)
	}

	catalog, err := gen.Load()
	if err != nil {
		t.Fatalf("failed to load catalog: %v", err)
	}

	if len(catalog.Modules) != 0 {
		t.Errorf("expected 0 modules, got %d", len(catalog.Modules))
	}

	// Try removing non-existent module
	if err := gen.RemoveModule("nonexistent"); err == nil {
		t.Error("expected error when removing non-existent module")
	}
}

func TestSearchModulesFunction(t *testing.T) {
	// This test ensures the SearchModules function exists and works
	tmpDir := t.TempDir()
	catalogPath := filepath.Join(tmpDir, "catalog.yaml")

	gen := NewGenerator(catalogPath)

	// Add test modules
	gen.AddModule(Module{Name: "proto/test/v1", Format: "proto"})
	gen.AddModule(Module{Name: "openapi/test/v1", Format: "openapi"})

	// Search with no filters
	results, err := SearchModules(gen, "", "")
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	// Search by format
	results, err = SearchModules(gen, "", "proto")
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 proto result, got %d", len(results))
	}
}

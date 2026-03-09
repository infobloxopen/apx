package catalog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSearch(t *testing.T) {
	// Create temp catalog file
	tmpDir := t.TempDir()
	catalogPath := filepath.Join(tmpDir, "catalog.yaml")

	// Create test catalog with identity fields
	cat := &Catalog{
		Version: 1,
		Org:     "testorg",
		Repo:    "apis",
		Modules: []Module{
			{
				ID:          "proto/payments/ledger/v1",
				Name:        "proto/payments/ledger/v1",
				Format:      "proto",
				Domain:      "payments",
				APILine:     "v1",
				Description: "Payment ledger API",
				Version:     "v1.2.3",
				Lifecycle:   "stable",
				Path:        "proto/payments/ledger/v1",
				Owners:      []string{"@platform/payments"},
			},
			{
				ID:          "proto/payments/wallet/v1",
				Name:        "proto/payments/wallet/v1",
				Format:      "proto",
				Domain:      "payments",
				APILine:     "v1",
				Description: "Digital wallet API",
				Version:     "v1.0.0",
				Lifecycle:   "beta",
				Path:        "proto/payments/wallet/v1",
				Owners:      []string{"@platform/payments"},
			},
			{
				ID:               "openapi/customer/accounts/v2",
				Name:             "openapi/customer/accounts/v2",
				Format:           "openapi",
				Domain:           "customer",
				APILine:          "v2",
				Description:      "Customer accounts API",
				Version:          "v2.0.0",
				LatestStable:     "v2.0.0",
				LatestPrerelease: "v2.1.0-beta.1",
				Lifecycle:        "stable",
				Path:             "openapi/customer/accounts/v2",
				Owners:           []string{"@platform/customer"},
			},
		},
	}

	gen := NewGenerator(catalogPath)
	if err := gen.Save(cat); err != nil {
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
				foundNames[result.DisplayName()] = true
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

		cat := &Catalog{
			Version: 1,
			Org:     "testorg",
			Repo:    "apis",
			Modules: []Module{
				{
					ID:     "proto/test/api/v1",
					Name:   "proto/test/api/v1",
					Format: "proto",
				},
			},
		}

		gen := NewGenerator(catalogPath)
		if err := gen.Save(cat); err != nil {
			t.Fatalf("failed to save catalog: %v", err)
		}

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

		if loaded.Modules[0].ID != "proto/test/api/v1" {
			t.Errorf("expected ID 'proto/test/api/v1', got %s", loaded.Modules[0].ID)
		}
	})

	t.Run("load catalog with identity fields round-trip", func(t *testing.T) {
		tmpDir := t.TempDir()
		catalogPath := filepath.Join(tmpDir, "catalog.yaml")

		cat := &Catalog{
			Version: 1,
			Org:     "acme",
			Repo:    "apis",
			Modules: []Module{
				{
					ID:               "proto/payments/ledger/v1",
					Format:           "proto",
					Domain:           "payments",
					APILine:          "v1",
					Description:      "Ledger API",
					Version:          "v1.2.3",
					LatestStable:     "v1.2.3",
					LatestPrerelease: "v1.3.0-beta.1",
					Lifecycle:        "stable",
					Path:             "proto/payments/ledger/v1",
					Owners:           []string{"team-payments"},
				},
			},
		}

		gen := NewGenerator(catalogPath)
		if err := gen.Save(cat); err != nil {
			t.Fatalf("failed to save: %v", err)
		}

		loaded, err := gen.Load()
		if err != nil {
			t.Fatalf("failed to load: %v", err)
		}

		m := loaded.Modules[0]
		if m.ID != "proto/payments/ledger/v1" {
			t.Errorf("ID: got %s", m.ID)
		}
		if m.Domain != "payments" {
			t.Errorf("Domain: got %s", m.Domain)
		}
		if m.APILine != "v1" {
			t.Errorf("APILine: got %s", m.APILine)
		}
		if m.LatestStable != "v1.2.3" {
			t.Errorf("LatestStable: got %s", m.LatestStable)
		}
		if m.LatestPrerelease != "v1.3.0-beta.1" {
			t.Errorf("LatestPrerelease: got %s", m.LatestPrerelease)
		}
		if m.Lifecycle != "stable" {
			t.Errorf("Lifecycle: got %s", m.Lifecycle)
		}
	})

	t.Run("load non-existent catalog", func(t *testing.T) {
		gen := NewGenerator("/nonexistent/catalog.yaml")
		cat, err := gen.Load()
		if err != nil {
			t.Fatalf("should return empty catalog, not error: %v", err)
		}

		if len(cat.Modules) != 0 {
			t.Errorf("expected empty catalog, got %d modules", len(cat.Modules))
		}
	})
}

func TestAddModule(t *testing.T) {
	tmpDir := t.TempDir()
	catalogPath := filepath.Join(tmpDir, "catalog.yaml")

	gen := NewGenerator(catalogPath)

	module := Module{
		ID:          "proto/test/api/v1",
		Name:        "proto/test/api/v1",
		Format:      "proto",
		Domain:      "test",
		APILine:     "v1",
		Description: "Test API",
		Version:     "v1.0.0",
	}

	if err := gen.AddModule(module); err != nil {
		t.Fatalf("failed to add module: %v", err)
	}

	cat, err := gen.Load()
	if err != nil {
		t.Fatalf("failed to load catalog: %v", err)
	}

	if len(cat.Modules) != 1 {
		t.Errorf("expected 1 module, got %d", len(cat.Modules))
	}

	if cat.Modules[0].ID != "proto/test/api/v1" {
		t.Errorf("expected ID 'proto/test/api/v1', got %s", cat.Modules[0].ID)
	}

	// Update by ID should replace, not duplicate
	module.Version = "v1.1.0"
	module.Lifecycle = "stable"
	if err := gen.AddModule(module); err != nil {
		t.Fatalf("failed to update module: %v", err)
	}
	cat, _ = gen.Load()
	if len(cat.Modules) != 1 {
		t.Errorf("expected 1 module after update, got %d", len(cat.Modules))
	}
	if cat.Modules[0].Version != "v1.1.0" {
		t.Errorf("expected version v1.1.0 after update, got %s", cat.Modules[0].Version)
	}
}

func TestRemoveModule(t *testing.T) {
	tmpDir := t.TempDir()
	catalogPath := filepath.Join(tmpDir, "catalog.yaml")

	gen := NewGenerator(catalogPath)

	// Add module with ID
	module := Module{
		ID:     "proto/test/api/v1",
		Name:   "proto/test/api/v1",
		Format: "proto",
	}
	if err := gen.AddModule(module); err != nil {
		t.Fatalf("failed to add module: %v", err)
	}

	// Remove by ID
	if err := gen.RemoveModule("proto/test/api/v1"); err != nil {
		t.Fatalf("failed to remove module: %v", err)
	}

	cat, err := gen.Load()
	if err != nil {
		t.Fatalf("failed to load catalog: %v", err)
	}

	if len(cat.Modules) != 0 {
		t.Errorf("expected 0 modules, got %d", len(cat.Modules))
	}

	// Try removing non-existent module
	if err := gen.RemoveModule("nonexistent"); err == nil {
		t.Error("expected error when removing non-existent module")
	}
}

func TestSearchModulesFunction(t *testing.T) {
	tmpDir := t.TempDir()
	catalogPath := filepath.Join(tmpDir, "catalog.yaml")

	gen := NewGenerator(catalogPath)

	gen.AddModule(Module{ID: "proto/test/v1", Name: "proto/test/v1", Format: "proto"})
	gen.AddModule(Module{ID: "openapi/test/v1", Name: "openapi/test/v1", Format: "openapi"})

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

func TestSearchModulesOpts(t *testing.T) {
	tmpDir := t.TempDir()
	catalogPath := filepath.Join(tmpDir, "catalog.yaml")

	cat := &Catalog{
		Version: 1,
		Org:     "testorg",
		Repo:    "apis",
		Modules: []Module{
			{ID: "proto/payments/ledger/v1", Format: "proto", Domain: "payments", APILine: "v1", Lifecycle: "stable", Description: "Ledger API"},
			{ID: "proto/payments/wallet/v1", Format: "proto", Domain: "payments", APILine: "v1", Lifecycle: "beta", Description: "Wallet API"},
			{ID: "openapi/customer/accounts/v2", Format: "openapi", Domain: "customer", APILine: "v2", Lifecycle: "stable", Description: "Accounts API"},
			{ID: "proto/customer/orders/v1", Format: "proto", Domain: "customer", APILine: "v1", Lifecycle: "deprecated", Description: "Orders API"},
		},
	}
	gen := NewGenerator(catalogPath)
	gen.Save(cat)

	tests := []struct {
		name     string
		opts     SearchOptions
		expected []string
	}{
		{
			name:     "filter by lifecycle stable",
			opts:     SearchOptions{Lifecycle: "stable"},
			expected: []string{"proto/payments/ledger/v1", "openapi/customer/accounts/v2"},
		},
		{
			name:     "filter by lifecycle beta",
			opts:     SearchOptions{Lifecycle: "beta"},
			expected: []string{"proto/payments/wallet/v1"},
		},
		{
			name:     "filter by domain payments",
			opts:     SearchOptions{Domain: "payments"},
			expected: []string{"proto/payments/ledger/v1", "proto/payments/wallet/v1"},
		},
		{
			name:     "filter by domain customer",
			opts:     SearchOptions{Domain: "customer"},
			expected: []string{"openapi/customer/accounts/v2", "proto/customer/orders/v1"},
		},
		{
			name:     "filter by api-line v2",
			opts:     SearchOptions{APILine: "v2"},
			expected: []string{"openapi/customer/accounts/v2"},
		},
		{
			name:     "filter by format and domain",
			opts:     SearchOptions{Format: "proto", Domain: "customer"},
			expected: []string{"proto/customer/orders/v1"},
		},
		{
			name:     "filter by lifecycle and domain",
			opts:     SearchOptions{Lifecycle: "stable", Domain: "payments"},
			expected: []string{"proto/payments/ledger/v1"},
		},
		{
			name:     "query searches domain too",
			opts:     SearchOptions{Query: "customer"},
			expected: []string{"openapi/customer/accounts/v2", "proto/customer/orders/v1"},
		},
		{
			name:     "query with format filter",
			opts:     SearchOptions{Query: "ledger", Format: "proto"},
			expected: []string{"proto/payments/ledger/v1"},
		},
		{
			name:     "no matches",
			opts:     SearchOptions{Domain: "nonexistent"},
			expected: []string{},
		},
		{
			name:     "case insensitive lifecycle",
			opts:     SearchOptions{Lifecycle: "STABLE"},
			expected: []string{"proto/payments/ledger/v1", "openapi/customer/accounts/v2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := SearchModulesOpts(gen, tt.opts)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(results) != len(tt.expected) {
				t.Fatalf("expected %d results, got %d", len(tt.expected), len(results))
			}
			found := make(map[string]bool)
			for _, r := range results {
				found[r.DisplayName()] = true
			}
			for _, e := range tt.expected {
				if !found[e] {
					t.Errorf("expected %s in results", e)
				}
			}
		})
	}
}

func TestDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		module   Module
		expected string
	}{
		{
			name:     "ID set",
			module:   Module{ID: "proto/payments/ledger/v1", Name: "legacy-name"},
			expected: "proto/payments/ledger/v1",
		},
		{
			name:     "ID empty, use Name",
			module:   Module{Name: "my-api"},
			expected: "my-api",
		},
		{
			name:     "both empty",
			module:   Module{},
			expected: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.module.DisplayName(); got != tt.expected {
				t.Errorf("DisplayName() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestDetectAPIIdentity(t *testing.T) {
	tests := []struct {
		name           string
		relPath        string
		expectID       string
		expectDomain   string
		expectLine     string
		expectDetected bool
	}{
		{
			name:           "standard proto path",
			relPath:        "proto/payments/ledger/v1/ledger.proto",
			expectID:       "proto/payments/ledger/v1",
			expectDomain:   "payments",
			expectLine:     "v1",
			expectDetected: true,
		},
		{
			name:           "openapi path",
			relPath:        "openapi/customer/accounts/v2/accounts.yaml",
			expectID:       "openapi/customer/accounts/v2",
			expectDomain:   "customer",
			expectLine:     "v2",
			expectDetected: true,
		},
		{
			name:           "avro path",
			relPath:        "avro/analytics/events/v1/event.avsc",
			expectID:       "avro/analytics/events/v1",
			expectDomain:   "analytics",
			expectLine:     "v1",
			expectDetected: true,
		},
		{
			name:           "deeper nested file",
			relPath:        "proto/payments/ledger/v1/sub/nested.proto",
			expectID:       "proto/payments/ledger/v1",
			expectDomain:   "payments",
			expectLine:     "v1",
			expectDetected: true,
		},
		{
			name:           "too few path parts",
			relPath:        "proto/payments/ledger.proto",
			expectID:       "",
			expectDomain:   "",
			expectLine:     "",
			expectDetected: false,
		},
		{
			name:           "unknown format",
			relPath:        "graphql/payments/ledger/v1/schema.graphql",
			expectID:       "",
			expectDomain:   "",
			expectLine:     "",
			expectDetected: false,
		},
		{
			name:           "v0 is not valid",
			relPath:        "proto/payments/ledger/v0/ledger.proto",
			expectID:       "",
			expectDomain:   "",
			expectLine:     "",
			expectDetected: false,
		},
		{
			name:           "non-version in line position",
			relPath:        "proto/payments/ledger/src/ledger.proto",
			expectID:       "",
			expectDomain:   "",
			expectLine:     "",
			expectDetected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, domain, line := detectAPIIdentity(tt.relPath)
			detected := id != ""
			if detected != tt.expectDetected {
				t.Fatalf("detected=%v, want %v (id=%q)", detected, tt.expectDetected, id)
			}
			if id != tt.expectID {
				t.Errorf("id=%q, want %q", id, tt.expectID)
			}
			if domain != tt.expectDomain {
				t.Errorf("domain=%q, want %q", domain, tt.expectDomain)
			}
			if line != tt.expectLine {
				t.Errorf("line=%q, want %q", line, tt.expectLine)
			}
		})
	}
}

func TestIsVersionLine(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"v1", true},
		{"v2", true},
		{"v10", true},
		{"v0", false},
		{"v", false},
		{"1", false},
		{"v01", false},
		{"src", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := isVersionLine(tt.input); got != tt.expected {
				t.Errorf("isVersionLine(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestScanDirectoryIdentity(t *testing.T) {
	tmpDir := t.TempDir()

	// Create proto/payments/ledger/v1/ledger.proto
	protoDir := filepath.Join(tmpDir, "proto", "payments", "ledger", "v1")
	if err := os.MkdirAll(protoDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(protoDir, "ledger.proto"), []byte("syntax = \"proto3\";"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create openapi/customer/accounts/v2/accounts.yaml
	openapiDir := filepath.Join(tmpDir, "openapi", "customer", "accounts", "v2")
	if err := os.MkdirAll(openapiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(openapiDir, "accounts.yaml"), []byte("openapi: 3.0.0"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a file in a non-standard location (should be found by name without identity)
	flatDir := filepath.Join(tmpDir, "misc")
	if err := os.MkdirAll(flatDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(flatDir, "readme.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	catalogPath := filepath.Join(tmpDir, "catalog.yaml")
	gen := NewGenerator(catalogPath)

	modules, err := gen.ScanDirectory(tmpDir)
	if err != nil {
		t.Fatalf("ScanDirectory failed: %v", err)
	}

	// Should have at least 2 identity-detected modules + 1 flat file
	if len(modules) < 2 {
		t.Fatalf("expected at least 2 modules, got %d", len(modules))
	}

	// Check identity-detected modules
	found := make(map[string]Module)
	for _, m := range modules {
		found[m.ID] = m
	}

	ledger, ok := found["proto/payments/ledger/v1"]
	if !ok {
		t.Fatal("expected module proto/payments/ledger/v1")
	}
	if ledger.Format != "proto" {
		t.Errorf("expected format proto, got %s", ledger.Format)
	}
	if ledger.Domain != "payments" {
		t.Errorf("expected domain payments, got %s", ledger.Domain)
	}
	if ledger.APILine != "v1" {
		t.Errorf("expected api-line v1, got %s", ledger.APILine)
	}

	accounts, ok := found["openapi/customer/accounts/v2"]
	if !ok {
		t.Fatal("expected module openapi/customer/accounts/v2")
	}
	if accounts.Format != "openapi" {
		t.Errorf("expected format openapi, got %s", accounts.Format)
	}
	if accounts.Domain != "customer" {
		t.Errorf("expected domain customer, got %s", accounts.Domain)
	}
}

func TestSearchModulesOpts_OriginFilter(t *testing.T) {
	tmpDir := t.TempDir()
	catalogPath := filepath.Join(tmpDir, "catalog.yaml")

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
			{
				ID:     "proto/google/pubsub/v1",
				Format: "proto",
				Domain: "google",
				Path:   "google/pubsub/v1",
				Origin: "external",
			},
			{
				ID:     "proto/google/api/v1",
				Format: "proto",
				Domain: "google",
				Path:   "google/api",
				Origin: "forked",
			},
		},
	}

	gen := NewGenerator(catalogPath)
	if err := gen.Save(cat); err != nil {
		t.Fatalf("failed to save test catalog: %v", err)
	}

	tests := []struct {
		name          string
		origin        string
		expectedCount int
		expectedIDs   []string
	}{
		{
			name:          "no filter returns all",
			origin:        "",
			expectedCount: 3,
			expectedIDs:   []string{"proto/payments/ledger/v1", "proto/google/pubsub/v1", "proto/google/api/v1"},
		},
		{
			name:          "first-party only",
			origin:        "first-party",
			expectedCount: 1,
			expectedIDs:   []string{"proto/payments/ledger/v1"},
		},
		{
			name:          "external only",
			origin:        "external",
			expectedCount: 1,
			expectedIDs:   []string{"proto/google/pubsub/v1"},
		},
		{
			name:          "forked only",
			origin:        "forked",
			expectedCount: 1,
			expectedIDs:   []string{"proto/google/api/v1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := SearchModulesOpts(gen, SearchOptions{Origin: tt.origin})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(results) != tt.expectedCount {
				t.Errorf("expected %d results, got %d", tt.expectedCount, len(results))
			}
			foundIDs := make(map[string]bool)
			for _, r := range results {
				foundIDs[r.ID] = true
			}
			for _, id := range tt.expectedIDs {
				if !foundIDs[id] {
					t.Errorf("expected module %s not found in results", id)
				}
			}
		})
	}
}

func TestSearchModulesOpts_TagFilter(t *testing.T) {
	tmpDir := t.TempDir()
	catalogPath := filepath.Join(tmpDir, "catalog.yaml")

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
				Tags:   []string{"public", "core"},
			},
			{
				ID:     "proto/internal/metrics/v1",
				Format: "proto",
				Domain: "infra",
				Path:   "proto/internal/metrics/v1",
				Tags:   []string{"internal"},
			},
			{
				ID:     "openapi/billing/invoices/v1",
				Format: "openapi",
				Domain: "billing",
				Path:   "openapi/billing/invoices/v1",
				Tags:   []string{"public"},
			},
		},
	}

	gen := NewGenerator(catalogPath)
	if err := gen.Save(cat); err != nil {
		t.Fatalf("failed to save test catalog: %v", err)
	}

	tests := []struct {
		name          string
		tag           string
		expectedCount int
		expectedIDs   []string
	}{
		{
			name:          "no filter returns all",
			tag:           "",
			expectedCount: 3,
		},
		{
			name:          "filter by public tag",
			tag:           "public",
			expectedCount: 2,
			expectedIDs:   []string{"proto/payments/ledger/v1", "openapi/billing/invoices/v1"},
		},
		{
			name:          "filter by internal tag",
			tag:           "internal",
			expectedCount: 1,
			expectedIDs:   []string{"proto/internal/metrics/v1"},
		},
		{
			name:          "filter by core tag",
			tag:           "core",
			expectedCount: 1,
			expectedIDs:   []string{"proto/payments/ledger/v1"},
		},
		{
			name:          "case insensitive tag match",
			tag:           "PUBLIC",
			expectedCount: 2,
			expectedIDs:   []string{"proto/payments/ledger/v1", "openapi/billing/invoices/v1"},
		},
		{
			name:          "nonexistent tag",
			tag:           "missing",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := SearchModulesOpts(gen, SearchOptions{Tag: tt.tag})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(results) != tt.expectedCount {
				t.Errorf("expected %d results, got %d", tt.expectedCount, len(results))
			}
			foundIDs := make(map[string]bool)
			for _, r := range results {
				foundIDs[r.ID] = true
			}
			for _, id := range tt.expectedIDs {
				if !foundIDs[id] {
					t.Errorf("expected module %s not found in results", id)
				}
			}
		})
	}
}

func TestSearchModulesOpts_FreeTextMatchesTags(t *testing.T) {
	tmpDir := t.TempDir()
	catalogPath := filepath.Join(tmpDir, "catalog.yaml")

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
				Tags:   []string{"flagship"},
			},
			{
				ID:     "proto/billing/invoices/v1",
				Format: "proto",
				Domain: "billing",
				Path:   "proto/billing/invoices/v1",
			},
		},
	}

	gen := NewGenerator(catalogPath)
	if err := gen.Save(cat); err != nil {
		t.Fatalf("failed to save test catalog: %v", err)
	}

	results, err := SearchModulesOpts(gen, SearchOptions{Query: "flagship"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ID != "proto/payments/ledger/v1" {
		t.Errorf("expected proto/payments/ledger/v1, got %s", results[0].ID)
	}
}

func TestIsRemoteURL(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"https://raw.githubusercontent.com/org/apis/main/catalog.yaml", true},
		{"http://example.com/catalog.yaml", true},
		{"catalog/catalog.yaml", false},
		{"/absolute/path/catalog.yaml", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isRemoteURL(tt.input)
			if got != tt.expected {
				t.Errorf("isRemoteURL(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

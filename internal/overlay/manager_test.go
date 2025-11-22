package overlay

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateOverlay(t *testing.T) {
	tmpDir := t.TempDir()
	workspacePath := tmpDir

	mgr := NewManager(workspacePath)

	tests := []struct {
		name        string
		modulePath  string
		language    string
		expectError bool
		expectDir   string
	}{
		{
			name:        "create go overlay",
			modulePath:  "proto/payments/ledger/v1",
			language:    "go",
			expectError: false,
			expectDir:   "internal/gen/go/proto/payments/ledger/v1",
		},
		{
			name:        "create python overlay",
			modulePath:  "proto/payments/wallet/v1",
			language:    "python",
			expectError: false,
			expectDir:   "internal/gen/python/proto/payments/wallet/v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			overlay, err := mgr.Create(tt.modulePath, tt.language)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectError {
				if overlay.Path == "" {
					t.Error("expected overlay path but got empty string")
				}

				// Verify directory created
				fullPath := filepath.Join(workspacePath, tt.expectDir)
				if _, err := os.Stat(fullPath); os.IsNotExist(err) {
					t.Errorf("expected directory %s to exist", fullPath)
				}
			}
		})
	}
}

func TestSyncOverlays(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Create overlays
	overlays := []struct {
		path     string
		language string
	}{
		{"proto/payments/ledger/v1", "go"},
		{"proto/payments/wallet/v1", "go"},
	}

	for _, o := range overlays {
		if _, err := mgr.Create(o.path, o.language); err != nil {
			t.Fatalf("failed to create overlay: %v", err)
		}
	}

	// Sync to go.work
	if err := mgr.Sync(); err != nil {
		t.Fatalf("failed to sync overlays: %v", err)
	}

	// Verify go.work exists
	goWorkPath := filepath.Join(tmpDir, "go.work")
	if _, err := os.Stat(goWorkPath); os.IsNotExist(err) {
		t.Fatal("expected go.work to exist")
	}

	// Read go.work and verify entries
	content, err := os.ReadFile(goWorkPath)
	if err != nil {
		t.Fatalf("failed to read go.work: %v", err)
	}

	contentStr := string(content)
	expectedEntries := []string{
		"./internal/gen/go/proto/payments/ledger/v1",
		"./internal/gen/go/proto/payments/wallet/v1",
	}

	for _, expected := range expectedEntries {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("expected go.work to contain %s", expected)
		}
	}
}

func TestRemoveOverlay(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Create overlay
	modulePath := "proto/payments/ledger/v1"
	overlay, err := mgr.Create(modulePath, "go")
	if err != nil {
		t.Fatalf("failed to create overlay: %v", err)
	}

	// Sync it
	if err := mgr.Sync(); err != nil {
		t.Fatalf("failed to sync: %v", err)
	}

	// Remove overlay
	if err := mgr.Remove(modulePath); err != nil {
		t.Fatalf("failed to remove overlay: %v", err)
	}

	// Verify directory removed
	if _, err := os.Stat(overlay.Path); !os.IsNotExist(err) {
		t.Error("expected overlay directory to be removed")
	}

	// Verify go.work updated
	goWorkPath := filepath.Join(tmpDir, "go.work")
	content, err := os.ReadFile(goWorkPath)
	if err != nil {
		t.Fatalf("failed to read go.work: %v", err)
	}

	if strings.Contains(string(content), modulePath) {
		t.Error("expected go.work to not contain removed overlay")
	}
}

func TestListOverlays(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Create multiple overlays
	overlays := []struct {
		path     string
		language string
	}{
		{"proto/payments/ledger/v1", "go"},
		{"proto/payments/wallet/v1", "go"},
		{"openapi/customer/accounts/v2", "go"},
	}

	for _, o := range overlays {
		if _, err := mgr.Create(o.path, o.language); err != nil {
			t.Fatalf("failed to create overlay: %v", err)
		}
	}

	// List all overlays
	list, err := mgr.List()
	if err != nil {
		t.Fatalf("failed to list overlays: %v", err)
	}

	if len(list) != 3 {
		t.Errorf("expected 3 overlays, got %d", len(list))
	}

	// Verify all are present
	found := make(map[string]bool)
	for _, overlay := range list {
		found[overlay.ModulePath] = true
	}

	for _, expected := range overlays {
		if !found[expected.path] {
			t.Errorf("expected overlay %s not found", expected.path)
		}
	}
}

func TestSyncIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Create overlay
	if _, err := mgr.Create("proto/test/api/v1", "go"); err != nil {
		t.Fatalf("failed to create overlay: %v", err)
	}

	// Sync twice
	if err := mgr.Sync(); err != nil {
		t.Fatalf("first sync failed: %v", err)
	}

	goWorkPath := filepath.Join(tmpDir, "go.work")
	firstContent, err := os.ReadFile(goWorkPath)
	if err != nil {
		t.Fatalf("failed to read go.work: %v", err)
	}

	if err := mgr.Sync(); err != nil {
		t.Fatalf("second sync failed: %v", err)
	}

	secondContent, err := os.ReadFile(goWorkPath)
	if err != nil {
		t.Fatalf("failed to read go.work after second sync: %v", err)
	}

	if string(firstContent) != string(secondContent) {
		t.Error("sync is not idempotent - go.work changed after second sync")
	}
}

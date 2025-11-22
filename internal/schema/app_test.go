package schema

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAppScaffold(t *testing.T) {
	tests := []struct {
		name       string
		modulePath string
		org        string
		wantErr    bool
	}{
		{
			name:       "valid proto module path",
			modulePath: "internal/apis/proto/payments/ledger/v1",
			org:        "myorg",
			wantErr:    false,
		},
		{
			name:       "valid openapi module path",
			modulePath: "internal/apis/openapi/inventory/v2",
			org:        "testorg",
			wantErr:    false,
		},
		{
			name:       "empty module path",
			modulePath: "",
			org:        "myorg",
			wantErr:    true,
		},
		{
			name:       "empty org",
			modulePath: "internal/apis/proto/test/v1",
			org:        "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			scaffolder := NewAppScaffolder(tt.modulePath, tt.org)
			err := scaffolder.Generate(tmpDir)

			if (err != nil) != tt.wantErr {
				t.Errorf("AppScaffolder.Generate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Verify module directory was created
			modulePath := filepath.Join(tmpDir, tt.modulePath)
			if _, err := os.Stat(modulePath); os.IsNotExist(err) {
				t.Errorf("Expected module directory %s to exist", tt.modulePath)
			}

			// Verify apx.yaml was created in module directory
			apxYamlPath := filepath.Join(modulePath, "apx.yaml")
			if _, err := os.Stat(apxYamlPath); os.IsNotExist(err) {
				t.Errorf("Expected apx.yaml at %s", apxYamlPath)
			}

			// Verify .gitignore was created at root
			gitignorePath := filepath.Join(tmpDir, ".gitignore")
			if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
				t.Errorf("Expected .gitignore at project root")
			}
		})
	}
}

func TestAppScaffoldProtoStructure(t *testing.T) {
	tmpDir := t.TempDir()

	modulePath := "internal/apis/proto/payments/ledger/v1"
	scaffolder := NewAppScaffolder(modulePath, "myorg")

	if err := scaffolder.Generate(tmpDir); err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Verify proto-specific structure
	moduleDir := filepath.Join(tmpDir, modulePath)

	// Should have example proto file
	exampleProto := filepath.Join(moduleDir, "ledger.proto")
	if _, err := os.Stat(exampleProto); os.IsNotExist(err) {
		t.Errorf("Expected example proto file at %s", exampleProto)
	}

	// Verify apx.yaml contains proto configuration
	apxYaml := filepath.Join(moduleDir, "apx.yaml")
	content, err := os.ReadFile(apxYaml)
	if err != nil {
		t.Fatalf("Failed to read apx.yaml: %v", err)
	}

	expectedConfig := []string{
		"kind: proto",
		"module: payments.ledger.v1",
		"org: myorg",
	}

	for _, expected := range expectedConfig {
		if !stringContains(string(content), expected) {
			t.Errorf("apx.yaml should contain %q, content:\n%s", expected, content)
		}
	}
}

func TestAppScaffoldOpenAPIStructure(t *testing.T) {
	tmpDir := t.TempDir()

	modulePath := "internal/apis/openapi/inventory/v2"
	scaffolder := NewAppScaffolder(modulePath, "testorg")

	if err := scaffolder.Generate(tmpDir); err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Verify openapi-specific structure
	moduleDir := filepath.Join(tmpDir, modulePath)

	// Should have example openapi spec
	exampleSpec := filepath.Join(moduleDir, "inventory.yaml")
	if _, err := os.Stat(exampleSpec); os.IsNotExist(err) {
		t.Errorf("Expected example OpenAPI spec at %s", exampleSpec)
	}

	// Verify apx.yaml contains openapi configuration
	apxYaml := filepath.Join(moduleDir, "apx.yaml")
	content, err := os.ReadFile(apxYaml)
	if err != nil {
		t.Fatalf("Failed to read apx.yaml: %v", err)
	}

	expectedConfig := []string{
		"kind: openapi",
		"module: inventory/v2",
		"org: testorg",
	}

	for _, expected := range expectedConfig {
		if !stringContains(string(content), expected) {
			t.Errorf("apx.yaml should contain %q, content:\n%s", expected, content)
		}
	}
}

func stringContains(s, substr string) bool {
	return len(s) >= len(substr) && containsSubstring(s, substr)
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

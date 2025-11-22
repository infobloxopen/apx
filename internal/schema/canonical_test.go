package schema

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCanonicalScaffold(t *testing.T) {
	tests := []struct {
		name    string
		org     string
		repo    string
		wantErr bool
	}{
		{
			name:    "valid org and repo",
			org:     "myorg",
			repo:    "apis",
			wantErr: false,
		},
		{
			name:    "empty org",
			org:     "",
			repo:    "apis",
			wantErr: true,
		},
		{
			name:    "empty repo",
			org:     "myorg",
			repo:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			scaffolder := NewCanonicalScaffolder(tt.org, tt.repo)
			err := scaffolder.Generate(tmpDir)

			if (err != nil) != tt.wantErr {
				t.Errorf("Generate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			// Verify expected directory structure
			expectedDirs := []string{
				"proto",
				"openapi",
				"avro",
				"jsonschema",
				"parquet",
			}

			for _, dir := range expectedDirs {
				dirPath := filepath.Join(tmpDir, dir)
				if _, err := os.Stat(dirPath); os.IsNotExist(err) {
					t.Errorf("Expected directory not created: %s", dir)
				}
			}

			// Verify expected files
			expectedFiles := []string{
				"buf.yaml",
				"CODEOWNERS",
				"catalog.yaml",
				"README.md",
			}

			for _, file := range expectedFiles {
				filePath := filepath.Join(tmpDir, file)
				if _, err := os.Stat(filePath); os.IsNotExist(err) {
					t.Errorf("Expected file not created: %s", file)
				}
			}
		})
	}
}

func TestBufYamlGeneration(t *testing.T) {
	tmpDir := t.TempDir()

	scaffolder := NewCanonicalScaffolder("myorg", "apis")
	err := scaffolder.Generate(tmpDir)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	bufPath := filepath.Join(tmpDir, "buf.yaml")
	content, err := os.ReadFile(bufPath)
	if err != nil {
		t.Fatalf("Failed to read buf.yaml: %v", err)
	}

	// Verify buf.yaml contains expected content
	contentStr := string(content)
	expectedStrings := []string{
		"version: v2",
		"modules:",
		"path: proto",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("buf.yaml missing expected content: %s", expected)
		}
	}
}

func TestCodeownersGeneration(t *testing.T) {
	tmpDir := t.TempDir()

	scaffolder := NewCanonicalScaffolder("myorg", "apis")
	err := scaffolder.Generate(tmpDir)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	codeownersPath := filepath.Join(tmpDir, "CODEOWNERS")
	content, err := os.ReadFile(codeownersPath)
	if err != nil {
		t.Fatalf("Failed to read CODEOWNERS: %v", err)
	}

	// Verify CODEOWNERS contains expected patterns
	contentStr := string(content)
	expectedPatterns := []string{
		"*",
		"@myorg/api-owners",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(contentStr, pattern) {
			t.Errorf("CODEOWNERS missing expected pattern: %s", pattern)
		}
	}
}

func TestCatalogGeneration(t *testing.T) {
	tmpDir := t.TempDir()

	scaffolder := NewCanonicalScaffolder("myorg", "apis")
	err := scaffolder.Generate(tmpDir)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	catalogPath := filepath.Join(tmpDir, "catalog.yaml")
	content, err := os.ReadFile(catalogPath)
	if err != nil {
		t.Fatalf("Failed to read catalog.yaml: %v", err)
	}

	// Verify catalog.yaml contains expected content
	contentStr := string(content)
	expectedStrings := []string{
		"version: 1",
		"org: myorg",
		"repo: apis",
		"modules: []",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("catalog.yaml missing expected content: %s", expected)
		}
	}
}

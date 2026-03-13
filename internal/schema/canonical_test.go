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

			scaffolder := NewCanonicalScaffolder(tt.org, tt.repo, "", "")
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
				"README.md",
			}

			for _, file := range expectedFiles {
				filePath := filepath.Join(tmpDir, file)
				if _, err := os.Stat(filePath); os.IsNotExist(err) {
					t.Errorf("Expected file not created: %s", file)
				}
			}

			// Verify catalog/.gitignore and catalog/Dockerfile were created
			catalogGitignore := filepath.Join(tmpDir, "catalog", ".gitignore")
			if _, err := os.Stat(catalogGitignore); os.IsNotExist(err) {
				t.Errorf("Expected file not created: catalog/.gitignore")
			}
			catalogDockerfile := filepath.Join(tmpDir, "catalog", "Dockerfile")
			if _, err := os.Stat(catalogDockerfile); os.IsNotExist(err) {
				t.Errorf("Expected file not created: catalog/Dockerfile")
			}
		})
	}
}

func TestBufYamlGeneration(t *testing.T) {
	tmpDir := t.TempDir()

	scaffolder := NewCanonicalScaffolder("myorg", "apis", "", "")
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

	scaffolder := NewCanonicalScaffolder("myorg", "apis", "", "")
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

func TestCanonicalScaffoldWithImportRoot(t *testing.T) {
	tmpDir := t.TempDir()

	scaffolder := NewCanonicalScaffolder("myorg", "apis", "go.myorg.dev/apis", "")
	err := scaffolder.Generate(tmpDir)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	apxYamlPath := filepath.Join(tmpDir, "apx.yaml")
	content, err := os.ReadFile(apxYamlPath)
	if err != nil {
		t.Fatalf("Failed to read apx.yaml: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "import_root: go.myorg.dev/apis") {
		t.Errorf("apx.yaml should contain import_root, content:\n%s", contentStr)
	}
}

func TestCatalogDockerfileGeneration(t *testing.T) {
	tmpDir := t.TempDir()

	scaffolder := NewCanonicalScaffolder("myorg", "apis", "", "")
	err := scaffolder.Generate(tmpDir)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Verify catalog/Dockerfile was created with OCI labels
	dockerfilePath := filepath.Join(tmpDir, "catalog", "Dockerfile")
	content, err := os.ReadFile(dockerfilePath)
	if err != nil {
		t.Fatalf("Failed to read catalog/Dockerfile: %v", err)
	}

	contentStr := string(content)
	expectedStrings := []string{
		"FROM scratch",
		"COPY catalog.yaml /catalog.yaml",
		"org.opencontainers.image.title",
		"org.opencontainers.image.vendor=\"myorg\"",
		"dev.apx.type=\"catalog\"",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("catalog/Dockerfile missing expected content: %s", expected)
		}
	}

	// Verify catalog/.gitignore was created
	gitignorePath := filepath.Join(tmpDir, "catalog", ".gitignore")
	gitignoreContent, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("Failed to read catalog/.gitignore: %v", err)
	}

	if !strings.Contains(string(gitignoreContent), "catalog.yaml") {
		t.Errorf("catalog/.gitignore should contain catalog.yaml")
	}
}

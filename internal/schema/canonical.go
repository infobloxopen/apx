package schema

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/infobloxopen/apx/internal/schema/templates"
)

// CanonicalScaffolder creates a canonical API repository structure
type CanonicalScaffolder struct {
	org  string
	repo string
}

// NewCanonicalScaffolder creates a new canonical scaffolder
func NewCanonicalScaffolder(org, repo string) *CanonicalScaffolder {
	return &CanonicalScaffolder{
		org:  org,
		repo: repo,
	}
}

// Generate creates the canonical repository structure
func (s *CanonicalScaffolder) Generate(targetDir string) error {
	if s.org == "" {
		return fmt.Errorf("org is required")
	}
	if s.repo == "" {
		return fmt.Errorf("repo is required")
	}

	// Create directory structure
	dirs := []string{
		"proto",
		"openapi",
		"avro",
		"jsonschema",
		"parquet",
		"catalog",
	}

	for _, dir := range dirs {
		dirPath := filepath.Join(targetDir, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}

		// Create .gitkeep file to ensure empty directories are tracked (skip for catalog as it will have catalog.yaml)
		if dir != "catalog" {
			gitkeepPath := filepath.Join(dirPath, ".gitkeep")
			if err := os.WriteFile(gitkeepPath, []byte(""), 0644); err != nil {
				return fmt.Errorf("failed to create .gitkeep in %s: %w", dir, err)
			}
		}
	}

	// Generate buf.yaml
	bufContent := templates.GenerateBufYaml()
	bufPath := filepath.Join(targetDir, "buf.yaml")
	if err := os.WriteFile(bufPath, []byte(bufContent), 0644); err != nil {
		return fmt.Errorf("failed to write buf.yaml: %w", err)
	}

	// Generate CODEOWNERS
	codeownersContent := templates.GenerateCodeowners(s.org)
	codeownersPath := filepath.Join(targetDir, "CODEOWNERS")
	if err := os.WriteFile(codeownersPath, []byte(codeownersContent), 0644); err != nil {
		return fmt.Errorf("failed to write CODEOWNERS: %w", err)
	}

	// Generate catalog.yaml in catalog/ directory
	catalogContent := templates.GenerateCatalog(s.org, s.repo)
	catalogPath := filepath.Join(targetDir, "catalog", "catalog.yaml")
	if err := os.WriteFile(catalogPath, []byte(catalogContent), 0644); err != nil {
		return fmt.Errorf("failed to write catalog.yaml: %w", err)
	}

	// Generate buf.work.yaml
	bufWorkContent := templates.GenerateBufWorkYaml()
	bufWorkPath := filepath.Join(targetDir, "buf.work.yaml")
	if err := os.WriteFile(bufWorkPath, []byte(bufWorkContent), 0644); err != nil {
		return fmt.Errorf("failed to write buf.work.yaml: %w", err)
	}

	// Generate README.md
	readmeContent := templates.GenerateReadme(s.org, s.repo)
	readmePath := filepath.Join(targetDir, "README.md")
	if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
		return fmt.Errorf("failed to write README.md: %w", err)
	}

	return nil
}

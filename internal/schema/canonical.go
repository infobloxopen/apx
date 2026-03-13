package schema

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/infobloxopen/apx/internal/config"
	"github.com/infobloxopen/apx/internal/schema/templates"
)

// CanonicalScaffolder creates a canonical API repository structure
type CanonicalScaffolder struct {
	org        string
	repo       string
	importRoot string
	siteURL    string
}

// NewCanonicalScaffolder creates a new canonical scaffolder
func NewCanonicalScaffolder(org, repo, importRoot, siteURL string) *CanonicalScaffolder {
	return &CanonicalScaffolder{
		org:        org,
		repo:       repo,
		importRoot: importRoot,
		siteURL:    siteURL,
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
		".github/workflows",
	}

	for _, dir := range dirs {
		dirPath := filepath.Join(targetDir, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}

		// Create .gitkeep file to ensure empty directories are tracked
		// (skip for catalog as it will have catalog.yaml, and .github/workflows will have workflow files)
		if dir != "catalog" && dir != ".github/workflows" {
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

	// Generate catalog/.gitignore so generated catalog data is not committed
	catalogGitignorePath := filepath.Join(targetDir, "catalog", ".gitignore")
	if err := os.WriteFile(catalogGitignorePath, []byte("catalog.yaml\n"), 0644); err != nil {
		return fmt.Errorf("failed to write catalog/.gitignore: %w", err)
	}

	// Generate catalog/Dockerfile for CI-based container builds
	dockerfilePath := filepath.Join(targetDir, "catalog", "Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte(templates.GenerateCatalogDockerfile(s.org)), 0644); err != nil {
		return fmt.Errorf("failed to write catalog/Dockerfile: %w", err)
	}

	// Generate README.md
	readmeContent := templates.GenerateReadme(s.org, s.repo)
	readmePath := filepath.Join(targetDir, "README.md")
	if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
		return fmt.Errorf("failed to write README.md: %w", err)
	}

	// Generate apx.yaml configuration (skip if already exists)
	apxYamlPath := filepath.Join(targetDir, "apx.yaml")
	if _, err := os.Stat(apxYamlPath); os.IsNotExist(err) {
		cfg := config.DefaultConfig()
		cfg.Org = s.org
		cfg.Repo = s.repo
		cfg.ImportRoot = s.importRoot
		cfg.SiteURL = s.siteURL
		content, err := config.MarshalConfigString(cfg)
		if err != nil {
			return fmt.Errorf("failed to generate apx.yaml: %w", err)
		}
		if err := os.WriteFile(apxYamlPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write apx.yaml: %w", err)
		}
	}

	// Generate CI workflow — always overwrite so upgrades pick up new templates
	ciPath := filepath.Join(targetDir, ".github", "workflows", "ci.yml")
	if err := os.WriteFile(ciPath, []byte(templates.GenerateCanonicalCI()), 0644); err != nil {
		return fmt.Errorf("failed to write ci.yml: %w", err)
	}

	// Generate on-merge workflow — always overwrite so upgrades pick up new templates
	onMergePath := filepath.Join(targetDir, ".github", "workflows", "on-merge.yml")
	if err := os.WriteFile(onMergePath, []byte(templates.GenerateCanonicalOnMerge(s.org)), 0644); err != nil {
		return fmt.Errorf("failed to write on-merge.yml: %w", err)
	}

	return nil
}

package catalog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Module represents a schema module in the catalog
type Module struct {
	Name        string   `yaml:"name"`
	Format      string   `yaml:"format"`
	Description string   `yaml:"description,omitempty"`
	Version     string   `yaml:"version"`
	Path        string   `yaml:"path"`
	Tags        []string `yaml:"tags,omitempty"`
	Owners      []string `yaml:"owners,omitempty"`
}

// Catalog represents the schema catalog
type Catalog struct {
	Version int      `yaml:"version"`
	Org     string   `yaml:"org"`
	Repo    string   `yaml:"repo"`
	Modules []Module `yaml:"modules"`
}

// Generator handles catalog generation
type Generator struct {
	catalogPath string
}

// NewGenerator creates a new catalog generator
func NewGenerator(catalogPath string) *Generator {
	if catalogPath == "" {
		catalogPath = "catalog.yaml"
	}
	return &Generator{
		catalogPath: catalogPath,
	}
}

// Load loads the existing catalog
func (g *Generator) Load() (*Catalog, error) {
	data, err := os.ReadFile(g.catalogPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty catalog
			return &Catalog{
				Version: 1,
				Modules: []Module{},
			}, nil
		}
		return nil, fmt.Errorf("failed to read catalog: %w", err)
	}

	var catalog Catalog
	if err := yaml.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("failed to parse catalog: %w", err)
	}

	return &catalog, nil
}

// Save saves the catalog to disk
func (g *Generator) Save(catalog *Catalog) error {
	data, err := yaml.Marshal(catalog)
	if err != nil {
		return fmt.Errorf("failed to marshal catalog: %w", err)
	}

	if err := os.WriteFile(g.catalogPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write catalog: %w", err)
	}

	return nil
}

// AddModule adds a module to the catalog
func (g *Generator) AddModule(module Module) error {
	catalog, err := g.Load()
	if err != nil {
		return err
	}

	// Check if module already exists
	for i, m := range catalog.Modules {
		if m.Name == module.Name {
			// Update existing module
			catalog.Modules[i] = module
			return g.Save(catalog)
		}
	}

	// Add new module
	catalog.Modules = append(catalog.Modules, module)
	return g.Save(catalog)
}

// RemoveModule removes a module from the catalog
func (g *Generator) RemoveModule(name string) error {
	catalog, err := g.Load()
	if err != nil {
		return err
	}

	for i, m := range catalog.Modules {
		if m.Name == name {
			catalog.Modules = append(catalog.Modules[:i], catalog.Modules[i+1:]...)
			return g.Save(catalog)
		}
	}

	return fmt.Errorf("module not found: %s", name)
}

// DetectFormat detects the schema format from file path
func DetectFormat(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	dir := filepath.Dir(path)

	switch ext {
	case ".proto":
		return "proto"
	case ".avsc", ".avdl", ".avpr":
		return "avro"
	case ".parquet":
		return "parquet"
	case ".yaml", ".yml", ".json":
		if strings.Contains(dir, "openapi") {
			return "openapi"
		}
		if strings.Contains(dir, "jsonschema") {
			return "jsonschema"
		}
		if strings.Contains(dir, "avro") {
			return "avro"
		}
		return "unknown"
	}

	return "unknown"
}

// ScanDirectory scans a directory for schema modules
func (g *Generator) ScanDirectory(dir string) ([]Module, error) {
	modules := []Module{}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		format := DetectFormat(path)
		if format == "unknown" {
			return nil
		}

		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		module := Module{
			Name:   filepath.Base(path),
			Format: format,
			Path:   relPath,
		}

		modules = append(modules, module)
		return nil
	})

	return modules, err
}

// GenerateCatalog generates a catalog from a directory
func (g *Generator) GenerateCatalog(dir, org, repo string) error {
	modules, err := g.ScanDirectory(dir)
	if err != nil {
		return fmt.Errorf("failed to scan directory: %w", err)
	}

	catalog := &Catalog{
		Version: 1,
		Org:     org,
		Repo:    repo,
		Modules: modules,
	}

	return g.Save(catalog)
}

// Search searches the catalog for modules matching a query
func (g *Generator) Search(query string) ([]Module, error) {
	catalog, err := g.Load()
	if err != nil {
		return nil, err
	}

	queryLower := strings.ToLower(query)
	matches := []Module{}

	for _, module := range catalog.Modules {
		if strings.Contains(strings.ToLower(module.Name), queryLower) ||
			strings.Contains(strings.ToLower(module.Description), queryLower) {
			matches = append(matches, module)
		}
	}

	return matches, nil
}

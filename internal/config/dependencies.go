package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Dependency represents a schema module dependency for the manager
type Dependency struct {
	ModulePath string
	Version    string
	Format     string
}

// DependencyManager handles adding/removing/listing dependencies
type DependencyManager struct {
	configPath string
	lockPath   string
}

// NewDependencyManager creates a new dependency manager
func NewDependencyManager(configPath, lockPath string) *DependencyManager {
	return &DependencyManager{
		configPath: configPath,
		lockPath:   lockPath,
	}
}

// Add adds a dependency to apx.yaml and apx.lock
func (dm *DependencyManager) Add(modulePath, version string) error {
	// If no version specified, use "latest" placeholder
	if version == "" {
		version = "latest"
	}

	// Update apx.yaml with dependency
	if err := dm.addToConfig(modulePath); err != nil {
		return fmt.Errorf("failed to update apx.yaml: %w", err)
	}

	// Load or create lock file
	lockFile, err := dm.loadLock()
	if err != nil {
		return fmt.Errorf("failed to load lock file: %w", err)
	}

	// Add/update dependency in the map
	lockFile.Dependencies[modulePath] = DependencyLock{
		Repo:    "github.com/org/apis", // TODO: Get from config
		Ref:     version,
		Modules: []string{modulePath},
	}

	// Save lock file
	if err := dm.saveLock(lockFile); err != nil {
		return fmt.Errorf("failed to save lock file: %w", err)
	}

	return nil
}

// Remove removes a dependency
func (dm *DependencyManager) Remove(modulePath string) error {
	lockFile, err := dm.loadLock()
	if err != nil {
		return fmt.Errorf("failed to load lock file: %w", err)
	}

	// Check if exists
	if _, exists := lockFile.Dependencies[modulePath]; !exists {
		return fmt.Errorf("dependency not found: %s", modulePath)
	}

	// Remove from map
	delete(lockFile.Dependencies, modulePath)

	return dm.saveLock(lockFile)
}

// List returns all dependencies
func (dm *DependencyManager) List() ([]Dependency, error) {
	lockFile, err := dm.loadLock()
	if err != nil {
		return nil, err
	}

	// Convert map to slice
	deps := []Dependency{}
	for modulePath, lock := range lockFile.Dependencies {
		deps = append(deps, Dependency{
			ModulePath: modulePath,
			Version:    lock.Ref,
		})
	}

	return deps, nil
}

// loadLock loads the lock file
func (dm *DependencyManager) loadLock() (*LockFile, error) {
	data, err := os.ReadFile(dm.lockPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty lock file
			return &LockFile{
				Version:      1,
				Toolchains:   make(map[string]ToolchainLock),
				Dependencies: make(map[string]DependencyLock),
			}, nil
		}
		return nil, err
	}

	var lockFile LockFile
	if err := yaml.Unmarshal(data, &lockFile); err != nil {
		return nil, err
	}

	// Ensure maps are initialized
	if lockFile.Dependencies == nil {
		lockFile.Dependencies = make(map[string]DependencyLock)
	}
	if lockFile.Toolchains == nil {
		lockFile.Toolchains = make(map[string]ToolchainLock)
	}

	return &lockFile, nil
}

// saveLock saves the lock file
func (dm *DependencyManager) saveLock(lockFile *LockFile) error {
	data, err := yaml.Marshal(lockFile)
	if err != nil {
		return err
	}

	return os.WriteFile(dm.lockPath, data, 0644)
}

// addToConfig adds a dependency to apx.yaml
func (dm *DependencyManager) addToConfig(modulePath string) error {
	// Read existing apx.yaml
	var appConfig map[string]interface{}
	data, err := os.ReadFile(dm.configPath)
	if err != nil {
		return fmt.Errorf("failed to read apx.yaml: %w", err)
	}

	if err := yaml.Unmarshal(data, &appConfig); err != nil {
		return fmt.Errorf("failed to parse apx.yaml: %w", err)
	}

	// Get or create dependencies list
	var dependencies []string
	if deps, ok := appConfig["dependencies"]; ok {
		if depsList, ok := deps.([]interface{}); ok {
			for _, d := range depsList {
				if str, ok := d.(string); ok {
					dependencies = append(dependencies, str)
				}
			}
		}
	}

	// Check if already exists
	for _, dep := range dependencies {
		if dep == modulePath {
			return nil // Already exists
		}
	}

	// Add new dependency
	dependencies = append(dependencies, modulePath)
	appConfig["dependencies"] = dependencies

	// Write back to file
	updatedData, err := yaml.Marshal(appConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal apx.yaml: %w", err)
	}

	return os.WriteFile(dm.configPath, updatedData, 0644)
}

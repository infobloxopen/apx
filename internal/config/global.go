package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// KnownOrg represents an organization the user has interacted with.
type KnownOrg struct {
	Name  string   `yaml:"name"`            // e.g. "infobloxopen"
	Repos []string `yaml:"repos,omitempty"` // canonical repos, e.g. ["apis"]
}

// GlobalConfig is the user-level apx config stored at ~/.config/apx/config.yaml.
// It tracks known organizations and their canonical API repositories so that
// catalog discovery works without per-repo configuration.
type GlobalConfig struct {
	Version    int        `yaml:"version"`
	DefaultOrg string     `yaml:"default_org,omitempty"`
	Orgs       []KnownOrg `yaml:"orgs,omitempty"`
}

// GlobalConfigPath returns the path to the global config file.
func GlobalConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	dir := filepath.Join(home, ".config", "apx")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("cannot create config directory: %w", err)
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// LoadGlobal reads the global config from ~/.config/apx/config.yaml.
// Returns a default empty config if the file does not exist.
func LoadGlobal() (*GlobalConfig, error) {
	p, err := GlobalConfigPath()
	if err != nil {
		return &GlobalConfig{Version: 1}, nil
	}

	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return &GlobalConfig{Version: 1}, nil
		}
		return nil, fmt.Errorf("failed to read global config: %w", err)
	}

	var cfg GlobalConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse global config: %w", err)
	}
	if cfg.Version == 0 {
		cfg.Version = 1
	}
	return &cfg, nil
}

// SaveGlobal writes the global config to ~/.config/apx/config.yaml.
func SaveGlobal(cfg *GlobalConfig) error {
	p, err := GlobalConfigPath()
	if err != nil {
		return err
	}
	if cfg.Version == 0 {
		cfg.Version = 1
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal global config: %w", err)
	}
	return os.WriteFile(p, data, 0644)
}

// AddOrg adds or updates an org entry in the global config.
// Repos are merged (deduplicated) with any existing entry.
func (g *GlobalConfig) AddOrg(name string, repos []string) {
	for i, org := range g.Orgs {
		if org.Name == name {
			g.Orgs[i].Repos = mergeStringSlice(org.Repos, repos)
			return
		}
	}
	g.Orgs = append(g.Orgs, KnownOrg{Name: name, Repos: repos})
}

// SetDefaultOrg sets the default org. If the org is not already known,
// it is added with no repos.
func (g *GlobalConfig) SetDefaultOrg(name string) {
	g.DefaultOrg = name
	// Ensure the org exists in the list
	for _, org := range g.Orgs {
		if org.Name == name {
			return
		}
	}
	g.Orgs = append(g.Orgs, KnownOrg{Name: name})
}

// KnownOrgNames returns the names of all known orgs.
func (g *GlobalConfig) KnownOrgNames() []string {
	names := make([]string, len(g.Orgs))
	for i, org := range g.Orgs {
		names[i] = org.Name
	}
	return names
}

// FindOrg returns the KnownOrg with the given name, or nil if not found.
func (g *GlobalConfig) FindOrg(name string) *KnownOrg {
	for i := range g.Orgs {
		if g.Orgs[i].Name == name {
			return &g.Orgs[i]
		}
	}
	return nil
}

func mergeStringSlice(existing, additions []string) []string {
	seen := make(map[string]bool, len(existing))
	for _, s := range existing {
		seen[s] = true
	}
	merged := make([]string, len(existing))
	copy(merged, existing)
	for _, s := range additions {
		if !seen[s] {
			merged = append(merged, s)
			seen[s] = true
		}
	}
	return merged
}

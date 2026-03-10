package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// MigrateResult holds the outcome of a migration operation.
type MigrateResult struct {
	Migrated    bool     `json:"migrated"`
	FromVersion int      `json:"from_version"`
	ToVersion   int      `json:"to_version"`
	Backup      string   `json:"backup,omitempty"`
	Changes     []Change `json:"changes"`
}

// MigrateFile reads an apx.yaml, determines its version, and applies the
// migration chain to bring it to CurrentSchemaVersion. It backs up the
// original file before writing changes.
func MigrateFile(path string) (*MigrateResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse to extract version
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	root := &doc
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		root = root.Content[0]
	}

	version, _, err := extractVersion(root)
	if err != nil {
		return nil, fmt.Errorf("cannot determine config version: %w", err)
	}

	// Already current
	if version == CurrentSchemaVersion {
		return &MigrateResult{
			Migrated:    false,
			FromVersion: version,
			ToVersion:   CurrentSchemaVersion,
		}, nil
	}

	// Future version — unsupported
	if version > CurrentSchemaVersion {
		return nil, fmt.Errorf("apx.yaml declares version %d, but this APX binary only supports up to version %d; upgrade APX to handle this configuration version", version, CurrentSchemaVersion)
	}

	// Version older than any known — unsupported
	if version < 1 {
		return nil, fmt.Errorf("unsupported schema version %d", version)
	}

	// Chain migrations from version → CurrentSchemaVersion
	current := data
	var allChanges []Change

	for v := version; v < CurrentSchemaVersion; v++ {
		migFn, ok := Registry.Migrations[v]
		if !ok {
			return nil, fmt.Errorf("no migration defined from version %d to %d", v, v+1)
		}
		migrated, changes, err := migFn(current)
		if err != nil {
			return nil, fmt.Errorf("migration from version %d to %d failed: %w", v, v+1, err)
		}
		current = migrated
		allChanges = append(allChanges, changes...)
	}

	// Backup original file
	backupPath, err := backupFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to back up config file: %w", err)
	}

	// Write migrated file
	if err := os.WriteFile(path, current, 0644); err != nil {
		return nil, fmt.Errorf("failed to write migrated config: %w", err)
	}

	return &MigrateResult{
		Migrated:    true,
		FromVersion: version,
		ToVersion:   CurrentSchemaVersion,
		Backup:      backupPath,
		Changes:     allChanges,
	}, nil
}

// backupFile copies the file at path to path.bak. If path.bak already exists,
// a timestamped suffix is appended (e.g., path.bak.20260307T143000).
func backupFile(path string) (string, error) {
	ext := ".bak"
	bakPath := path + ext

	if _, err := os.Stat(bakPath); err == nil {
		// .bak exists — use timestamp
		ts := time.Now().Format("20060102T150405")
		bakPath = path + ext + "." + ts
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(bakPath, data, 0644); err != nil {
		return "", err
	}

	// Return just the filename (relative from the directory of path)
	return filepath.Base(bakPath), nil
}

// MarshalConfig serializes a Config struct to YAML bytes.
func MarshalConfig(cfg *Config) ([]byte, error) {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	// yaml.Marshal produces valid YAML but we want a clean header comment
	return data, nil
}

// MarshalConfigString serializes a Config struct to a YAML string,
// applying basic formatting improvements.
func MarshalConfigString(cfg *Config) (string, error) {
	data, err := MarshalConfig(cfg)
	if err != nil {
		return "", err
	}

	// Add blank lines between top-level sections for readability
	lines := strings.Split(string(data), "\n")
	var result []string
	topLevelKeys := map[string]bool{
		"site_url:":         true,
		"module_roots:":     true,
		"language_targets:": true,
		"policy:":           true,
		"release:":          true,
		"tools:":            true,
		"execution:":        true,
	}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if i > 0 && topLevelKeys[trimmed] {
			result = append(result, "")
		}
		result = append(result, line)
	}

	return strings.Join(result, "\n"), nil
}

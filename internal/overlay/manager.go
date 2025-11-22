// Package overlay manages Go workspace overlays for canonical import paths.
//
// # Overview
//
// Overlays enable applications to use canonical import paths (e.g.,
// github.com/org/apis-go/proto/payments/ledger/v1) while transparently
// resolving them to locally generated code during development. This allows
// seamless transitions from local development to published module consumption
// without changing any import statements.
//
// # How It Works
//
//  1. Generated code is placed in versioned overlay directories:
//     internal/gen/go/proto/payments/ledger@v1.2.3/
//
//  2. Each overlay contains a go.mod with the canonical module path:
//     module github.com/org/apis-go/proto/payments/ledger
//
//  3. The workspace's go.work file maps canonical paths to local overlays:
//     use ./internal/gen/go/proto/payments/ledger@v1.2.3
//
//  4. Application code imports canonical paths:
//     import ledgerv1 "github.com/org/apis-go/proto/payments/ledger/v1"
//
// 5. Go resolves imports to local overlays during development via go.work
//
//  6. When ready, remove overlay and fetch published module:
//     apx unlink proto/payments/ledger/v1
//     go get github.com/org/apis-go/proto/payments/ledger@v1.2.3
//
// Same imports now resolve to published module - zero code changes!
//
// # Directory Structure
//
//	your-service/
//	├── go.mod                                    # module github.com/company/service
//	├── go.work                                   # managed by APX
//	├── internal/
//	│   ├── gen/                                  # all generated code (git-ignored)
//	│   │   ├── go/proto/payments/ledger@v1.2.3/ # Go overlay (no lang subdir)
//	│   │   │   ├── go.mod                       # canonical module path
//	│   │   │   └── v1/*.pb.go                   # generated code
//	│   │   ├── python/proto/payments/ledger/    # Python overlay (lang subdir)
//	│   │   └── java/proto/users/profile/        # Java overlay (lang subdir)
//	│   └── service/
//	│       └── payment_service.go               # app code with canonical imports
//	└── main.go
//
// # Overlay Lifecycle
//
// Create: apx gen go → generates code into internal/gen/go/{path}@{version}/
// Sync:   apx sync   → updates go.work with all Go overlay paths
// Remove: apx unlink → deletes overlay, regenerates go.work
//
// # Language-Specific Behavior
//
// - Go overlays:    internal/gen/go/{modulePath}@{version}/
// - Other languages: internal/gen/{language}/{modulePath}/
//
// Only Go overlays are added to go.work. Other languages use their own
// resolution mechanisms (Python PYTHONPATH, Java classpath, etc.)
//
// See /specs/001-align-docs-experience/overlays.md for detailed design documentation.
package overlay

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Overlay represents a generated code overlay that shadows a canonical module path.
//
// During development, overlays allow applications to import canonical paths
// (e.g., github.com/org/apis-go/proto/payments/ledger/v1) which resolve to
// locally generated code via go.work mappings. When ready to consume the
// published module, the overlay is removed and the application fetches the
// real module - without changing any imports.
//
// Example:
//
//	overlay := Overlay{
//	    ModulePath: "proto/payments/ledger/v1",      // schema path in canonical repo
//	    Language:   "go",                             // target language
//	    Path:       "/path/to/internal/gen/go/proto/payments/ledger@v1.2.3",
//	}
type Overlay struct {
	// ModulePath is the schema's path in the canonical repository.
	// Examples: "proto/payments/ledger/v1", "openapi/users/profile/v2"
	ModulePath string

	// Language is the target programming language.
	// Examples: "go", "python", "java"
	Language string

	// Path is the absolute path to the overlay directory containing generated code.
	// For Go: internal/gen/go/{ModulePath}@{version}/
	// For others: internal/gen/{Language}/{ModulePath}/
	Path string
}

// Manager handles go.work overlay creation and management.
//
// The Manager orchestrates the overlay lifecycle:
//   - Creating overlay directories for generated code
//   - Maintaining go.work file with overlay mappings
//   - Listing active overlays
//   - Removing overlays when transitioning to published modules
//
// All overlays are stored under internal/gen/ with language-specific
// subdirectories. The Manager ensures go.work stays synchronized with
// the actual overlay state on disk.
type Manager struct {
	workspaceRoot string // absolute path to workspace root
	overlayDir    string // absolute path to internal/gen/
}

// NewManager creates a new overlay manager for the given workspace.
//
// The workspace root should be the directory containing go.mod and go.work.
// The overlay directory will be set to {workspaceRoot}/internal/gen/.
//
// Example:
//
//	mgr := overlay.NewManager("/path/to/my-service")
//	// overlayDir will be /path/to/my-service/internal/gen/
func NewManager(workspaceRoot string) *Manager {
	return &Manager{
		workspaceRoot: workspaceRoot,
		overlayDir:    filepath.Join(workspaceRoot, "internal", "gen"),
	}
}

// Create creates a new overlay for a module.
//
// Directory structure differs by language:
//   - Go:     internal/gen/go/{modulePath}/  (e.g., internal/gen/go/proto/payments/ledger@v1.2.3/)
//   - Python: internal/gen/python/{modulePath}/  (e.g., internal/gen/python/proto/payments/ledger/)
//   - Java:   internal/gen/java/{modulePath}/    (e.g., internal/gen/java/proto/payments/ledger/)
//
// The overlay directory is created if it doesn't exist. After creating an overlay,
// you should call Sync() to update go.work (for Go overlays).
//
// Example:
//
//	overlay, err := mgr.Create("proto/payments/ledger/v1", "go")
//	if err != nil {
//	    return err
//	}
//	// Generate code into overlay.Path
//	// Then call mgr.Sync() to update go.work
func (m *Manager) Create(modulePath, language string) (*Overlay, error) {
	// All languages get a language-specific subdirectory
	overlayPath := filepath.Join(m.overlayDir, language, modulePath)

	if err := os.MkdirAll(overlayPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create overlay directory: %w", err)
	}

	overlay := &Overlay{
		ModulePath: modulePath,
		Language:   language,
		Path:       overlayPath,
	}

	return overlay, nil
}

// Sync synchronizes overlays with go.work file.
//
// This scans the internal/gen/ directory tree, identifies all overlays,
// and regenerates go.work to include Go overlays. Call this after:
//   - Creating new overlays (Create)
//   - Removing overlays (Remove)
//   - Updating dependency versions
//
// The operation is idempotent - safe to call multiple times.
//
// Example:
//
//	mgr.Create("proto/payments/ledger/v1", "go")
//	mgr.Sync()  // updates go.work
func (m *Manager) Sync() error {
	return m.SyncWorkFile()
}

// Remove removes an overlay and updates go.work.
//
// This deletes the overlay directory and all generated code within it,
// then regenerates go.work to exclude the removed overlay. Use this when
// transitioning from local development to published module consumption.
//
// After removing the overlay, fetch the published module:
//
//	mgr.Remove("proto/payments/ledger/v1")
//	// Then run: go get github.com/org/apis-go/proto/payments/ledger@v1.2.3
//
// The modulePath should match the schema path in the canonical repository,
// not the full overlay path. This removes all language variants of the overlay
// (Go, Python, Java, etc.) by removing the entire modulePath from all language
// directories.
func (m *Manager) Remove(modulePath string) error {
	// List all overlays and remove matching ones
	overlays, err := m.List()
	if err != nil {
		return fmt.Errorf("failed to list overlays: %w", err)
	}

	removed := false
	for _, overlay := range overlays {
		if overlay.ModulePath == modulePath {
			if err := os.RemoveAll(overlay.Path); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to remove overlay: %w", err)
			}
			removed = true
		}
	}

	if !removed {
		// Try to remove from each language directory in case List() missed it
		for _, lang := range []string{"go", "python", "java"} {
			overlayPath := filepath.Join(m.overlayDir, lang, modulePath)
			if err := os.RemoveAll(overlayPath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to remove overlay: %w", err)
			}
		}
	}

	return m.Sync()
}

// List returns all overlays in the workspace.
//
// This scans the internal/gen/ directory tree and identifies actual overlay
// directories (as opposed to intermediate path directories). An overlay is
// identified as:
//  1. A language-specific directory (ends with /python, /java, etc.), OR
//  2. A leaf directory with no subdirectories (except language subdirs)
//
// List returns all overlays in the workspace.
//
// This scans the internal/gen/ directory tree and identifies actual overlay
// directories. The structure is:
//
//	internal/gen/{language}/{modulePath}/
//
// For example:
//
//	internal/gen/
//	├── go/proto/payments/ledger@v1.2.3/  (IS an overlay)
//	├── python/proto/payments/ledger/      (IS an overlay)
//	└── java/proto/users/profile/          (IS an overlay)
//
// This is used internally by Sync() to rebuild go.work and by external
// callers to enumerate active overlays.
func (m *Manager) List() ([]Overlay, error) {
	var overlays []Overlay

	if _, err := os.Stat(m.overlayDir); os.IsNotExist(err) {
		return overlays, nil
	}

	// Walk each language directory
	languageDirs, err := os.ReadDir(m.overlayDir)
	if err != nil {
		return nil, err
	}

	for _, langDir := range languageDirs {
		if !langDir.IsDir() {
			continue
		}

		language := langDir.Name()
		langPath := filepath.Join(m.overlayDir, language)

		// Walk the language directory to find module overlays
		err := filepath.Walk(langPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.IsDir() || path == langPath {
				return nil
			}

			relPath, err := filepath.Rel(langPath, path)
			if err != nil {
				return err
			}
			
			// Normalize to forward slashes for cross-platform consistency
			relPath = filepath.ToSlash(relPath)

			// Check if this directory has subdirectories
			entries, err := os.ReadDir(path)
			if err != nil {
				return err
			}

			hasSubdirs := false
			for _, entry := range entries {
				if entry.IsDir() {
					hasSubdirs = true
					break
				}
			}

			// This is an overlay if it's a leaf directory (no subdirectories)
			if !hasSubdirs {
				overlays = append(overlays, Overlay{
					ModulePath: relPath,
					Language:   language,
					Path:       path,
				})
			}

			return nil
		})

		if err != nil {
			return nil, err
		}
	}

	return overlays, nil
}

// CreateOverlay creates a go.work overlay for a module
func (m *Manager) CreateOverlay(canonicalImportPath, localPath string) error {
	// Ensure overlay directory exists
	if err := os.MkdirAll(m.overlayDir, 0755); err != nil {
		return fmt.Errorf("failed to create overlay directory: %w", err)
	}

	// Create symbolic link or copy module to overlay
	targetPath := filepath.Join(m.overlayDir, canonicalImportPath)
	targetDir := filepath.Dir(targetPath)

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Create symlink
	if err := os.Symlink(localPath, targetPath); err != nil {
		if !os.IsExist(err) {
			return fmt.Errorf("failed to create symlink: %w", err)
		}
	}

	return nil
}

// RemoveOverlay removes an overlay for a module
func (m *Manager) RemoveOverlay(canonicalImportPath string) error {
	targetPath := filepath.Join(m.overlayDir, canonicalImportPath)

	if err := os.Remove(targetPath); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove overlay: %w", err)
		}
	}

	return nil
}

// SyncWorkFile updates go.work to include all Go overlays.
//
// This regenerates the entire go.work file with:
//   - Go version directive (go 1.24)
//   - Application module (use .)
//   - All Go overlays (use ./internal/gen/go/...)
//
// The operation is idempotent and safe to call multiple times. Only Go
// overlays are included - Python and Java use different resolution mechanisms.
//
// Generated go.work structure:
//
//	go 1.24
//
//	use (
//	    .                                             // your app module
//	    ./internal/gen/go/proto/payments/ledger@v1.2.3
//	    ./internal/gen/go/proto/users/profile@v1.0.1
//	)
//
// This file should generally be git-ignored as it's development-specific.
// CI environments regenerate it from apx.lock via 'apx gen go && apx sync'.
func (m *Manager) SyncWorkFile() error {
	workFilePath := filepath.Join(m.workspaceRoot, "go.work")

	// List all overlays
	overlays, err := m.List()
	if err != nil {
		return fmt.Errorf("failed to list overlays: %w", err)
	}

	// Build go.work content
	var content strings.Builder
	content.WriteString("go 1.24\n\n")
	content.WriteString("use (\n")
	content.WriteString("\t.\n")

	for _, overlay := range overlays {
		if overlay.Language == "go" {
			relPath, err := filepath.Rel(m.workspaceRoot, overlay.Path)
			if err != nil {
				return fmt.Errorf("failed to get relative path: %w", err)
			}
			// Always use forward slashes in go.work (even on Windows)
			relPath = filepath.ToSlash(relPath)
			content.WriteString(fmt.Sprintf("\t./%s\n", relPath))
		}
	}

	content.WriteString(")\n")

	// Write updated go.work
	if err := os.WriteFile(workFilePath, []byte(content.String()), 0644); err != nil {
		return fmt.Errorf("failed to write go.work: %w", err)
	}

	return nil
}

// ListOverlays lists all active overlays
func (m *Manager) ListOverlays() ([]string, error) {
	overlays := []string{}

	if _, err := os.Stat(m.overlayDir); os.IsNotExist(err) {
		return overlays, nil
	}

	err := filepath.Walk(m.overlayDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.Mode()&os.ModeSymlink != 0 {
			relPath, err := filepath.Rel(m.overlayDir, path)
			if err != nil {
				return err
			}
			overlays = append(overlays, relPath)
		}

		return nil
	})

	return overlays, err
}

// CleanOverlays removes all overlays
func (m *Manager) CleanOverlays() error {
	if _, err := os.Stat(m.overlayDir); os.IsNotExist(err) {
		return nil
	}

	if err := os.RemoveAll(m.overlayDir); err != nil {
		return fmt.Errorf("failed to clean overlays: %w", err)
	}

	return nil
}

// TransitionToPublished transitions from overlay to published module
func (m *Manager) TransitionToPublished(canonicalImportPath, publishedVersion string) error {
	// Remove overlay
	if err := m.RemoveOverlay(canonicalImportPath); err != nil {
		return fmt.Errorf("failed to remove overlay: %w", err)
	}

	// Update go.work to remove overlay reference
	if err := m.SyncWorkFile(); err != nil {
		return fmt.Errorf("failed to sync go.work: %w", err)
	}

	return nil
}

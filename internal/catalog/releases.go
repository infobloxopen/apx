package catalog

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/mod/semver"
	"gopkg.in/yaml.v3"
)

// ReleaseManifest represents a release manifest file in releases/<id>/<version>.yaml.
type ReleaseManifest struct {
	ID         string `yaml:"id"`
	Version    string `yaml:"version"`
	Format     string `yaml:"format"`
	SourceRepo string `yaml:"source_repo"`
	SourceRef  string `yaml:"source_ref"`
	SourcePath string `yaml:"source_path"`
	ImportMode string `yaml:"import_mode"`
	ReleasedAt string `yaml:"released_at"`
}

// GenerateFromReleases builds a Catalog by scanning release manifest files
// under the given directory (typically "releases/").
// Each manifest contributes a module version; the latest version per API ID
// becomes the catalog entry.
func GenerateFromReleases(releasesDir string, org, repo string) (*Catalog, error) {
	apis := make(map[string]*releaseAccum)

	err := filepath.Walk(releasesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".yaml" && filepath.Ext(path) != ".yml" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}

		var manifest ReleaseManifest
		if err := yaml.Unmarshal(data, &manifest); err != nil {
			return fmt.Errorf("parsing %s: %w", path, err)
		}

		if manifest.ID == "" || manifest.Version == "" {
			return fmt.Errorf("%s: missing id or version", path)
		}

		// Ensure version has v prefix for semver comparison
		version := manifest.Version
		if !strings.HasPrefix(version, "v") {
			version = "v" + version
		}
		if !semver.IsValid(version) {
			return fmt.Errorf("%s: invalid semver version %q", path, manifest.Version)
		}

		acc, ok := apis[manifest.ID]
		if !ok {
			parts := strings.Split(manifest.ID, "/")
			acc = &releaseAccum{
				Format:     manifest.Format,
				SourceRepo: manifest.SourceRepo,
				SourcePath: manifest.SourcePath,
				ImportMode: manifest.ImportMode,
			}
			if len(parts) >= 4 {
				acc.Domain = parts[1]
				acc.APILine = parts[3]
			}
			apis[manifest.ID] = acc
		}

		if isStableVersion(version) {
			if acc.LatestStable == "" || semver.Compare(version, acc.LatestStable) > 0 {
				acc.LatestStable = version
			}
		} else {
			if acc.LatestPrerelease == "" || semver.Compare(version, acc.LatestPrerelease) > 0 {
				acc.LatestPrerelease = version
			}
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scanning releases: %w", err)
	}

	// Build sorted module list
	ids := make([]string, 0, len(apis))
	for id := range apis {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	modules := make([]Module, 0, len(ids))
	for _, id := range ids {
		acc := apis[id]
		m := Module{
			ID:          id,
			Name:        id,
			Format:      acc.Format,
			Domain:      acc.Domain,
			APILine:     acc.APILine,
			Path:        acc.SourcePath,
			Origin:      "sourced",
			ManagedRepo: acc.SourceRepo,
			ImportMode:  acc.ImportMode,
		}

		if acc.LatestStable != "" {
			m.LatestStable = acc.LatestStable
			m.Version = acc.LatestStable
			m.Lifecycle = "stable"
		}
		if acc.LatestPrerelease != "" {
			m.LatestPrerelease = acc.LatestPrerelease
			if m.Version == "" {
				m.Version = acc.LatestPrerelease
			}
		}
		if m.Lifecycle == "" && m.LatestPrerelease != "" {
			pre := strings.ToLower(semver.Prerelease(acc.LatestPrerelease))
			switch {
			case strings.HasPrefix(pre, "-alpha"):
				m.Lifecycle = "experimental"
			case strings.HasPrefix(pre, "-beta"), strings.HasPrefix(pre, "-rc"):
				m.Lifecycle = "beta"
			default:
				m.Lifecycle = "experimental"
			}
		}

		modules = append(modules, m)
	}

	return &Catalog{
		Version: 1,
		Org:     org,
		Repo:    repo,
		Modules: modules,
	}, nil
}

type releaseAccum struct {
	Format           string
	Domain           string
	APILine          string
	SourceRepo       string
	SourcePath       string
	ImportMode       string
	LatestStable     string
	LatestPrerelease string
}

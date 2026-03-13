package catalog

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/infobloxopen/apx/internal/config"
)

// FetchRemoteTags fetches git tags from a remote repository without cloning.
// It uses `git ls-remote --tags` which is fast and works with both public
// repos and private repos where git credentials are configured.
func FetchRemoteTags(repoURL string) ([]string, error) {
	url := repoURL
	if !strings.Contains(url, "://") {
		url = "https://" + url + ".git"
	}

	cmd := exec.Command("git", "ls-remote", "--tags", "--refs", url)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-remote %s: %w", repoURL, err)
	}

	raw := strings.TrimSpace(string(output))
	if raw == "" {
		return nil, nil
	}

	var tags []string
	for _, line := range strings.Split(raw, "\n") {
		// Format: "<sha>\trefs/tags/<tagname>"
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		ref := parts[1]
		tag := strings.TrimPrefix(ref, "refs/tags/")
		tags = append(tags, tag)
	}

	return tags, nil
}

// MergeAPISources fetches tags from each configured API source repository,
// parses them using the standard release tag pattern, and merges the
// resulting modules into the catalog.
//
// Modules from API sources have origin="sourced" and carry provenance
// metadata (managed_repo, source path, import_mode).
func MergeAPISources(cat *Catalog, sources []config.APISource) error {
	if len(sources) == 0 {
		return nil
	}

	// Build index of existing module IDs to detect conflicts
	existingIDs := make(map[string]bool)
	for _, m := range cat.Modules {
		existingIDs[m.ID] = true
	}

	for _, src := range sources {
		if err := src.Validate(); err != nil {
			return fmt.Errorf("invalid api_source %q: %w", src.Repo, err)
		}

		tags, err := FetchRemoteTags(src.Repo)
		if err != nil {
			return fmt.Errorf("fetching tags from %s: %w", src.Repo, err)
		}

		// Parse tags using the standard pattern
		remoteCat := GenerateFromTags(tags, "", "")

		for _, m := range remoteCat.Modules {
			// Check for conflicts with local or previously-added modules
			if existingIDs[m.ID] {
				return fmt.Errorf("api_source %q: module %q conflicts with existing module", src.Repo, m.ID)
			}

			// Apply source provenance
			m.Origin = config.OriginSourced
			m.ManagedRepo = src.Repo
			m.ImportMode = src.ImportMode

			// Override path from path_map if configured
			m.Path = src.SourcePathFor(m.ID)

			existingIDs[m.ID] = true
			cat.Modules = append(cat.Modules, m)
		}
	}

	return nil
}

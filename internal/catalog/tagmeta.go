package catalog

import (
	"os/exec"
	"sort"
	"strings"
)

// TagRecord is a release git tag paired with the metadata recorded in its
// annotation. `apx release finalize` writes an annotated tag whose body carries
// the lifecycle the release was cut under and any first-party catalog tags:
//
//	Release <api-id> <version>
//
//	Lifecycle: deprecated
//	Tags: audience:device, product:universal-ddi
//	Source: github.com/org/repo/openapi/...
//
// Reading it back at `apx catalog generate` is what lets a module released
// `--lifecycle deprecated` surface as deprecated in the generated catalog rather
// than being re-derived as `stable` from its semver alone (WS-035 F-32), and is
// the carrier that round-trips first-party tags through release → generate → show
// (WS-035 F-33).
type TagRecord struct {
	Tag       string
	Lifecycle string   // recorded lifecycle from the annotation ("" if none/lightweight)
	Tags      []string // recorded catalog tags from the annotation
}

// ReadTagRecords lists the repo's tags and, for each, reads the metadata
// recorded in its annotation body. Lightweight (non-annotated) tags yield an
// empty body and therefore no metadata. A single `git for-each-ref` call is used
// so this stays cheap even for a large tag set.
func ReadTagRecords(repoDir string) ([]TagRecord, error) {
	// %(refname:strip=2) is the short tag name; %(contents:body) is the
	// annotation body (empty for lightweight tags). Fields are separated by \x1f
	// and records by \x1e so multi-line bodies parse unambiguously.
	cmd := exec.Command("git", "for-each-ref",
		"--format=%(refname:strip=2)\x1f%(contents:body)\x1e", "refs/tags/")
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var records []TagRecord
	for _, rec := range strings.Split(string(out), "\x1e") {
		rec = strings.Trim(rec, "\n")
		if rec == "" {
			continue
		}
		name, body, _ := strings.Cut(rec, "\x1f")
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		lifecycle, tags := parseAnnotationMeta(body)
		records = append(records, TagRecord{Tag: name, Lifecycle: lifecycle, Tags: tags})
	}
	return records, nil
}

// parseAnnotationMeta extracts the "Lifecycle:" and "Tags:" fields from an
// annotated-tag body. Unknown lines are ignored.
func parseAnnotationMeta(body string) (lifecycle string, tags []string) {
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "Lifecycle:"):
			lifecycle = strings.TrimSpace(strings.TrimPrefix(line, "Lifecycle:"))
		case strings.HasPrefix(line, "Tags:"):
			tags = SplitTags(strings.TrimPrefix(line, "Tags:"))
		}
	}
	return lifecycle, tags
}

// SplitTags parses a comma-separated tag list into a trimmed, non-empty slice.
func SplitTags(s string) []string {
	var out []string
	for _, t := range strings.Split(s, ",") {
		if t = strings.TrimSpace(t); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// UnionTags returns the sorted, de-duplicated union of two tag lists. It is used
// so a re-release or regeneration never drops a curated tag.
func UnionTags(a, b []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, t := range append(append([]string{}, a...), b...) {
		if t = strings.TrimSpace(t); t != "" && !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	sort.Strings(out)
	return out
}

// FormatTagList renders a tag slice as a comma-separated string for an
// annotation body. Returns "" for an empty slice.
func FormatTagList(tags []string) string {
	return strings.Join(tags, ", ")
}

// lifecycleRank orders lifecycle states from earliest to latest. Lifecycle only
// moves forward, so a higher rank is "further along". Unknown states rank 0.
func lifecycleRank(lc string) int {
	switch lc {
	case "experimental":
		return 1
	case "beta":
		return 2
	case "stable":
		return 3
	case "deprecated":
		return 4
	case "sunset":
		return 5
	default:
		return 0
	}
}

// PreserveCuratedFields reconciles a freshly tag-derived catalog with the
// existing committed catalog at path. For each module present in both, it keeps
// the facts that git tags cannot express and that a regeneration would otherwise
// discard:
//
//   - Lifecycle: the further-along of the two (by lifecycleRank). This is what
//     lets `apx release promote --to deprecated` deprecate a module IN PLACE —
//     writing the state to the catalog without minting a new version — and have
//     it survive the next `apx catalog generate` (WS-035 F-32).
//   - Tags: the union, so curated first-party tags are never dropped (F-33).
//   - Owners / Description: taken from the existing catalog when the tag-derived
//     module lacks them.
//
// A missing or unreadable existing catalog is a no-op (first generation).
func PreserveCuratedFields(cat *Catalog, path string) {
	if cat == nil {
		return
	}
	existing, err := SourceFor(path).Load()
	if err != nil || existing == nil {
		return
	}
	byID := make(map[string]Module, len(existing.Modules))
	for _, m := range existing.Modules {
		byID[m.DisplayName()] = m
	}
	for i := range cat.Modules {
		prev, ok := byID[cat.Modules[i].DisplayName()]
		if !ok {
			continue
		}
		if lifecycleRank(prev.Lifecycle) > lifecycleRank(cat.Modules[i].Lifecycle) {
			cat.Modules[i].Lifecycle = prev.Lifecycle
		}
		cat.Modules[i].Tags = UnionTags(cat.Modules[i].Tags, prev.Tags)
		if len(cat.Modules[i].Owners) == 0 {
			cat.Modules[i].Owners = prev.Owners
		}
		if cat.Modules[i].Description == "" {
			cat.Modules[i].Description = prev.Description
		}
	}
}

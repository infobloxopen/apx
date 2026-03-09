package language

import (
	"fmt"
	"strings"

	"github.com/infobloxopen/apx/internal/config"
)

// FormatIdentityReport produces a human-readable multi-line report
// of an API's canonical identity information, using registered plugins
// for the language-specific section.
func FormatIdentityReport(api *config.APIIdentity, source *config.SourceIdentity, release *config.ReleaseInfo, langs map[string]config.LanguageCoords) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("API:        %s\n", api.ID))
	sb.WriteString(fmt.Sprintf("Format:     %s\n", api.Format))
	sb.WriteString(fmt.Sprintf("Domain:     %s\n", api.Domain))
	sb.WriteString(fmt.Sprintf("Name:       %s\n", api.Name))
	sb.WriteString(fmt.Sprintf("Line:       %s\n", api.Line))

	if api.Lifecycle != "" {
		sb.WriteString(fmt.Sprintf("Lifecycle:  %s\n", api.Lifecycle))
	}

	if source != nil {
		sb.WriteString(fmt.Sprintf("Source:     %s/%s\n", source.Repo, source.Path))
	}

	if release != nil && release.Current != "" {
		sb.WriteString(fmt.Sprintf("Release:    %s\n", release.Current))
		sb.WriteString(fmt.Sprintf("Tag:        %s\n", config.DeriveTag(api.ID, release.Current)))
	}

	// Language section — iterate plugins in display order.
	for _, p := range All() {
		coords, ok := langs[p.Name()]
		if !ok {
			continue
		}
		for _, line := range p.ReportLines(coords) {
			sb.WriteString(fmt.Sprintf("%-12s%s\n", line.Label+":", line.Value))
		}
	}

	return sb.String()
}

// FormatLanguageLines returns only the language-specific lines from
// the identity report. Useful for manifest and record formatting.
func FormatLanguageLines(langs map[string]config.LanguageCoords) string {
	var sb strings.Builder
	for _, p := range All() {
		coords, ok := langs[p.Name()]
		if !ok {
			continue
		}
		for _, line := range p.ReportLines(coords) {
			sb.WriteString(fmt.Sprintf("%-12s%s\n", line.Label+":", line.Value))
		}
	}
	return sb.String()
}

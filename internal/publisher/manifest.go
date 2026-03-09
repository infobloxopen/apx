package publisher

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/infobloxopen/apx/internal/config"
)

// ReleaseManifest is the single machine-readable artifact that travels from
// app repo to canonical CI. It is the source of truth for validation, audit,
// troubleshooting, and release inspection.
type ReleaseManifest struct {
	// Header
	SchemaVersion string `yaml:"schema_version" json:"schema_version"` // always "1"

	// State
	State ReleaseState `yaml:"state" json:"state"`

	// Identity
	APIID     string `yaml:"api_id" json:"api_id"`
	Format    string `yaml:"format" json:"format"`
	Domain    string `yaml:"domain" json:"domain"`
	Name      string `yaml:"name" json:"name"`
	Line      string `yaml:"line" json:"line"`
	Lifecycle string `yaml:"lifecycle,omitempty" json:"lifecycle,omitempty"`

	// Source
	SourceRepo   string `yaml:"source_repo" json:"source_repo"`
	SourcePath   string `yaml:"source_path" json:"source_path"`
	SourceCommit string `yaml:"source_commit,omitempty" json:"source_commit,omitempty"`

	// Release
	RequestedVersion string `yaml:"requested_version" json:"requested_version"`

	// Canonical destination
	CanonicalRepo string `yaml:"canonical_repo" json:"canonical_repo"`
	CanonicalPath string `yaml:"canonical_path" json:"canonical_path"`

	// Language coordinates
	GoModule       string `yaml:"go_module,omitempty" json:"go_module,omitempty"`
	GoImport       string `yaml:"go_import,omitempty" json:"go_import,omitempty"`
	PythonDistName string `yaml:"python_dist_name,omitempty" json:"python_dist_name,omitempty"`
	PythonImport   string `yaml:"python_import,omitempty" json:"python_import,omitempty"`

	// Tag
	Tag string `yaml:"tag" json:"tag"`

	// PR metadata (populated after submit creates a pull request)
	PRNumber int    `yaml:"pr_number,omitempty" json:"pr_number,omitempty"`
	PRURL    string `yaml:"pr_url,omitempty" json:"pr_url,omitempty"`
	PRBranch string `yaml:"pr_branch,omitempty" json:"pr_branch,omitempty"`

	// CI provenance (populated when submit runs in a CI environment)
	CIProvider string `yaml:"ci_provider,omitempty" json:"ci_provider,omitempty"`
	CIRunURL   string `yaml:"ci_run_url,omitempty" json:"ci_run_url,omitempty"`

	// Validation results
	Validation *ValidationResults `yaml:"validation,omitempty" json:"validation,omitempty"`

	// Timestamps
	PreparedAt  string `yaml:"prepared_at,omitempty" json:"prepared_at,omitempty"`
	SubmittedAt string `yaml:"submitted_at,omitempty" json:"submitted_at,omitempty"`
	FinalizedAt string `yaml:"finalized_at,omitempty" json:"finalized_at,omitempty"`

	// Error info (populated when state == StateFailed)
	Error *ManifestError `yaml:"error,omitempty" json:"error,omitempty"`
}

// ValidationResults records which validation steps passed or failed.
type ValidationResults struct {
	Lint      ValidationStatus `yaml:"lint" json:"lint"`
	Breaking  ValidationStatus `yaml:"breaking" json:"breaking"`
	Policy    ValidationStatus `yaml:"policy" json:"policy"`
	GoPackage ValidationStatus `yaml:"go_package,omitempty" json:"go_package,omitempty"`
	GoMod     ValidationStatus `yaml:"go_mod,omitempty" json:"go_mod,omitempty"`
}

// ValidationStatus is a typed string for validation outcomes.
type ValidationStatus string

const (
	ValidationPassed  ValidationStatus = "passed"
	ValidationFailed  ValidationStatus = "failed"
	ValidationSkipped ValidationStatus = "skipped"
)

// ManifestError captures failure details in the manifest.
type ManifestError struct {
	Code    string `yaml:"code" json:"code"`
	Message string `yaml:"message" json:"message"`
	Phase   string `yaml:"phase,omitempty" json:"phase,omitempty"`
}

// NewManifest creates a ReleaseManifest from the existing identity types.
// The manifest starts in StateDraft.
func NewManifest(
	api *config.APIIdentity,
	source *config.SourceIdentity,
	langs map[string]config.LanguageCoords,
	version string,
	canonicalRepo string,
) *ReleaseManifest {
	m := &ReleaseManifest{
		SchemaVersion:    "1",
		State:            StateDraft,
		APIID:            api.ID,
		Format:           api.Format,
		Domain:           api.Domain,
		Name:             api.Name,
		Line:             api.Line,
		Lifecycle:        api.Lifecycle,
		SourceRepo:       source.Repo,
		SourcePath:       source.Path,
		RequestedVersion: version,
		CanonicalRepo:    canonicalRepo,
		CanonicalPath:    api.ID,
		Tag:              config.DeriveTag(api.ID, version),
	}

	if goCoords, ok := langs["go"]; ok {
		m.GoModule = goCoords.Module
		m.GoImport = goCoords.Import
	}

	if pyCoords, ok := langs["python"]; ok {
		m.PythonDistName = pyCoords.Module
		m.PythonImport = pyCoords.Import
	}

	return m
}

// SetState transitions the manifest to a new state, recording the timestamp.
// Returns an error if the transition is illegal.
func (m *ReleaseManifest) SetState(next ReleaseState) error {
	if err := ValidateTransition(m.State, next); err != nil {
		return err
	}
	m.State = next

	now := time.Now().UTC().Format(time.RFC3339)
	switch next {
	case StatePrepared:
		m.PreparedAt = now
	case StateSubmitted, StateCanonicalPROpen:
		m.SubmittedAt = now
	case StateCanonicalReleased, StatePackagePublished:
		m.FinalizedAt = now
	}

	return nil
}

// Fail transitions the manifest to StateFailed with error details.
func (m *ReleaseManifest) Fail(code, message, phase string) {
	m.State = StateFailed
	m.Error = &ManifestError{
		Code:    code,
		Message: message,
		Phase:   phase,
	}
}

// WriteManifest serializes the manifest to YAML and writes it to the given path.
func WriteManifest(m *ReleaseManifest, path string) error {
	data, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshalling manifest: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing manifest to %s: %w", path, err)
	}
	return nil
}

// ReadManifest reads and deserializes a manifest from a YAML file.
func ReadManifest(path string) (*ReleaseManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading manifest from %s: %w", path, err)
	}
	var m ReleaseManifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing manifest from %s: %w", path, err)
	}
	return &m, nil
}

// MarshalManifest returns the YAML bytes for a manifest.
func MarshalManifest(m *ReleaseManifest) ([]byte, error) {
	return yaml.Marshal(m)
}

// FormatManifestReport returns a human-readable summary of the manifest.
func FormatManifestReport(m *ReleaseManifest) string {
	var lines []string
	lines = append(lines, fmt.Sprintf("State:       %s", m.State))
	lines = append(lines, fmt.Sprintf("API ID:      %s", m.APIID))
	lines = append(lines, fmt.Sprintf("Format:      %s", m.Format))
	lines = append(lines, fmt.Sprintf("Domain:      %s", m.Domain))
	lines = append(lines, fmt.Sprintf("Name:        %s", m.Name))
	lines = append(lines, fmt.Sprintf("Line:        %s", m.Line))
	if m.Lifecycle != "" {
		lines = append(lines, fmt.Sprintf("Lifecycle:   %s", m.Lifecycle))
	}
	lines = append(lines, fmt.Sprintf("Version:     %s", m.RequestedVersion))
	lines = append(lines, fmt.Sprintf("Tag:         %s", m.Tag))
	lines = append(lines, fmt.Sprintf("Source:      %s/%s", m.SourceRepo, m.SourcePath))
	if m.SourceCommit != "" {
		lines = append(lines, fmt.Sprintf("Commit:      %s", m.SourceCommit))
	}
	lines = append(lines, fmt.Sprintf("Canonical:   %s/%s", m.CanonicalRepo, m.CanonicalPath))
	if m.GoModule != "" {
		lines = append(lines, fmt.Sprintf("Go module:   %s", m.GoModule))
		lines = append(lines, fmt.Sprintf("Go import:   %s", m.GoImport))
	}
	if m.PythonDistName != "" {
		lines = append(lines, fmt.Sprintf("Py dist:     %s", m.PythonDistName))
		lines = append(lines, fmt.Sprintf("Py import:   %s", m.PythonImport))
	}
	if m.PRURL != "" {
		lines = append(lines, fmt.Sprintf("PR URL:      %s", m.PRURL))
		if m.PRNumber != 0 {
			lines = append(lines, fmt.Sprintf("PR number:   %d", m.PRNumber))
		}
		if m.PRBranch != "" {
			lines = append(lines, fmt.Sprintf("PR branch:   %s", m.PRBranch))
		}
	}
	if m.CIProvider != "" {
		lines = append(lines, fmt.Sprintf("CI provider: %s", m.CIProvider))
		if m.CIRunURL != "" {
			lines = append(lines, fmt.Sprintf("CI run URL:  %s", m.CIRunURL))
		}
	}
	if m.Validation != nil {
		lines = append(lines, "Validation:")
		lines = append(lines, fmt.Sprintf("  lint:      %s", m.Validation.Lint))
		lines = append(lines, fmt.Sprintf("  breaking:  %s", m.Validation.Breaking))
		lines = append(lines, fmt.Sprintf("  policy:    %s", m.Validation.Policy))
		if m.Validation.GoPackage != "" {
			lines = append(lines, fmt.Sprintf("  go_package:%s", m.Validation.GoPackage))
		}
		if m.Validation.GoMod != "" {
			lines = append(lines, fmt.Sprintf("  go_mod:    %s", m.Validation.GoMod))
		}
	}
	if m.Error != nil {
		lines = append(lines, fmt.Sprintf("Error:       [%s] %s", m.Error.Code, m.Error.Message))
		if m.Error.Phase != "" {
			lines = append(lines, fmt.Sprintf("  phase:     %s", m.Error.Phase))
		}
	}
	if m.PreparedAt != "" {
		lines = append(lines, fmt.Sprintf("Prepared:    %s", m.PreparedAt))
	}
	if m.SubmittedAt != "" {
		lines = append(lines, fmt.Sprintf("Submitted:   %s", m.SubmittedAt))
	}
	if m.FinalizedAt != "" {
		lines = append(lines, fmt.Sprintf("Finalized:   %s", m.FinalizedAt))
	}

	result := ""
	for _, l := range lines {
		result += l + "\n"
	}
	return result
}

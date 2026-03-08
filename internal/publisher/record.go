package publisher

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// ReleaseRecord is the immutable audit artifact produced by canonical CI
// after a release has been finalized. It captures exactly what was published,
// from where, by which pipeline, and with which artifacts.
type ReleaseRecord struct {
	// Header
	SchemaVersion string `yaml:"schema_version" json:"schema_version"` // always "1"
	Kind          string `yaml:"kind" json:"kind"`                     // always "release-record"

	// Identity (copied from manifest)
	APIID     string `yaml:"api_id" json:"api_id"`
	Format    string `yaml:"format" json:"format"`
	Domain    string `yaml:"domain" json:"domain"`
	Name      string `yaml:"name" json:"name"`
	Line      string `yaml:"line" json:"line"`
	Lifecycle string `yaml:"lifecycle,omitempty" json:"lifecycle,omitempty"`

	// Source provenance
	SourceRepo   string `yaml:"source_repo" json:"source_repo"`
	SourcePath   string `yaml:"source_path" json:"source_path"`
	SourceCommit string `yaml:"source_commit" json:"source_commit"`

	// Published version
	Version string `yaml:"version" json:"version"`
	Tag     string `yaml:"tag" json:"tag"`

	// Canonical destination
	CanonicalRepo   string `yaml:"canonical_repo" json:"canonical_repo"`
	CanonicalPath   string `yaml:"canonical_path" json:"canonical_path"`
	CanonicalCommit string `yaml:"canonical_commit,omitempty" json:"canonical_commit,omitempty"`

	// Language coordinates
	GoModule string `yaml:"go_module,omitempty" json:"go_module,omitempty"`
	GoImport string `yaml:"go_import,omitempty" json:"go_import,omitempty"`

	// Validation results (re-validated in canonical CI)
	Validation *ValidationResults `yaml:"validation,omitempty" json:"validation,omitempty"`

	// Published artifacts
	Artifacts []ReleaseArtifact `yaml:"artifacts,omitempty" json:"artifacts,omitempty"`

	// Catalog
	CatalogUpdated bool   `yaml:"catalog_updated" json:"catalog_updated"`
	CatalogPath    string `yaml:"catalog_path,omitempty" json:"catalog_path,omitempty"`

	// CI provenance (optional, populated by CI environment)
	CIProvider string `yaml:"ci_provider,omitempty" json:"ci_provider,omitempty"`
	CIRunID    string `yaml:"ci_run_id,omitempty" json:"ci_run_id,omitempty"`
	CIRunURL   string `yaml:"ci_run_url,omitempty" json:"ci_run_url,omitempty"`

	// Timestamps
	PreparedAt  string `yaml:"prepared_at,omitempty" json:"prepared_at,omitempty"`
	SubmittedAt string `yaml:"submitted_at,omitempty" json:"submitted_at,omitempty"`
	FinalizedAt string `yaml:"finalized_at" json:"finalized_at"`
}

// ReleaseArtifact describes a single artifact produced by a release.
type ReleaseArtifact struct {
	Type    string `yaml:"type" json:"type"`                           // e.g. "go-module", "npm-package", "proto-descriptor"
	Name    string `yaml:"name" json:"name"`                           // e.g. module path or package name
	Version string `yaml:"version,omitempty" json:"version,omitempty"` // artifact version if different
	Status  string `yaml:"status" json:"status"`                       // "published", "skipped", "failed"
}

// NewReleaseRecord creates a ReleaseRecord from a finalized manifest.
func NewReleaseRecord(m *ReleaseManifest) *ReleaseRecord {
	return &ReleaseRecord{
		SchemaVersion:  "1",
		Kind:           "release-record",
		APIID:          m.APIID,
		Format:         m.Format,
		Domain:         m.Domain,
		Name:           m.Name,
		Line:           m.Line,
		Lifecycle:      m.Lifecycle,
		SourceRepo:     m.SourceRepo,
		SourcePath:     m.SourcePath,
		SourceCommit:   m.SourceCommit,
		Version:        m.RequestedVersion,
		Tag:            m.Tag,
		CanonicalRepo:  m.CanonicalRepo,
		CanonicalPath:  m.CanonicalPath,
		GoModule:       m.GoModule,
		GoImport:       m.GoImport,
		Validation:     m.Validation,
		CatalogUpdated: false,
		PreparedAt:     m.PreparedAt,
		SubmittedAt:    m.SubmittedAt,
		FinalizedAt:    time.Now().UTC().Format(time.RFC3339),
	}
}

// AddArtifact adds a published artifact to the record.
func (r *ReleaseRecord) AddArtifact(artifactType, name, version, status string) {
	r.Artifacts = append(r.Artifacts, ReleaseArtifact{
		Type:    artifactType,
		Name:    name,
		Version: version,
		Status:  status,
	})
}

// DetectCI captures CI environment variables into the record.
func (r *ReleaseRecord) DetectCI() {
	// GitHub Actions
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		r.CIProvider = "github-actions"
		r.CIRunID = os.Getenv("GITHUB_RUN_ID")
		serverURL := os.Getenv("GITHUB_SERVER_URL")
		repo := os.Getenv("GITHUB_REPOSITORY")
		runID := os.Getenv("GITHUB_RUN_ID")
		if serverURL != "" && repo != "" && runID != "" {
			r.CIRunURL = fmt.Sprintf("%s/%s/actions/runs/%s", serverURL, repo, runID)
		}
		return
	}

	// GitLab CI
	if os.Getenv("GITLAB_CI") == "true" {
		r.CIProvider = "gitlab-ci"
		r.CIRunID = os.Getenv("CI_PIPELINE_ID")
		r.CIRunURL = os.Getenv("CI_PIPELINE_URL")
		return
	}

	// Jenkins
	if os.Getenv("JENKINS_URL") != "" {
		r.CIProvider = "jenkins"
		r.CIRunID = os.Getenv("BUILD_ID")
		r.CIRunURL = os.Getenv("BUILD_URL")
		return
	}
}

// WriteReleaseRecord serializes the record to YAML and writes it to the given path.
func WriteReleaseRecord(r *ReleaseRecord, path string) error {
	data, err := yaml.Marshal(r)
	if err != nil {
		return fmt.Errorf("marshalling release record: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing release record to %s: %w", path, err)
	}
	return nil
}

// ReadReleaseRecord reads and deserializes a release record from YAML.
func ReadReleaseRecord(path string) (*ReleaseRecord, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading release record from %s: %w", path, err)
	}
	var r ReleaseRecord
	if err := yaml.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("parsing release record from %s: %w", path, err)
	}
	return &r, nil
}

// MarshalReleaseRecord returns the YAML bytes for a release record.
func MarshalReleaseRecord(r *ReleaseRecord) ([]byte, error) {
	return yaml.Marshal(r)
}

// FormatRecordReport returns a human-readable summary of the release record.
func FormatRecordReport(r *ReleaseRecord) string {
	var lines []string
	lines = append(lines, fmt.Sprintf("API ID:      %s", r.APIID))
	lines = append(lines, fmt.Sprintf("Version:     %s", r.Version))
	lines = append(lines, fmt.Sprintf("Tag:         %s", r.Tag))
	lines = append(lines, fmt.Sprintf("Lifecycle:   %s", r.Lifecycle))
	lines = append(lines, fmt.Sprintf("Format:      %s", r.Format))
	lines = append(lines, fmt.Sprintf("Source:      %s/%s @ %s", r.SourceRepo, r.SourcePath, r.SourceCommit))
	lines = append(lines, fmt.Sprintf("Canonical:   %s/%s", r.CanonicalRepo, r.CanonicalPath))
	if r.CanonicalCommit != "" {
		lines = append(lines, fmt.Sprintf("  commit:    %s", r.CanonicalCommit))
	}
	if r.GoModule != "" {
		lines = append(lines, fmt.Sprintf("Go module:   %s", r.GoModule))
		lines = append(lines, fmt.Sprintf("Go import:   %s", r.GoImport))
	}
	if r.Validation != nil {
		lines = append(lines, "Validation:")
		lines = append(lines, fmt.Sprintf("  lint:      %s", r.Validation.Lint))
		lines = append(lines, fmt.Sprintf("  breaking:  %s", r.Validation.Breaking))
		lines = append(lines, fmt.Sprintf("  policy:    %s", r.Validation.Policy))
	}
	if len(r.Artifacts) > 0 {
		lines = append(lines, "Artifacts:")
		for _, a := range r.Artifacts {
			lines = append(lines, fmt.Sprintf("  %s: %s (%s)", a.Type, a.Name, a.Status))
		}
	}
	lines = append(lines, fmt.Sprintf("Catalog:     %v", r.CatalogUpdated))
	if r.CIProvider != "" {
		lines = append(lines, fmt.Sprintf("CI:          %s", r.CIProvider))
		if r.CIRunURL != "" {
			lines = append(lines, fmt.Sprintf("  run:       %s", r.CIRunURL))
		}
	}
	lines = append(lines, fmt.Sprintf("Finalized:   %s", r.FinalizedAt))

	result := ""
	for _, l := range lines {
		result += l + "\n"
	}
	return result
}

package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/infobloxopen/apx/internal/catalog"
	"github.com/infobloxopen/apx/internal/publisher"
	"github.com/infobloxopen/apx/internal/ui"
)

// ciOnlyConfigYAML is a minimal, schema-valid apx.yaml for a ci_only canonical
// repo, used by the finalize ci_only gate tests.
const ciOnlyConfigYAML = `version: 1
org: acme
repo: apis
release:
  tag_format: "{subdir}/v{version}"
  ci_only: true
`

// clearCIEnv makes runningInCI() report false for the duration of a test so the
// ci_only local guidance gate is exercised deterministically regardless of the
// host CI environment.
func clearCIEnv(t *testing.T) {
	t.Helper()
	t.Setenv("GITHUB_ACTIONS", "")
	t.Setenv("GITLAB_CI", "")
	t.Setenv("JENKINS_URL", "")
	t.Setenv("CI", "")
}

func TestInferLifecycleFromVersion(t *testing.T) {
	tests := []struct {
		version  string
		expected string
	}{
		{"v1.0.0", "stable"},
		{"v1.2.3", "stable"},
		{"v1.0.0-alpha.1", "experimental"},
		{"v1.0.0-alpha.2+build", "experimental"},
		{"v1.0.0-beta.1", "beta"},
		{"v1.0.0-beta.3", "beta"},
		{"v1.0.0-rc.1", "beta"},
		{"v1.0.0-dev.1", ""}, // dev is unknown
		{"not-a-version", ""},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			got := inferLifecycleFromVersion(tt.version)
			if got != tt.expected {
				t.Errorf("inferLifecycleFromVersion(%q) = %q, want %q", tt.version, got, tt.expected)
			}
		})
	}
}

func TestCurrentLifecycleLabel(t *testing.T) {
	if got := currentLifecycleLabel(""); got != "(unknown)" {
		t.Errorf("currentLifecycleLabel(\"\") = %q, want \"(unknown)\"", got)
	}
	if got := currentLifecycleLabel("stable"); got != "stable" {
		t.Errorf("currentLifecycleLabel(\"stable\") = %q, want \"stable\"", got)
	}
}

func TestUpdateLatestStable(t *testing.T) {
	tests := []struct {
		current   string
		version   string
		lifecycle string
		expected  string
	}{
		{"", "v1.0.0", "stable", "v1.0.0"},
		{"v1.0.0", "v1.1.0", "stable", "v1.1.0"},
		{"v1.1.0", "v1.0.0", "stable", "v1.1.0"},
		{"v1.0.0", "v1.1.0-beta.1", "beta", "v1.0.0"},
		{"", "v1.0.0-alpha.1", "experimental", ""},
	}

	for _, tt := range tests {
		got := updateLatestStable(tt.current, tt.version, tt.lifecycle)
		if got != tt.expected {
			t.Errorf("updateLatestStable(%q, %q, %q) = %q, want %q",
				tt.current, tt.version, tt.lifecycle, got, tt.expected)
		}
	}
}

func TestUpdateLatestPrerelease(t *testing.T) {
	tests := []struct {
		current   string
		version   string
		lifecycle string
		expected  string
	}{
		{"", "v1.0.0-beta.1", "beta", "v1.0.0-beta.1"},
		{"v1.0.0-beta.1", "v1.0.0-beta.2", "beta", "v1.0.0-beta.2"},
		{"v1.0.0-beta.2", "v1.0.0-beta.1", "beta", "v1.0.0-beta.2"},
		{"v1.0.0-beta.1", "v1.0.0", "stable", "v1.0.0-beta.1"},
		{"", "v1.0.0", "stable", ""},
	}

	for _, tt := range tests {
		got := updateLatestPrerelease(tt.current, tt.version, tt.lifecycle)
		if got != tt.expected {
			t.Errorf("updateLatestPrerelease(%q, %q, %q) = %q, want %q",
				tt.current, tt.version, tt.lifecycle, got, tt.expected)
		}
	}
}

func TestContainsString(t *testing.T) {
	slice := []string{"a", "b", "c"}
	if !containsString(slice, "b") {
		t.Error("expected containsString to return true for 'b'")
	}
	if containsString(slice, "d") {
		t.Error("expected containsString to return false for 'd'")
	}
}

func TestReleaseFinalizeCmd_NoManifest(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	var stdout strings.Builder
	ui.SetOutput(&stdout)
	defer ui.SetOutput(os.Stdout)

	cmd := NewRootCmd("test")
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"release", "finalize"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when no manifest exists")
	}
}

func TestReleaseFinalizeCmd_WrongState(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Create a manifest in "prepared" state (not submitted)
	m := &publisher.ReleaseManifest{
		SchemaVersion:    "1",
		State:            publisher.StatePrepared,
		APIID:            "proto/payments/ledger/v1",
		Format:           "proto",
		Domain:           "payments",
		Name:             "ledger",
		Line:             "v1",
		RequestedVersion: "v1.0.0",
		Tag:              "proto/payments/ledger/v1/v1.0.0",
		CanonicalRepo:    "github.com/acme/apis",
		CanonicalPath:    "proto/payments/ledger/v1",
	}
	publisher.WriteManifest(m, ".apx-release.yaml")

	var stdout strings.Builder
	ui.SetOutput(&stdout)
	defer ui.SetOutput(os.Stdout)

	cmd := NewRootCmd("test")
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"release", "finalize"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when manifest is in 'prepared' state")
	}
}

func TestReleaseFinalizeCmd_AlreadyFinalized(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Create a manifest in "package-published" state
	m := &publisher.ReleaseManifest{
		SchemaVersion:    "1",
		State:            publisher.StatePackagePublished,
		APIID:            "proto/payments/ledger/v1",
		Format:           "proto",
		RequestedVersion: "v1.0.0",
		Tag:              "proto/payments/ledger/v1/v1.0.0",
	}
	publisher.WriteManifest(m, ".apx-release.yaml")

	var stdout strings.Builder
	ui.SetOutput(&stdout)
	defer ui.SetOutput(os.Stdout)

	cmd := NewRootCmd("test")
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"release", "finalize"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected no error for already-finalized release: %v", err)
	}
	if !strings.Contains(stdout.String(), "already finalized") {
		t.Error("expected 'already finalized' message")
	}
}

func TestReleaseHistoryCmd_NoAPIID(t *testing.T) {
	var stdout strings.Builder
	ui.SetOutput(&stdout)
	defer ui.SetOutput(os.Stdout)

	cmd := NewRootCmd("test")
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"release", "history"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when no API ID provided")
	}
}

func TestReleaseHistoryCmd_InvalidAPIID(t *testing.T) {
	var stdout strings.Builder
	ui.SetOutput(&stdout)
	defer ui.SetOutput(os.Stdout)

	cmd := NewRootCmd("test")
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"release", "history", "invalid-id"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid API ID")
	}
}

func TestReleasePromoteCmd_NoTarget(t *testing.T) {
	var stdout strings.Builder
	ui.SetOutput(&stdout)
	defer ui.SetOutput(os.Stdout)

	cmd := NewRootCmd("test")
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"release", "promote", "proto/payments/ledger/v1"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --to not provided")
	}
}

func TestReleasePromoteCmd_InvalidTarget(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	var stdout strings.Builder
	ui.SetOutput(&stdout)
	defer ui.SetOutput(os.Stdout)

	cmd := NewRootCmd("test")
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"release", "promote", "proto/payments/ledger/v1", "--to", "invalid"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid lifecycle target")
	}
}

func TestFindDependents_NoCatalog(t *testing.T) {
	tmpDir := t.TempDir()
	deps, err := FindDependents(tmpDir, "proto/payments/ledger/v1", filepath.Join(tmpDir, "catalog.yaml"))
	if err != nil {
		// Catalog doesn't exist — expect error from Load
		return
	}
	if len(deps) != 0 {
		t.Errorf("expected no dependents, got %v", deps)
	}
}

// writeCIOnlySubmittedManifest sets up a tmp dir (as cwd) with a ci_only
// apx.yaml and a submitted release manifest, returning the tmp dir.
func writeCIOnlySubmittedManifest(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "apx.yaml"), []byte(ciOnlyConfigYAML), 0o644); err != nil {
		t.Fatalf("writing apx.yaml: %v", err)
	}
	m := &publisher.ReleaseManifest{
		SchemaVersion:    "1",
		State:            publisher.StateSubmitted,
		APIID:            "proto/infoblox/field/v1",
		Format:           "proto",
		Domain:           "infoblox",
		Name:             "field",
		Line:             "v1",
		RequestedVersion: "v1.0.0-alpha.2",
		Tag:              "proto/infoblox/field/v1.0.0-alpha.2",
		CanonicalRepo:    "github.com/acme/apis",
		CanonicalPath:    "proto/infoblox/field/v1",
	}
	oldWd, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(oldWd) })
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	if err := publisher.WriteManifest(m, ".apx-release.yaml"); err != nil {
		t.Fatalf("writing manifest: %v", err)
	}
	return tmpDir
}

// TestReleaseFinalizeCmd_CIOnlyGuidance verifies that finalizing a ci_only repo
// locally (no --local, not in CI) fails fast with actionable guidance naming
// the exact prerequisites, instead of attempting to push a protected tag.
func TestReleaseFinalizeCmd_CIOnlyGuidance(t *testing.T) {
	clearCIEnv(t)
	writeCIOnlySubmittedManifest(t)

	var stdout strings.Builder
	ui.SetOutput(&stdout)
	defer ui.SetOutput(os.Stdout)

	cmd := NewRootCmd("test")
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"release", "finalize"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected ci_only finalize to fail fast with guidance")
	}
	msg := err.Error()
	for _, want := range []string{"ci_only", "APX_APP_ID", "APX_APP_PRIVATE_KEY", "tag-ruleset bypass", "--local"} {
		if !strings.Contains(msg, want) {
			t.Errorf("guidance missing %q; got: %s", want, msg)
		}
	}
	// It must include the CI-mode finalize command with the manifest coordinates.
	if !strings.Contains(msg, "--api proto/infoblox/field/v1 --version v1.0.0-alpha.2") {
		t.Errorf("guidance missing CI-mode finalize command; got: %s", msg)
	}
}

// TestReleaseFinalizeCmd_CIOnlyLocalBypass verifies that --local bypasses the
// ci_only guidance gate. The command then proceeds and fails elsewhere (there
// is no git repo in the temp dir), but the error must NOT be the ci_only
// guidance — proving the gate was bypassed.
func TestReleaseFinalizeCmd_CIOnlyLocalBypass(t *testing.T) {
	clearCIEnv(t)
	writeCIOnlySubmittedManifest(t)

	var stdout strings.Builder
	ui.SetOutput(&stdout)
	defer ui.SetOutput(os.Stdout)

	cmd := NewRootCmd("test")
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"release", "finalize", "--local"})

	err := cmd.Execute()
	if err != nil && strings.Contains(err.Error(), "APX_APP_ID") {
		t.Errorf("--local should bypass the ci_only guidance gate; got guidance error: %v", err)
	}
}

func TestTagModulePrefix(t *testing.T) {
	tests := []struct {
		tag  string
		want string
	}{
		{"proto/infoblox/field/v1.0.0-alpha.2", "proto/infoblox/field"},
		{"proto/payments/ledger/v2/v2.0.0", "proto/payments/ledger/v2"},
		{"openapi/users/v1.1.0", "openapi/users"},
		{"v0.15.0", ""}, // repo-level tag, not a module release tag
		{"edge", ""},    // non-version tag
		{"proto/x/notver", ""},
	}
	for _, tt := range tests {
		if got := tagModulePrefix(tt.tag); got != tt.want {
			t.Errorf("tagModulePrefix(%q) = %q, want %q", tt.tag, got, tt.want)
		}
	}
}

func TestCatalogDriftFromTags(t *testing.T) {
	cat := &catalog.Catalog{
		Version: 1,
		Modules: []catalog.Module{
			{ID: "proto/infoblox/authz/v1", Format: "proto"},
			{ID: "proto/infoblox/storage/v1", Format: "proto"},
		},
	}
	tags := []string{
		"proto/infoblox/authz/v1.1.0",         // cataloged
		"proto/infoblox/storage/v1.0.0",       // cataloged
		"proto/infoblox/field/v1.0.0-alpha.2", // tagged but NOT cataloged → drift
		"proto/infoblox/field/v1.0.0-alpha.1", // same module, dedup
		"v0.15.0",                             // repo tag, ignored
	}
	drift := catalogDriftFromTags(tags, cat)
	if len(drift) != 1 || drift[0] != "proto/infoblox/field" {
		t.Fatalf("expected drift [proto/infoblox/field], got %v", drift)
	}

	// No drift when every tagged module is cataloged.
	cat.Modules = append(cat.Modules, catalog.Module{ID: "proto/infoblox/field/v1", Format: "proto"})
	if drift := catalogDriftFromTags(tags, cat); len(drift) != 0 {
		t.Fatalf("expected no drift, got %v", drift)
	}
}

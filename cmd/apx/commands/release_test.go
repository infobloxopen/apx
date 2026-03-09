package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/infobloxopen/apx/internal/publisher"
	"github.com/infobloxopen/apx/internal/ui"
)

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

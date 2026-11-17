package validator

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func opidFixture(name string) string {
	return filepath.Join("testdata", "operationid", name)
}

func TestCheckOperationIDs(t *testing.T) {
	tests := []struct {
		file string
		want []string // substrings of the error; nil means no error
	}{
		{"base.yaml", nil},
		{"no_paths.yaml", nil},
		{"removed.yaml", []string{"1 problems", "GET /widgets/{id}: operationId is missing"}},
		{"duplicate.yaml", []string{`operationId "widgets" is not unique: GET /widgets, GET /widgets/{id}`}},
		{"missing-file.yaml", []string{"operationId check"}},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			err := CheckOperationIDs(opidFixture(tt.file))
			if tt.want == nil {
				if err != nil {
					t.Fatalf("CheckOperationIDs() = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatal("CheckOperationIDs() = nil, want an error")
			}
			for _, sub := range tt.want {
				if !strings.Contains(err.Error(), sub) {
					t.Errorf("error %q does not contain %q", err, sub)
				}
			}
		})
	}
}

func TestCompareOperationIDs(t *testing.T) {
	tests := []struct {
		name      string
		base, rev string
		want      []string // substrings of the error; nil means no error
	}{
		{"no change", "base.yaml", "base.yaml", nil},
		{"a renamed path variable is not a change", "base.yaml", "same.yaml", nil},
		{"renamed id", "base.yaml", "renamed.yaml", []string{`GET /widgets: operationId "listWidgets" was renamed to "listAllWidgets"`}},
		{"removed id", "base.yaml", "removed.yaml", []string{`GET /widgets/{id}: operationId "getWidget" was removed`}},
		{"removed operation", "base.yaml", "operation_removed.yaml", []string{`operationId "getWidget" was removed with its operation`}},
		{"an id that moves to another path stays", "base.yaml", "moved.yaml", nil},
		{"an id added where none was", "removed.yaml", "base.yaml", nil},
		{"a base that does not load is skipped", "missing-file.yaml", "base.yaml", nil},
		{"a revision without paths is skipped", "base.yaml", "no_paths.yaml", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CompareOperationIDs(opidFixture(tt.base), opidFixture(tt.rev))
			if tt.want == nil {
				if err != nil {
					t.Fatalf("CompareOperationIDs() = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatal("CompareOperationIDs() = nil, want an error")
			}
			for _, sub := range tt.want {
				if !strings.Contains(err.Error(), sub) {
					t.Errorf("error %q does not contain %q", err, sub)
				}
			}
		})
	}
}

// TestOpenAPIValidator_LintOperationIDs shows that the built-in check runs
// without spectral: a resolver in offline mode finds no tool, and the
// operationId finding is still in the error.
func TestOpenAPIValidator_LintOperationIDs(t *testing.T) {
	offline := NewToolchainResolver(WithOfflineMode(true), WithBundlePath(t.TempDir()))

	v := NewOpenAPIValidator(offline)
	err := v.Lint(opidFixture("duplicate.yaml"))
	if err == nil || !strings.Contains(err.Error(), "is not unique") {
		t.Errorf("Lint() = %v, want the operationId finding", err)
	}

	v.SetOperationIDChecks(false)
	err = v.Lint(opidFixture("duplicate.yaml"))
	if err != nil && strings.Contains(err.Error(), "is not unique") {
		t.Errorf("Lint() with the check off still reports operationIds: %v", err)
	}
}

// TestOpenAPIValidator_BreakingOperationIDs runs the real oasdiff on fixtures.
// It records what oasdiff alone says about a renamed and a removed
// operationId, and it requires that Breaking reports both.
func TestOpenAPIValidator_BreakingOperationIDs(t *testing.T) {
	if _, err := exec.LookPath("oasdiff"); err != nil {
		t.Skip("oasdiff not available")
	}
	base := opidFixture("base.yaml")

	for _, rev := range []string{"renamed.yaml", "removed.yaml"} {
		t.Run(rev, func(t *testing.T) {
			// oasdiff alone, as Breaking calls it.
			alone := NewOpenAPIValidator(NewToolchainResolver())
			alone.SetOperationIDChecks(false)
			if err := alone.Breaking(opidFixture(rev), base); err != nil {
				t.Logf("oasdiff alone reports the change: %v", err)
			} else {
				t.Logf("oasdiff alone does NOT report the change of %s as breaking (exit 0)", rev)
			}

			// Breaking with the apx comparison.
			v := NewOpenAPIValidator(NewToolchainResolver())
			err := v.Breaking(opidFixture(rev), base)
			if err == nil {
				t.Fatal("Breaking() = nil, want the operationId change as a breaking finding")
			}
			if !strings.Contains(err.Error(), "operationId") {
				t.Errorf("Breaking() error does not name the operationId: %v", err)
			}
		})
	}

	t.Run("compatible revision", func(t *testing.T) {
		v := NewOpenAPIValidator(NewToolchainResolver())
		if err := v.Breaking(opidFixture("same.yaml"), base); err != nil {
			t.Errorf("Breaking() = %v, want nil for a revision that only renames a path variable", err)
		}
	})
}

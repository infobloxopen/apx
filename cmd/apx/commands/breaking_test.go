package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/infobloxopen/apx/internal/ui"
)

const breakingBaseSpec = `openapi: 3.0.3
info:
  title: Widgets
  version: 1.0.0
paths:
  /widgets:
    get:
      operationId: listWidgets
      responses:
        '200':
          description: ok
`

// TestBreaking_OperationIDRename runs `apx breaking` with the real oasdiff on a
// revision whose only change is a renamed operationId.
func TestBreaking_OperationIDRename(t *testing.T) {
	if _, err := exec.LookPath("oasdiff"); err != nil {
		t.Skip("oasdiff not available")
	}
	dir := t.TempDir()
	base := filepath.Join(dir, "base.yaml")
	rev := filepath.Join(dir, "rev.yaml")
	if err := os.WriteFile(base, []byte(breakingBaseSpec), 0o644); err != nil {
		t.Fatal(err)
	}
	renamed := strings.Replace(breakingBaseSpec, "listWidgets", "listAllWidgets", 1)
	if err := os.WriteFile(rev, []byte(renamed), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		extra   []string
		wantErr bool
		wantOut string
	}{
		{"a renamed operationId is breaking", nil, true, `operationId "listWidgets" was renamed to "listAllWidgets"`},
		{"--advisory reports and does not fail", []string{"--advisory"}, false, "advisory"},
		{"--operation-ids=false turns the comparison off", []string{"--operation-ids=false"}, false, "No breaking changes detected"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out strings.Builder
			ui.SetOutput(&out)
			ui.SetErrorOutput(&out)
			defer ui.SetOutput(os.Stdout)
			defer ui.SetErrorOutput(os.Stderr)

			cmd := NewRootCmd("test")
			cmd.SetOut(&out)
			cmd.SetErr(&out)
			cmd.SetArgs(append([]string{"breaking", rev, "--against", base, "--format", "openapi"}, tt.extra...))
			err := cmd.Execute()
			if (err != nil) != tt.wantErr {
				t.Fatalf("breaking error = %v, wantErr %v\noutput:\n%s", err, tt.wantErr, out.String())
			}
			if !strings.Contains(out.String(), tt.wantOut) {
				t.Errorf("output does not contain %q:\n%s", tt.wantOut, out.String())
			}
		})
	}
}

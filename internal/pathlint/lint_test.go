package pathlint

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const ingressYAML = `apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: test-ing
spec:
  rules:
    - http:
        paths:
          - path: /matched
            pathType: Prefix
          - path: /undeclared
            pathType: Exact
`

const specYAML = `openapi: 3.0.3
info:
  title: t
  version: v1
paths:
  /matched:
    get:
      responses:
        "200":
          description: ok
  /unreachable:
    get:
      responses:
        "200":
          description: ok
`

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestAnalyzeCoverageBuckets(t *testing.T) {
	dir := t.TempDir()
	ingress := writeFile(t, dir, "ingress.yaml", ingressYAML)
	spec := writeFile(t, dir, "spec.yaml", specYAML)

	r, err := Analyze([]string{ingress}, []string{spec}, nil, "test")
	if err != nil {
		t.Fatal(err)
	}

	// /matched (Prefix) ↔ /matched spec = 1 matched pair.
	// /undeclared (Exact) has no spec = 1 undeclared.
	// /unreachable spec has no ingress rule = 1 unreachable.
	if r.Undeclared != 1 {
		t.Errorf("Undeclared = %d, want 1", r.Undeclared)
	}
	if r.Unreachable != 1 {
		t.Errorf("Unreachable = %d, want 1", r.Unreachable)
	}
	if r.MatchedCount != 1 {
		t.Errorf("MatchedCount = %d, want 1", r.MatchedCount)
	}
	if !r.Drifted() {
		t.Errorf("Drifted() = false, want true")
	}
}

func TestReportWriteResultLine(t *testing.T) {
	dir := t.TempDir()
	ingress := writeFile(t, dir, "ingress.yaml", ingressYAML)
	spec := writeFile(t, dir, "spec.yaml", specYAML)
	r, err := Analyze([]string{ingress}, []string{spec}, nil, "test")
	if err != nil {
		t.Fatal(err)
	}

	var fail bytes.Buffer
	r.Write(&fail, false)
	got := fail.String()
	for _, want := range []string{
		"pathlint report",
		"[1] UNDECLARED INGRESS SURFACE",
		"[2] SPEC PATHS NOT REACHABLE",
		"[3] MATCHED",
		"RESULT: FAIL (undeclared=1 unreachable=1 matched_pairs=1)",
		"/undeclared",
		"/unreachable",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("report missing %q\n--- report ---\n%s", want, got)
		}
	}

	var warn bytes.Buffer
	r.Write(&warn, true)
	if !strings.Contains(warn.String(), "RESULT: WARN (--warn-only)") {
		t.Errorf("warn-only report missing WARN result line:\n%s", warn.String())
	}
}

func TestAnalyzeNoDrift(t *testing.T) {
	dir := t.TempDir()
	// Ingress and spec agree exactly: one prefix rule, one spec path under it.
	ingress := writeFile(t, dir, "ingress.yaml", `apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: clean
spec:
  rules:
    - http:
        paths:
          - path: /v2
            pathType: Prefix
`)
	spec := writeFile(t, dir, "spec.yaml", `openapi: 3.0.3
info: {title: t, version: v2}
paths:
  /v2/things/{id}:
    get:
      responses:
        "200":
          description: ok
`)
	r, err := Analyze([]string{ingress}, []string{spec}, nil, "test")
	if err != nil {
		t.Fatal(err)
	}
	if r.Drifted() {
		t.Errorf("Drifted() = true, want false (undeclared=%d unreachable=%d)", r.Undeclared, r.Unreachable)
	}
}

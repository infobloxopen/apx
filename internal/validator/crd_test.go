package validator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validCRD = `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: widgets.example.com
spec:
  group: example.com
  names:
    kind: Widget
    listKind: WidgetList
    plural: widgets
    singular: widget
  scope: Namespaced
  versions:
    - name: v1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                size:
                  type: integer
                  maximum: 100
                color:
                  type: string
                  enum: [red, green, blue]
                label:
                  type: string
                  maxLength: 64
              required: [size, color]
`

func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	return p
}

func newCRDValidator() *CRDValidator {
	return NewCRDValidator(&ToolchainResolver{})
}

func TestCRDLint_Valid(t *testing.T) {
	p := writeTemp(t, "widget.yaml", validCRD)
	if err := newCRDValidator().Lint(p); err != nil {
		t.Fatalf("expected valid CRD to pass lint, got: %v", err)
	}
}

func TestCRDLint_Violations(t *testing.T) {
	bad := `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: wrong.example.com
spec:
  group: example.com
  names:
    kind: Widget
    plural: widgets
    singular: widget
  scope: Weird
  versions:
    - name: v1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                broken:
                  type: notatype
    - name: v2
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
`
	p := writeTemp(t, "bad.yaml", bad)
	err := newCRDValidator().Lint(p)
	if err == nil {
		t.Fatal("expected lint to fail")
	}
	for _, want := range []string{
		"metadata.name must be",
		"spec.scope must be Namespaced or Cluster",
		"exactly one version must set storage: true (found 2)",
		"invalid type",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("lint error missing %q; got:\n%s", want, err.Error())
		}
	}
}

func TestCRDLint_MissingType(t *testing.T) {
	crd := `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: widgets.example.com
spec:
  group: example.com
  names: {kind: Widget, plural: widgets, singular: widget}
  scope: Namespaced
  versions:
    - name: v1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                notyped: {}
`
	p := writeTemp(t, "c.yaml", crd)
	err := newCRDValidator().Lint(p)
	if err == nil || !strings.Contains(err.Error(), "missing type") {
		t.Fatalf("expected missing-type violation, got: %v", err)
	}
}

func TestCRDLint_EscapeHatches(t *testing.T) {
	// x-kubernetes-preserve-unknown-fields and x-kubernetes-int-or-string are
	// structural escape hatches; a node with either need not carry a type.
	crd := `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: widgets.example.com
spec:
  group: example.com
  names: {kind: Widget, plural: widgets, singular: widget}
  scope: Namespaced
  versions:
    - name: v1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                freeform:
                  x-kubernetes-preserve-unknown-fields: true
                quantity:
                  x-kubernetes-int-or-string: true
`
	p := writeTemp(t, "c.yaml", crd)
	if err := newCRDValidator().Lint(p); err != nil {
		t.Fatalf("expected escape-hatch CRD to pass, got: %v", err)
	}
}

func TestCRDLint_V1beta1Rejected(t *testing.T) {
	crd := strings.Replace(validCRD, "apiextensions.k8s.io/v1", "apiextensions.k8s.io/v1beta1", 1)
	p := writeTemp(t, "beta.yaml", crd)
	err := newCRDValidator().Lint(p)
	if err == nil || !strings.Contains(err.Error(), "v1beta1 is not supported") {
		t.Fatalf("expected v1beta1 rejection, got: %v", err)
	}
}

func TestCRDBreaking_Breaking(t *testing.T) {
	newCRD := `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: widgets.example.com
spec:
  group: example.com
  names: {kind: Widget, plural: widgets, singular: widget}
  scope: Namespaced
  versions:
    - name: v1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                size: {type: integer, maximum: 50}
                label: {type: string, maxLength: 32}
              required: [size, label]
`
	oldP := writeTemp(t, "old.yaml", validCRD)
	newP := writeTemp(t, "new.yaml", newCRD)
	err := newCRDValidator().Breaking(newP, oldP)
	if err == nil {
		t.Fatal("expected breaking change to be flagged")
	}
	for _, want := range []string{
		"color: field was removed",
		"label: became required",
		"maxLength tightened",
		"size: maximum tightened",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("breaking output missing %q; got:\n%s", want, err.Error())
		}
	}
}

func TestCRDBreaking_EnumRemovedAndTypeChange(t *testing.T) {
	newCRD := strings.Replace(validCRD,
		"                  enum: [red, green, blue]",
		"                  enum: [red]", 1)
	newCRD = strings.Replace(newCRD,
		"                size:\n                  type: integer",
		"                size:\n                  type: string", 1)
	oldP := writeTemp(t, "old.yaml", validCRD)
	newP := writeTemp(t, "new.yaml", newCRD)
	err := newCRDValidator().Breaking(newP, oldP)
	if err == nil {
		t.Fatal("expected breaking change")
	}
	if !strings.Contains(err.Error(), "enum value green was removed") ||
		!strings.Contains(err.Error(), "enum value blue was removed") {
		t.Errorf("expected removed enum values; got:\n%s", err.Error())
	}
	if !strings.Contains(err.Error(), `type changed from "integer" to "string"`) {
		t.Errorf("expected type change; got:\n%s", err.Error())
	}
}

func TestCRDBreaking_Compatible(t *testing.T) {
	// Additive: new optional field, relaxed maximum, larger maxLength.
	newCRD := `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: widgets.example.com
spec:
  group: example.com
  names: {kind: Widget, plural: widgets, singular: widget}
  scope: Namespaced
  versions:
    - name: v1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                size: {type: integer, maximum: 200}
                color: {type: string, enum: [red, green, blue]}
                label: {type: string, maxLength: 128}
                weight: {type: number}
              required: [size, color]
`
	oldP := writeTemp(t, "old.yaml", validCRD)
	newP := writeTemp(t, "new.yaml", newCRD)
	if err := newCRDValidator().Breaking(newP, oldP); err != nil {
		t.Fatalf("expected compatible change to pass, got: %v", err)
	}
}

func TestCRDBreaking_AlphaExempt(t *testing.T) {
	// The same breaking change on an alpha version must NOT be flagged: alpha
	// versions carry no compatibility guarantee under the Kubernetes policy.
	oldAlpha := strings.Replace(validCRD, "name: v1", "name: v1alpha1", 1)
	newAlpha := strings.Replace(oldAlpha,
		"                color:\n                  type: string\n                  enum: [red, green, blue]\n", "", 1)
	oldP := writeTemp(t, "old.yaml", oldAlpha)
	newP := writeTemp(t, "new.yaml", newAlpha)
	if err := newCRDValidator().Breaking(newP, oldP); err != nil {
		t.Fatalf("alpha breaking change should be exempt, got: %v", err)
	}
}

func TestCRDBreaking_RemovedServedVersion(t *testing.T) {
	oldCRD := `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata: {name: widgets.example.com}
spec:
  group: example.com
  names: {kind: Widget, plural: widgets, singular: widget}
  scope: Namespaced
  versions:
    - name: v1
      served: true
      storage: false
      schema: {openAPIV3Schema: {type: object}}
    - name: v2
      served: true
      storage: true
      schema: {openAPIV3Schema: {type: object}}
`
	newCRD := `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata: {name: widgets.example.com}
spec:
  group: example.com
  names: {kind: Widget, plural: widgets, singular: widget}
  scope: Namespaced
  versions:
    - name: v2
      served: true
      storage: true
      schema: {openAPIV3Schema: {type: object}}
`
	oldP := writeTemp(t, "old.yaml", oldCRD)
	newP := writeTemp(t, "new.yaml", newCRD)
	err := newCRDValidator().Breaking(newP, oldP)
	if err == nil || !strings.Contains(err.Error(), "served version v1 was removed") {
		t.Fatalf("expected removed-served-version breaking, got: %v", err)
	}
}

func TestCRDBreaking_PreserveUnknownRemoved(t *testing.T) {
	oldCRD := `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata: {name: widgets.example.com}
spec:
  group: example.com
  names: {kind: Widget, plural: widgets, singular: widget}
  scope: Namespaced
  versions:
    - name: v1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            data:
              type: object
              x-kubernetes-preserve-unknown-fields: true
`
	newCRD := `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata: {name: widgets.example.com}
spec:
  group: example.com
  names: {kind: Widget, plural: widgets, singular: widget}
  scope: Namespaced
  versions:
    - name: v1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            data:
              type: object
              properties:
                known: {type: string}
`
	oldP := writeTemp(t, "old.yaml", oldCRD)
	newP := writeTemp(t, "new.yaml", newCRD)
	err := newCRDValidator().Breaking(newP, oldP)
	if err == nil || !strings.Contains(err.Error(), "x-kubernetes-preserve-unknown-fields was removed") {
		t.Fatalf("expected preserve-unknown removal breaking, got: %v", err)
	}
}

func TestCRDBreaking_NoBaseline(t *testing.T) {
	newP := writeTemp(t, "new.yaml", validCRD)
	if err := newCRDValidator().Breaking(newP, filepath.Join(t.TempDir(), "missing.yaml")); err != nil {
		t.Fatalf("missing baseline should be treated as first release, got: %v", err)
	}
}

func TestLooksLikeCRD(t *testing.T) {
	if !LooksLikeCRD([]byte(validCRD)) {
		t.Error("valid CRD not recognized")
	}
	if LooksLikeCRD([]byte("apiVersion: v1\nkind: ConfigMap\n")) {
		t.Error("ConfigMap wrongly recognized as CRD")
	}
	if LooksLikeCRD([]byte("not: yaml: [")) {
		t.Error("garbage wrongly recognized as CRD")
	}
}

func TestDetectFormat_CRDContent(t *testing.T) {
	p := writeTemp(t, "anything.yaml", validCRD)
	if got := DetectFormat(p); got != FormatCRD {
		t.Errorf("content sniff: got %v, want crd", got)
	}
}

func TestDetectFormat_CRDDir(t *testing.T) {
	// A non-CRD yaml under a crd/ directory still resolves to crd.
	dir := filepath.Join(t.TempDir(), "crd", "example.com", "widget", "v1")
	_ = os.MkdirAll(dir, 0o755)
	p := filepath.Join(dir, "widget.yaml")
	_ = os.WriteFile(p, []byte("kind: Other\n"), 0o644)
	if got := detectFormatFromFile(p); got != FormatCRD {
		t.Errorf("dir heuristic: got %v, want crd", got)
	}
}

func TestIsCRDVersion(t *testing.T) {
	cases := map[string]bool{
		"v1": true, "v2": true, "v10": true,
		"v1alpha1": true, "v1beta1": true, "v2beta3": true,
		"v0": false, "v1alpha": false, "valpha1": false,
		"v1.0": false, "1": false, "": false,
	}
	for in, want := range cases {
		if got := IsCRDVersion(in); got != want {
			t.Errorf("IsCRDVersion(%q)=%v, want %v", in, got, want)
		}
	}
}

func TestExtractCRDInfo(t *testing.T) {
	crd := `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata: {name: widgets.example.com}
spec:
  group: example.com
  names: {kind: Widget, plural: widgets, singular: widget}
  scope: Cluster
  versions:
    - name: v1alpha1
      served: true
      storage: false
      schema: {openAPIV3Schema: {type: object}}
    - name: v1
      served: true
      storage: true
      schema: {openAPIV3Schema: {type: object}}
`
	p := writeTemp(t, "c.yaml", crd)
	info, err := ExtractCRDInfo(p)
	if err != nil {
		t.Fatalf("ExtractCRDInfo: %v", err)
	}
	if info.Group != "example.com" || info.Kind != "Widget" || info.Scope != "Cluster" {
		t.Errorf("unexpected GVK: %+v", info)
	}
	if info.StorageVersion != "v1" {
		t.Errorf("storage version: got %q want v1", info.StorageVersion)
	}
	if len(info.ServedVersions) != 2 {
		t.Errorf("served versions: got %v", info.ServedVersions)
	}
}

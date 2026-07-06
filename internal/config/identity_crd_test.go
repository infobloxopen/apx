package config

import "testing"

func TestParseAPIID_CRD(t *testing.T) {
	api, err := ParseAPIID("crd/appkit.infoblox.dev/appcontract/v1alpha1")
	if err != nil {
		t.Fatalf("ParseAPIID crd: %v", err)
	}
	if api.Format != "crd" {
		t.Errorf("format: got %q want crd", api.Format)
	}
	if api.Domain != "appkit.infoblox.dev" {
		t.Errorf("domain: got %q want appkit.infoblox.dev", api.Domain)
	}
	if api.Name != "appcontract" {
		t.Errorf("name: got %q want appcontract", api.Name)
	}
	if api.Line != "v1alpha1" {
		t.Errorf("line: got %q want v1alpha1", api.Line)
	}
}

func TestLineMajor_K8s(t *testing.T) {
	cases := map[string]int{
		"v1": 1, "v2": 2, "v10": 10, "v0": 0,
		"v1alpha1": 1, "v1beta2": 1, "v2beta3": 2,
	}
	for line, want := range cases {
		got, err := LineMajor(line)
		if err != nil {
			t.Errorf("LineMajor(%q) error: %v", line, err)
			continue
		}
		if got != want {
			t.Errorf("LineMajor(%q)=%d want %d", line, got, want)
		}
	}
	if _, err := LineMajor("valpha1"); err == nil {
		t.Error("expected error for invalid line valpha1")
	}
}

func TestIsValidLine_K8s(t *testing.T) {
	valid := []string{"v0", "v1", "v2", "v1alpha1", "v1beta1", "v2beta3"}
	for _, l := range valid {
		if !isValidLine(l) {
			t.Errorf("isValidLine(%q) = false, want true", l)
		}
	}
	invalid := []string{"v", "v1alpha", "alpha1", "v1.0", "1"}
	for _, l := range invalid {
		if isValidLine(l) {
			t.Errorf("isValidLine(%q) = true, want false", l)
		}
	}
}

func TestDeriveTag_CRD(t *testing.T) {
	// One module per CRD version: the tag prefix keeps the full version segment.
	got := DeriveTag("crd/appkit.infoblox.dev/appcontract/v1alpha1", "v1.0.0-alpha.1")
	want := "crd/appkit.infoblox.dev/appcontract/v1alpha1/v1.0.0-alpha.1"
	if got != want {
		t.Errorf("DeriveTag = %q, want %q", got, want)
	}
}

func TestValidateVersionLine_CRD(t *testing.T) {
	// A CRD v1alpha1 line has major 1 → releases as v1.x.x.
	if err := ValidateVersionLine("v1.0.0-alpha.1", "v1alpha1"); err != nil {
		t.Errorf("v1.0.0-alpha.1 should match line v1alpha1: %v", err)
	}
	if err := ValidateVersionLine("v2.0.0", "v1alpha1"); err == nil {
		t.Error("v2.0.0 should NOT match line v1alpha1")
	}
	if err := ValidateVersionLine("v2.0.0", "v2beta1"); err != nil {
		t.Errorf("v2.0.0 should match line v2beta1: %v", err)
	}
}

package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestClientTargetYAMLRoundTrip(t *testing.T) {
	src := `version: 1
org: example
repo: apis
clients:
  - name: web
    generator: typescript-angular
    scope: "@example"
    package: notesd-client
    spec: openapi/notesd.openapi.yaml
    output: clients/web
`
	var cfg Config
	if err := yaml.Unmarshal([]byte(src), &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(cfg.Clients) != 1 {
		t.Fatalf("expected 1 client, got %d", len(cfg.Clients))
	}
	ct := cfg.Clients[0]
	if ct.Name != "web" || ct.Generator != "typescript-angular" || ct.Scope != "@example" ||
		ct.Package != "notesd-client" || ct.Spec != "openapi/notesd.openapi.yaml" || ct.Output != "clients/web" {
		t.Fatalf("unexpected client target: %+v", ct)
	}
}

func TestClientsAcceptedBySchema(t *testing.T) {
	src := []byte(`version: 1
org: example
repo: apis
clients:
  - name: web
    generator: typescript-angular
    scope: "@example"
    package: notesd-client
`)
	res, err := ValidateBytes(src)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if !res.Valid {
		t.Fatalf("expected valid config with clients, got errors: %v", res.Errors)
	}
}

func TestClientsOmittedStillValid(t *testing.T) {
	// Backward-compat: configs without a clients field remain valid.
	src := []byte(`version: 1
org: example
repo: apis
`)
	res, err := ValidateBytes(src)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if !res.Valid {
		t.Fatalf("expected valid config without clients, got errors: %v", res.Errors)
	}
}

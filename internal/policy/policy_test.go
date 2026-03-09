package policy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/infobloxopen/apx/internal/config"
)

func TestCheck_ForbiddenProtoOptions(t *testing.T) {
	dir := t.TempDir()
	// Write a proto file with a gorm option
	protoContent := `syntax = "proto3";
package test;

import "gorm/options/gorm.proto";

option go_package = "example.com/test";

message User {
  option (gorm.opts).ormable = true;
  string name = 1;
}
`
	if err := os.WriteFile(filepath.Join(dir, "test.proto"), []byte(protoContent), 0644); err != nil {
		t.Fatal(err)
	}

	pol := config.Policy{
		ForbiddenProtoOptions: []string{`^gorm\.`},
	}

	result, err := Check(pol, dir)
	if err != nil {
		t.Fatal(err)
	}

	if result.Passed() {
		t.Error("expected policy check to fail for forbidden gorm option, but it passed")
	}

	found := false
	for _, v := range result.Violations {
		if v.Rule == "forbidden_proto_option" {
			found = true
			t.Logf("violation: %s", v.Message)
		}
	}
	if !found {
		t.Error("expected forbidden_proto_option violation")
	}
}

func TestCheck_ForbiddenProtoOptions_Clean(t *testing.T) {
	dir := t.TempDir()
	protoContent := `syntax = "proto3";
package test;
option go_package = "example.com/test";
message User { string name = 1; }
`
	if err := os.WriteFile(filepath.Join(dir, "test.proto"), []byte(protoContent), 0644); err != nil {
		t.Fatal(err)
	}

	pol := config.Policy{
		ForbiddenProtoOptions: []string{`^gorm\.`},
	}

	result, err := Check(pol, dir)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Passed() {
		t.Errorf("expected policy check to pass; got violations: %v", result.Violations)
	}
}

func TestCheck_AllowedPlugins_Violation(t *testing.T) {
	dir := t.TempDir()
	genYAML := `version: v1
plugins:
  - plugin: protoc-gen-go
    out: gen/go
  - plugin: protoc-gen-grpc-gateway
    out: gen/go
`
	if err := os.WriteFile(filepath.Join(dir, "buf.gen.yaml"), []byte(genYAML), 0644); err != nil {
		t.Fatal(err)
	}
	// Also need a proto file so format detection works
	if err := os.WriteFile(filepath.Join(dir, "test.proto"), []byte("syntax = \"proto3\";\n"), 0644); err != nil {
		t.Fatal(err)
	}

	pol := config.Policy{
		AllowedProtoPlugins: []string{"protoc-gen-go", "protoc-gen-go-grpc"},
	}

	result, err := Check(pol, dir)
	if err != nil {
		t.Fatal(err)
	}
	if result.Passed() {
		t.Error("expected violation for protoc-gen-grpc-gateway (not in allowed list)")
	}
	found := false
	for _, v := range result.Violations {
		if v.Rule == "allowed_proto_plugin" {
			found = true
			t.Logf("violation: %s", v.Message)
		}
	}
	if !found {
		t.Error("expected allowed_proto_plugin violation")
	}
}

func TestCheck_AllowedPlugins_Clean(t *testing.T) {
	dir := t.TempDir()
	genYAML := `version: v1
plugins:
  - plugin: protoc-gen-go
    out: gen/go
  - plugin: protoc-gen-go-grpc
    out: gen/go
`
	if err := os.WriteFile(filepath.Join(dir, "buf.gen.yaml"), []byte(genYAML), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "test.proto"), []byte("syntax = \"proto3\";\n"), 0644); err != nil {
		t.Fatal(err)
	}

	pol := config.Policy{
		AllowedProtoPlugins: []string{"protoc-gen-go", "protoc-gen-go-grpc"},
	}

	result, err := Check(pol, dir)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Passed() {
		t.Errorf("expected no violations; got: %v", result.Violations)
	}
}

func TestCheck_AvroCompatibility_Invalid(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "test.avsc"), []byte(`{"type":"record","name":"Test","fields":[]}`), 0644); err != nil {
		t.Fatal(err)
	}

	pol := config.Policy{}
	pol.Avro.Compatibility = "INVALID_MODE"

	result, err := Check(pol, dir)
	if err != nil {
		t.Fatal(err)
	}
	if result.Passed() {
		t.Error("expected violation for invalid avro compatibility mode")
	}
}

func TestCheck_AvroCompatibility_Valid(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "test.avsc"), []byte(`{"type":"record","name":"Test","fields":[]}`), 0644); err != nil {
		t.Fatal(err)
	}

	pol := config.Policy{}
	pol.Avro.Compatibility = "BACKWARD"

	result, err := Check(pol, dir)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Passed() {
		t.Errorf("expected no violations for BACKWARD; got: %v", result.Violations)
	}
}

func TestCheck_JSONSchemaBreakingMode_Invalid(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "test.schema.json"), []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}

	pol := config.Policy{}
	pol.JSONSchema.BreakingMode = "chaos"

	result, err := Check(pol, dir)
	if err != nil {
		t.Fatal(err)
	}
	if result.Passed() {
		t.Error("expected violation for invalid breaking mode")
	}
}

func TestCheck_SpectralRuleset_Missing(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "openapi.yaml"), []byte("openapi: 3.0.0\n"), 0644); err != nil {
		t.Fatal(err)
	}

	pol := config.Policy{}
	pol.OpenAPI.SpectralRuleset = ".spectral.yaml"

	result, err := Check(pol, dir)
	if err != nil {
		t.Fatal(err)
	}
	if result.Passed() {
		t.Error("expected violation for missing spectral ruleset")
	}
}

func TestCheck_SpectralRuleset_Present(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "openapi.yaml"), []byte("openapi: 3.0.0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".spectral.yaml"), []byte("rules: {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	pol := config.Policy{}
	pol.OpenAPI.SpectralRuleset = ".spectral.yaml"

	result, err := Check(pol, dir)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Passed() {
		t.Errorf("expected no violations; got: %v", result.Violations)
	}
}

func TestCheck_EmptyPolicy(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "test.proto"), []byte("syntax = \"proto3\";\n"), 0644); err != nil {
		t.Fatal(err)
	}

	pol := config.Policy{}
	result, err := Check(pol, dir)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Passed() {
		t.Errorf("expected no violations for empty policy; got: %v", result.Violations)
	}
	if result.Checked != 0 {
		t.Errorf("expected 0 rules checked for empty policy; got %d", result.Checked)
	}
}

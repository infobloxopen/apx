// Package client provides API-client code generation for apx.
//
// A Generator turns an OpenAPI v3 spec into a packaged, buildable client
// (e.g. a TypeScript/Angular npm package). apx does not reimplement codegen;
// each generator orchestrates an external tool (such as ng-openapi-gen) and
// emits the package scaffolding around the generated output.
//
// Generators register themselves via init() into the package registry so that
// future targets (typescript-fetch, go, python) slot in without touching the
// command layer. This mirrors internal/language's plugin registry.
package client

import "context"

// GenerateContext carries the inputs a Generator needs to produce a client
// package for a single spec.
type GenerateContext struct {
	// SpecPath is the absolute-or-relative path to the OpenAPI v3 spec file.
	SpecPath string

	// OutputDir is the directory into which the packaged client is emitted.
	// Generated source lives under OutputDir/src; package scaffolding
	// (package.json, tsconfig.json, README.md) lives at OutputDir.
	OutputDir string

	// PackageName is the bare package name (e.g. "notesd-client"). It may
	// already include a scope (e.g. "@acme/notesd-client"), in which case the
	// Scope field is ignored.
	PackageName string

	// Scope is the npm scope to prepend when PackageName is unscoped
	// (e.g. "@example"). May be empty.
	Scope string

	// PackageVersion is the semantic version stamped into the package
	// (e.g. "0.0.1"). Defaults to "0.0.0" when empty.
	PackageVersion string
}

// Result reports what a Generator produced, enough to tell the user where the
// package lives and how to consume it.
type Result struct {
	// PackageDir is the directory containing the emitted package.
	PackageDir string

	// PackageName is the fully composed package name (including scope).
	PackageName string

	// Files lists the notable files written (package scaffolding), for
	// reporting. It need not enumerate every generated source file.
	Files []string
}

// Generator turns an OpenAPI spec into a packaged, buildable client.
type Generator interface {
	// Name returns the canonical key for this generator
	// (e.g. "typescript-angular").
	Name() string

	// Generate produces the client package described by ctx and returns a
	// Result describing what was written.
	Generate(ctx context.Context, gc GenerateContext) (Result, error)
}

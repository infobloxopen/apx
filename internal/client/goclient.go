package client

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/infobloxopen/apx/internal/publisher"
	"github.com/infobloxopen/apx/internal/ui"
)

// oapiCodegenVersion pins the oapi-codegen tool version. Overridable later;
// hardcoded for now (mirrors ngOpenAPIGenVersion for the angular generator).
const oapiCodegenVersion = "v2.4.1"

// oapiCodegenModule is the `go run` target for the pinned oapi-codegen tool.
// Running via `go run …@version` needs no pre-installed binary — the Go analog
// of the angular generator's `npx --yes`.
const oapiCodegenModule = "github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen"

func init() {
	Register(&goGenerator{})
}

// goGenerator produces a typed Go API client by orchestrating oapi-codegen and
// emitting a buildable Go module around the generated source. It implements
// Builder (go build) and Publisher (record a go-module artifact + document the
// git-tag publish), so `apx client generate/publish --generator go` compile-
// verifies and records without the npm toolchain.
//
// For this generator PackageName is the Go MODULE PATH (e.g.
// github.com/example/toy-client); Scope is npm-only and ignored.
type goGenerator struct{}

func (g *goGenerator) Name() string { return "go" }

// ToolchainAvailable reports whether the Go toolchain (needed for oapi-codegen
// via `go run` and for `go build`) is on PATH. Implements ToolchainChecker.
func (g *goGenerator) ToolchainAvailable() (bool, string) {
	if _, err := exec.LookPath("go"); err != nil {
		return false, "Go toolchain not found on PATH (install Go >=1.22)"
	}
	return true, ""
}

func (g *goGenerator) Generate(ctx context.Context, gc GenerateContext) (Result, error) {
	// Preflight: the generator shells out to `go run` for oapi-codegen.
	if _, err := exec.LookPath("go"); err != nil {
		return Result{}, fmt.Errorf("Go toolchain required for the go client generator; install Go >=1.22: %w", err)
	}

	specAbs, err := filepath.Abs(gc.SpecPath)
	if err != nil {
		return Result{}, fmt.Errorf("resolving spec path %q: %w", gc.SpecPath, err)
	}
	if _, err := os.Stat(specAbs); err != nil {
		return Result{}, fmt.Errorf("OpenAPI spec not found at %q: %w", specAbs, err)
	}

	modulePath := strings.TrimSpace(gc.PackageName)
	if modulePath == "" {
		return Result{}, fmt.Errorf("a module path is required for the go generator (pass --package github.com/you/your-client)")
	}
	goPkg := goPackageIdent(modulePath)

	if err := os.MkdirAll(gc.OutputDir, 0o755); err != nil {
		return Result{}, fmt.Errorf("creating output dir %q: %w", gc.OutputDir, err)
	}

	genFile := filepath.Join(gc.OutputDir, goPkg+".gen.go")
	cfgPath := filepath.Join(gc.OutputDir, ".oapi-codegen.yaml")
	if err := os.WriteFile(cfgPath, []byte(oapiCodegenConfig(goPkg, genFile)), 0o644); err != nil {
		return Result{}, fmt.Errorf("writing oapi-codegen config: %w", err)
	}

	// Orchestrate oapi-codegen. Unknown OpenAPI vendor extensions (e.g. the
	// x-aip-* keys from devedge-sdk's enriched spec) are ignored by oapi-codegen;
	// native readOnly/required/enum drive the generated Go types.
	ui.Info("Running oapi-codegen@%s ...", oapiCodegenVersion)
	pkgArg := oapiCodegenModule + "@" + oapiCodegenVersion
	cmd := exec.CommandContext(ctx, "go", "run", pkgArg, "-config", cfgPath, specAbs)
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return Result{}, fmt.Errorf("oapi-codegen failed: %w\n%s", err, tail(string(out), 40))
	}

	files := []string{genFile}

	gomodPath := filepath.Join(gc.OutputDir, "go.mod")
	if err := os.WriteFile(gomodPath, []byte(goModContent(modulePath)), 0o644); err != nil {
		return Result{}, fmt.Errorf("writing go.mod: %w", err)
	}
	files = append(files, gomodPath)

	docPath := filepath.Join(gc.OutputDir, "doc.go")
	if err := os.WriteFile(docPath, []byte(goDocContent(goPkg, modulePath)), 0o644); err != nil {
		return Result{}, fmt.Errorf("writing doc.go: %w", err)
	}
	files = append(files, docPath)

	readmePath := filepath.Join(gc.OutputDir, "README.md")
	if err := os.WriteFile(readmePath, []byte(goReadme(modulePath)), 0o644); err != nil {
		return Result{}, fmt.Errorf("writing README.md: %w", err)
	}
	files = append(files, readmePath)

	return Result{
		PackageDir:  gc.OutputDir,
		PackageName: modulePath,
		Files:       files,
	}, nil
}

// Build resolves the generated client's module deps and compiles it, proving the
// client is well-formed. Implements Builder — the command layer MUST NOT publish
// a package whose Build failed.
func (g *goGenerator) Build(ctx context.Context, res Result) error {
	if _, err := exec.LookPath("go"); err != nil {
		return fmt.Errorf("Go toolchain required to build the client: %w", err)
	}
	// Resolve deps (oapi-codegen runtime, etc.) the generated client imports.
	ui.Info("Running go mod tidy in %s ...", res.PackageDir)
	tidy := exec.CommandContext(ctx, "go", "mod", "tidy")
	tidy.Dir = res.PackageDir
	tidy.Env = os.Environ()
	if out, err := tidy.CombinedOutput(); err != nil {
		return fmt.Errorf("go mod tidy failed: %w\n%s", err, tail(string(out), 40))
	}
	ui.Info("Running go build ./... in %s ...", res.PackageDir)
	build := exec.CommandContext(ctx, "go", "build", "./...")
	build.Dir = res.PackageDir
	build.Env = os.Environ()
	if out, err := build.CombinedOutput(); err != nil {
		return fmt.Errorf("go build failed: %w\n%s", err, tail(string(out), 40))
	}
	return nil
}

// Publish records a go-module release artifact for the generated client and, for
// a real publish, documents the git-tag step. Go modules are published by tag in
// their own repo (not to a package registry), so with no repo context here the
// keystone bar is: honor DryRun and record the artifact — no write token needed.
// Implements Publisher.
func (g *goGenerator) Publish(ctx context.Context, res Result, opts PublishOptions) error {
	version := opts.Version
	if version == "" {
		version = "0.0.0"
	}
	status := "published"
	if opts.DryRun {
		status = "dry-run"
	}
	if opts.RecordPath != "" {
		rec, err := publisher.ReadReleaseRecord(opts.RecordPath)
		if err != nil {
			return fmt.Errorf("reading release record %q: %w", opts.RecordPath, err)
		}
		rec.AddArtifact("go-module", res.PackageName, version, status)
		if err := publisher.WriteReleaseRecord(rec, opts.RecordPath); err != nil {
			return fmt.Errorf("updating release record %q: %w", opts.RecordPath, err)
		}
		ui.Info("Recorded go-module artifact %s in %s", res.PackageName, opts.RecordPath)
	}
	if opts.DryRun {
		ui.Success("Validated go module %s (dry-run) at %s — pass --dry-run=false to record for publication", res.PackageName, res.PackageDir)
		return nil
	}
	ui.Info("Go modules publish by git tag in their own repo. Commit %s to its module repo, then:", res.PackageDir)
	ui.Info("    git tag %s && git push origin %s", version, version)
	ui.Success("Recorded go module %s @ %s for publication", res.PackageName, version)
	return nil
}

// goPackageIdent derives a valid Go package identifier from the last element of
// a module path: "github.com/example/toy-client" -> "toyclient".
func goPackageIdent(modulePath string) string {
	base := modulePath
	if i := strings.LastIndex(base, "/"); i >= 0 {
		base = base[i+1:]
	}
	var b strings.Builder
	for _, r := range strings.ToLower(base) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	s := b.String()
	if s == "" || (s[0] >= '0' && s[0] <= '9') {
		s = "client" + s
	}
	return s
}

// oapiCodegenConfig renders an oapi-codegen v2 config that emits models + a typed
// client into a single file. The generated file's package is pkg.
func oapiCodegenConfig(pkg, outFile string) string {
	return fmt.Sprintf(`# Generated by apx client generate --generator go. Do not edit.
package: %s
output: %s
generate:
  models: true
  client: true
`, pkg, outFile)
}

// goModContent renders a minimal go.mod. Requires are resolved by `go mod tidy`
// during Build (the generated client imports the oapi-codegen runtime).
func goModContent(modulePath string) string {
	return fmt.Sprintf("module %s\n\ngo 1.22\n", modulePath)
}

func goDocContent(pkg, modulePath string) string {
	return fmt.Sprintf(`// Package %s is a generated, typed Go API client for %s.
//
// Generated by "apx client generate --generator go", which orchestrates
// oapi-codegen. Do not edit by hand — regenerate with apx.
package %s
`, pkg, modulePath, pkg)
}

func goReadme(modulePath string) string {
	return fmt.Sprintf(`# %s

Generated, typed Go API client.

Produced by `+"`apx client generate --generator go`"+`, which orchestrates
[`+"`oapi-codegen`"+`](https://github.com/oapi-codegen/oapi-codegen) from an
enriched OpenAPI v3 spec and wraps the output in a buildable Go module.

## Build

`+"```sh"+`
go mod tidy
go build ./...
`+"```"+`

## Consume

`+"```sh"+`
go get %s
`+"```"+`

> Generated code — do not edit by hand. Regenerate with `+"`apx client generate`"+`.
`, modulePath, modulePath)
}

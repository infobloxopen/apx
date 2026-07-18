package client

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/infobloxopen/apx/internal/ui"
)

// DefaultVerifyGenerators is the generator matrix Verify runs when a caller does
// not specify one. Go always runs — its toolchain ships with apx's release
// environment and it catches Go-specific codegen hazards (identifier collisions,
// reserved-name params). typescript-angular runs where a Node toolchain is
// present and is skipped otherwise (see ToolchainChecker).
var DefaultVerifyGenerators = []string{"go", "typescript-angular"}

// defaultVerifyPackage is the throwaway package/module name stamped into a
// client generated purely to compile-verify a spec.
const defaultVerifyPackage = "apx-verify-client"

// ToolchainChecker is an optional interface a Generator implements to report
// whether the external toolchain it orchestrates is available. Verify SKIPS
// (rather than fails) a generator whose toolchain is absent, so a bare
// `apx client verify` covers Go always and TypeScript "where a toolchain is
// present". Mirrors the optional Builder/Publisher interfaces.
type ToolchainChecker interface {
	// ToolchainAvailable reports whether the generator can run in this
	// environment. When false, reason explains what is missing.
	ToolchainAvailable() (ok bool, reason string)
}

// VerifyOptions configures a client-verification run.
type VerifyOptions struct {
	// SpecPath is the OpenAPI v3 spec to verify (required).
	SpecPath string
	// Generators names the generators to verify; empty => DefaultVerifyGenerators.
	Generators []string
	// PackageName is the package/module name stamped into each generated client;
	// empty => defaultVerifyPackage.
	PackageName string
	// Scope is the npm scope for generators that use one (ignored by go).
	Scope string
	// WorkDir is the parent directory for throwaway per-generator output. When
	// empty, Verify uses a temp dir it removes before returning.
	WorkDir string
}

// GeneratorResult is the outcome of verifying a single generator.
type GeneratorResult struct {
	Generator string
	OK        bool   // generated AND compiled
	Skipped   bool   // toolchain unavailable — not a gate failure
	Reason    string // human-readable skip reason
	Err       error  // generate/build error when !OK && !Skipped
}

// VerifyReport aggregates the per-generator results of a run.
type VerifyReport struct {
	Results []GeneratorResult
}

// Failed reports whether any verified (non-skipped) generator produced a client
// that did not generate+compile. This is the release-gate signal.
func (r VerifyReport) Failed() bool {
	for _, res := range r.Results {
		if !res.Skipped && !res.OK {
			return true
		}
	}
	return false
}

// Ran reports how many generators were actually verified (not skipped). Callers
// use it to warn when every generator was skipped, so an all-skipped run does
// not masquerade as a passing gate.
func (r VerifyReport) Ran() int {
	n := 0
	for _, res := range r.Results {
		if !res.Skipped {
			n++
		}
	}
	return n
}

// Verify generates a client for each requested generator and compiles it,
// proving the spec produces a buildable client. Each generator runs in its own
// throwaway directory. Generate/compile failures are collected per generator
// (see VerifyReport.Failed) so a single run reports every broken generator; a
// genuinely unusable request — a missing spec or an unknown generator name —
// returns an error directly.
func Verify(ctx context.Context, opts VerifyOptions) (VerifyReport, error) {
	var report VerifyReport

	if strings.TrimSpace(opts.SpecPath) == "" {
		return report, fmt.Errorf("a spec path is required to verify a client")
	}
	if _, err := os.Stat(opts.SpecPath); err != nil {
		return report, fmt.Errorf("OpenAPI spec not found at %q: %w", opts.SpecPath, err)
	}

	gens := opts.Generators
	if len(gens) == 0 {
		gens = DefaultVerifyGenerators
	}

	pkg := strings.TrimSpace(opts.PackageName)
	if pkg == "" {
		pkg = defaultVerifyPackage
	}

	base := opts.WorkDir
	if base == "" {
		tmp, err := os.MkdirTemp("", "apx-client-verify-")
		if err != nil {
			return report, fmt.Errorf("creating temp work dir: %w", err)
		}
		defer os.RemoveAll(tmp)
		base = tmp
	}

	for _, name := range gens {
		gen := Get(name)
		if gen == nil {
			return report, fmt.Errorf("unknown client generator %q; available: %s",
				name, strings.Join(Names(), ", "))
		}
		report.Results = append(report.Results, verifyGenerator(ctx, gen, GenerateContext{
			SpecPath:    opts.SpecPath,
			OutputDir:   filepath.Join(base, name),
			PackageName: pkg,
			Scope:       opts.Scope,
		}))
	}
	return report, nil
}

// verifyGenerator generates and compiles one client, classifying the outcome.
func verifyGenerator(ctx context.Context, gen Generator, gc GenerateContext) GeneratorResult {
	name := gen.Name()
	if tc, ok := gen.(ToolchainChecker); ok {
		if avail, reason := tc.ToolchainAvailable(); !avail {
			ui.Warning("Skipping %s client verification: %s", name, reason)
			return GeneratorResult{Generator: name, Skipped: true, Reason: reason}
		}
	}

	ui.Info("Verifying %s client ...", name)
	res, err := gen.Generate(ctx, gc)
	if err != nil {
		return GeneratorResult{Generator: name, Err: fmt.Errorf("generate: %w", err)}
	}
	if err := Build(ctx, gen, res); err != nil {
		return GeneratorResult{Generator: name, Err: fmt.Errorf("build: %w", err)}
	}
	ui.Success("%s client generated and compiled", name)
	return GeneratorResult{Generator: name, OK: true}
}

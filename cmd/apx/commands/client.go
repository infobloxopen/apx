package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/infobloxopen/apx/internal/client"
	"github.com/infobloxopen/apx/internal/config"
	"github.com/infobloxopen/apx/internal/publisher"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const defaultClientGenerator = "typescript-angular"

func newClientCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "client",
		Short: "Generate and manage API clients",
		Long:  "Generate, build, and publish packaged API clients from OpenAPI specs.",
	}
	cmd.AddCommand(newClientGenerateCmd())
	cmd.AddCommand(newClientPublishCmd())
	cmd.AddCommand(newClientVerifyCmd())
	return cmd
}

// addClientResolutionFlags registers the flags shared by `generate` and
// `publish` that select and shape the client package.
func addClientResolutionFlags(cmd *cobra.Command) {
	cmd.Flags().String("input", "", "OpenAPI spec path (overrides config/auto-detect)")
	cmd.Flags().String("from", "", "api-id of an apx.lock dependency to source the spec from (unreleased override)")
	cmd.Flags().String("output", "", "output directory for the generated package")
	cmd.Flags().String("scope", "", "npm scope for the package (e.g. @example)")
	cmd.Flags().String("package", "", "generated package name")
	cmd.Flags().String("generator", defaultClientGenerator, "client generator to use")
	cmd.Flags().String("version", "", "package version to stamp (default 0.0.0)")
}

func newClientGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate [target]",
		Short: "Generate an API client package from an OpenAPI spec",
		Long: fmt.Sprintf("Generate a packaged, buildable API client from an OpenAPI v3 spec.\n"+
			"Available generators: %s", strings.Join(client.Names(), ", ")),
		Args: cobra.MaximumNArgs(1),
		RunE: clientGenerateAction,
	}
	addClientResolutionFlags(cmd)
	cmd.Flags().Bool("build", false, "run npm install + npm run build in the output dir after generation")
	cmd.Flags().Bool("clean", false, "remove the output directory before generation")
	return cmd
}

func newClientPublishCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "publish [target]",
		Short: "Generate, build, and publish an API client package",
		Long: "Generate a client package from an OpenAPI v3 spec, build it, and publish it\n" +
			"as a consumable npm module (GitHub Packages by the package's publishConfig).\n\n" +
			"Publishing requires npm auth for the target registry (e.g. a GITHUB_TOKEN with\n" +
			"packages:write in CI). By default this runs `npm publish --dry-run`, which\n" +
			"validates the package and shows what would ship without publishing; pass\n" +
			"--no-dry-run to publish for real.",
		Args: cobra.MaximumNArgs(1),
		RunE: clientPublishAction,
	}
	addClientResolutionFlags(cmd)
	cmd.Flags().Bool("dry-run", true, "validate + show the tarball without publishing (default); --dry-run=false to publish")
	cmd.Flags().String("record", "", "append an npm-package artifact to this apx release record (optional)")
	return cmd
}

func newClientVerifyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify [target]",
		Short: "Generate and compile a client to prove a spec is buildable",
		Long: fmt.Sprintf("Generate an API client for one or more generators and compile it, failing the\n"+
			"command if any client does not build. This is the release gate that stops a spec\n"+
			"which cannot produce a buildable client from being published (a valid OpenAPI\n"+
			"spec can still generate Go that does not compile).\n\n"+
			"By default it verifies the %s generators; TypeScript is skipped where its Node\n"+
			"toolchain is absent. Use --generator to scope, and --warn-only to downgrade a\n"+
			"generate/compile failure to a warning (exit 0) instead of failing the gate.\n\n"+
			"Available generators: %s",
			strings.Join(client.DefaultVerifyGenerators, ", "), strings.Join(client.Names(), ", ")),
		Args: cobra.MaximumNArgs(1),
		RunE: clientVerifyAction,
	}
	cmd.Flags().String("input", "", "OpenAPI spec path (overrides config/auto-detect)")
	cmd.Flags().String("from", "", "api-id of an apx.lock dependency to source the spec from (unreleased override)")
	cmd.Flags().StringSlice("generator", nil, "generator(s) to verify (repeatable); default: release.verify_clients.generators, else go + typescript-angular")
	cmd.Flags().String("package", "", "package/module name stamped into the generated client")
	cmd.Flags().String("scope", "", "npm scope for the package (e.g. @example)")
	cmd.Flags().Bool("warn-only", false, "downgrade a generate/compile failure to a warning and exit 0 (default: fail the gate)")
	return cmd
}

// clientVerifyAction resolves a spec and generator matrix, generates+compiles a
// client per generator, and turns the aggregate result into an exit code: it
// fails when any verified client does not build, unless --warn-only (or the
// release.verify_clients.warn_only config default) downgrades that to a warning.
func clientVerifyAction(cmd *cobra.Command, args []string) error {
	target := ""
	if len(args) > 0 {
		target = args[0]
	}
	input, _ := cmd.Flags().GetString("input")
	from, _ := cmd.Flags().GetString("from")
	pkg, _ := cmd.Flags().GetString("package")
	scope, _ := cmd.Flags().GetString("scope")
	generators, _ := cmd.Flags().GetStringSlice("generator")

	// Best-effort config load; a valid apx.yaml is not required (e.g. with --input).
	cfg, _ := config.LoadRaw("")

	var ct *config.ClientTarget
	if target != "" {
		ct = findClientTarget(cfg, target)
		if ct == nil {
			return fmt.Errorf("no client target %q found in apx.yaml", target)
		}
		if scope == "" {
			scope = ct.Scope
		}
		if pkg == "" {
			pkg = ct.Package
		}
	}

	specPath, err := resolveClientSpec(input, from, ct, target)
	if err != nil {
		return err
	}

	// Generator matrix precedence: flag > config > built-in default (the empty
	// slice is resolved to client.DefaultVerifyGenerators inside Verify).
	if len(generators) == 0 && cfg != nil {
		generators = cfg.Release.VerifyClients.Generators
	}

	// warn-only precedence: an explicit flag wins; otherwise the config default.
	warnOnly, _ := cmd.Flags().GetBool("warn-only")
	if !cmd.Flags().Changed("warn-only") && cfg != nil {
		warnOnly = cfg.Release.VerifyClients.WarnOnly
	}

	ui.Info("Verifying client build for %s ...", specPath)
	report, err := client.Verify(cmd.Context(), client.VerifyOptions{
		SpecPath:    specPath,
		Generators:  generators,
		PackageName: pkg,
		Scope:       scope,
	})
	if err != nil {
		return fmt.Errorf("verifying client: %w", err)
	}

	for _, r := range report.Results {
		switch {
		case r.Skipped:
			ui.Warning("  %s: skipped (%s)", r.Generator, r.Reason)
		case r.OK:
			ui.Success("  %s: ok", r.Generator)
		default:
			ui.Error("  %s: FAILED — %v", r.Generator, r.Err)
		}
	}

	if report.Ran() == 0 {
		ui.Warning("No generators were verified (all skipped). Install the required toolchain(s) to run the gate.")
		return nil
	}

	if report.Failed() {
		if warnOnly {
			ui.Warning("Client verification failed, but --warn-only is set — not failing the gate.")
			return nil
		}
		return fmt.Errorf("client verification failed: one or more generated clients did not build")
	}

	ui.Success("All verified clients built successfully")
	return nil
}

// resolveClientContext applies the shared target/flag/spec resolution and
// returns the selected generator plus the fully-resolved GenerateContext.
//
// Field precedence is flag > named target (apx.yaml clients:) > default.
// Spec precedence is --input > --from/target.From > target.Spec >
// auto-detected openapi/*.openapi.yaml.
func resolveClientContext(cmd *cobra.Command, args []string) (client.Generator, client.GenerateContext, error) {
	target := ""
	if len(args) > 0 {
		target = args[0]
	}

	input, _ := cmd.Flags().GetString("input")
	from, _ := cmd.Flags().GetString("from")
	output, _ := cmd.Flags().GetString("output")
	scope, _ := cmd.Flags().GetString("scope")
	pkg, _ := cmd.Flags().GetString("package")
	generatorName, _ := cmd.Flags().GetString("generator")
	version, _ := cmd.Flags().GetString("version")

	// Best-effort config load; a valid apx.yaml is not required to generate a
	// client (e.g. when --input is given explicitly).
	cfg, _ := config.LoadRaw("")

	var ct *config.ClientTarget
	if target != "" {
		ct = findClientTarget(cfg, target)
		if ct == nil {
			return nil, client.GenerateContext{}, fmt.Errorf("no client target %q found in apx.yaml", target)
		}
	}

	if ct != nil {
		if generatorName == defaultClientGenerator && ct.Generator != "" {
			generatorName = ct.Generator
		}
		if scope == "" {
			scope = ct.Scope
		}
		if pkg == "" {
			pkg = ct.Package
		}
		if output == "" {
			output = ct.Output
		}
	}

	specPath, err := resolveClientSpec(input, from, ct, target)
	if err != nil {
		return nil, client.GenerateContext{}, err
	}

	if output == "" {
		if pkg != "" {
			output = pkg
		} else {
			output = "client"
		}
	}
	if pkg == "" {
		pkg = defaultPackageName(specPath)
	}

	gen := client.Get(generatorName)
	if gen == nil {
		return nil, client.GenerateContext{}, fmt.Errorf("unknown client generator %q; available: %s",
			generatorName, strings.Join(client.Names(), ", "))
	}

	return gen, client.GenerateContext{
		SpecPath:       specPath,
		OutputDir:      output,
		PackageName:    pkg,
		Scope:          scope,
		PackageVersion: version,
	}, nil
}

func clientGenerateAction(cmd *cobra.Command, args []string) error {
	doBuild, _ := cmd.Flags().GetBool("build")
	doClean, _ := cmd.Flags().GetBool("clean")

	gen, gc, err := resolveClientContext(cmd, args)
	if err != nil {
		return err
	}

	if doClean && gc.OutputDir != "" {
		if err := os.RemoveAll(gc.OutputDir); err != nil {
			return fmt.Errorf("cleaning output dir %q: %w", gc.OutputDir, err)
		}
	}

	ui.Info("Generating %s client from %s ...", gen.Name(), gc.SpecPath)
	res, err := gen.Generate(cmd.Context(), gc)
	if err != nil {
		return fmt.Errorf("generating client: %w", err)
	}

	if doBuild {
		if err := client.Build(cmd.Context(), gen, res); err != nil {
			return fmt.Errorf("building client package: %w", err)
		}
	}

	ui.Success("Generated %s at %s", res.PackageName, res.PackageDir)
	ui.Info("Consume it locally with a file: dependency: \"%s\": \"file:%s\"",
		res.PackageName, res.PackageDir)
	return nil
}

func clientPublishAction(cmd *cobra.Command, args []string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	recordPath, _ := cmd.Flags().GetString("record")

	gen, gc, err := resolveClientContext(cmd, args)
	if err != nil {
		return err
	}

	ui.Info("Generating %s client from %s ...", gen.Name(), gc.SpecPath)
	res, err := gen.Generate(cmd.Context(), gc)
	if err != nil {
		return fmt.Errorf("generating client: %w", err)
	}

	// Build first: a Builder-aware generator (e.g. go) compile-verifies; the npm
	// path builds dist/ which `npm publish` needs.
	if err := client.Build(cmd.Context(), gen, res); err != nil {
		return fmt.Errorf("building client package: %w", err)
	}

	// A Publisher-aware generator owns its publish + artifact record (e.g. the go
	// generator records a go-module artifact and documents the git-tag publish).
	if p, ok := gen.(client.Publisher); ok {
		return p.Publish(cmd.Context(), res, client.PublishOptions{
			DryRun:     dryRun,
			Version:    gc.PackageVersion,
			RecordPath: recordPath,
		})
	}

	// Default (npm) publish path — unchanged.
	if err := publishClientPackage(res.PackageDir, dryRun); err != nil {
		return fmt.Errorf("publishing client package: %w", err)
	}

	status := "published"
	if dryRun {
		status = "dry-run"
	}

	if recordPath != "" {
		if err := recordClientArtifact(recordPath, res.PackageName, gc.PackageVersion, status); err != nil {
			return err
		}
		ui.Info("Recorded npm-package artifact %s in %s", res.PackageName, recordPath)
	}

	if dryRun {
		ui.Success("Validated %s (dry-run) at %s — pass --dry-run=false to publish", res.PackageName, res.PackageDir)
	} else {
		ui.Success("Published %s from %s", res.PackageName, res.PackageDir)
	}
	return nil
}

// findClientTarget returns the ClientTarget with the given name from config, or nil.
func findClientTarget(cfg *config.Config, name string) *config.ClientTarget {
	if cfg == nil {
		return nil
	}
	for i := range cfg.Clients {
		if cfg.Clients[i].Name == name {
			return &cfg.Clients[i]
		}
	}
	return nil
}

// resolveClientSpec applies the spec-resolution precedence:
//
//	--input flag >
//	  --from / target.From (unreleased dependency override in apx.lock) >
//	    named target's Spec >
//	      auto-detect openapi/*.openapi.yaml.
//
// Auto-detect errors if it finds zero or more than one spec (the caller should
// then pass --input or name a target).
func resolveClientSpec(input, from string, ct *config.ClientTarget, target string) (string, error) {
	if input != "" {
		return input, nil
	}

	// --from (or a target's from:) sources the spec from an apx.lock dependency
	// that carries an unreleased override. This is the consumer side of the
	// local hot-loop: build against an unreleased upstream API.
	if from == "" && ct != nil {
		from = ct.From
	}
	if from != "" {
		return resolveFromDependency(from)
	}

	if ct != nil && ct.Spec != "" {
		return ct.Spec, nil
	}

	matches, err := filepath.Glob(filepath.Join("openapi", "*.openapi.yaml"))
	if err != nil {
		return "", fmt.Errorf("scanning for OpenAPI specs: %w", err)
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return "", fmt.Errorf("no OpenAPI spec found (looked for openapi/*.openapi.yaml); pass --input or configure a client target")
	default:
		return "", fmt.Errorf("multiple OpenAPI specs found (%s); pass --input or name a specific target",
			strings.Join(matches, ", "))
	}
}

// resolveFromDependency resolves the OpenAPI spec for the apx.lock dependency
// identified by apiID (the --from value). It errors clearly when:
//   - apx.lock is absent or the dependency is not locked;
//   - the dependency has no unreleased override (released-dep spec resolution
//     from the OCI catalog is out of scope for this phase).
func resolveFromDependency(apiID string) (string, error) {
	lock, err := loadLockFile("apx.lock")
	if err != nil {
		return "", fmt.Errorf("--from %q: %w", apiID, err)
	}

	dep, ok := lock.Dependencies[apiID]
	if !ok {
		return "", fmt.Errorf("--from %q: no such dependency in apx.lock (add it with `apx add %s --path <dir>` or `--git <repo> --ref <ref>`)", apiID, apiID)
	}

	if !dep.IsOverride() {
		return "", fmt.Errorf("--from %q resolves to a released dependency (%s@%s); resolving specs from released catalog versions is not yet supported in this phase — pass --input or use an unreleased override (apx add %s --path/--git)", apiID, dep.Repo, dep.Ref, apiID)
	}

	specPath, cleanup, err := config.MaterializeSpec(dep, apiID)
	if err != nil {
		return "", fmt.Errorf("--from %q: %w", apiID, err)
	}
	// cleanup is a no-op for both path and git overrides (the git clone lives in
	// a persistent cache); the generator reads the spec synchronously, so there
	// is nothing to defer here.
	_ = cleanup
	ui.Info("Sourcing spec for %s from unreleased override → %s", apiID, specPath)
	return specPath, nil
}

// loadLockFile reads and parses an apx.lock file. A missing file yields an
// empty (non-nil) lock so callers can distinguish "no such dependency" from a
// read error.
func loadLockFile(path string) (*config.LockFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &config.LockFile{Dependencies: map[string]config.DependencyLock{}}, nil
		}
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var lf config.LockFile
	if err := yaml.Unmarshal(data, &lf); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if lf.Dependencies == nil {
		lf.Dependencies = map[string]config.DependencyLock{}
	}
	return &lf, nil
}

// defaultPackageName derives a package name from a spec filename, stripping
// known suffixes: e.g. "openapi/notesd.openapi.yaml" -> "notesd-client".
func defaultPackageName(specPath string) string {
	base := filepath.Base(specPath)
	base = strings.TrimSuffix(base, ".yaml")
	base = strings.TrimSuffix(base, ".yml")
	base = strings.TrimSuffix(base, ".openapi")
	base = strings.TrimSuffix(base, ".swagger")
	if base == "" {
		base = "api"
	}
	return base + "-client"
}

// publishClientPackage runs `npm publish` (or `npm publish --dry-run`) in dir.
// The package's publishConfig selects the registry (GitHub Packages); auth for
// a real publish is supplied by the environment (e.g. GITHUB_TOKEN in CI).
func publishClientPackage(dir string, dryRun bool) error {
	if _, err := exec.LookPath("npm"); err != nil {
		return fmt.Errorf("npm required to publish; install Node >=18: %w", err)
	}

	npmArgs := []string{"publish"}
	if dryRun {
		npmArgs = append(npmArgs, "--dry-run")
	}

	ui.Info("Running npm %s in %s ...", strings.Join(npmArgs, " "), dir)
	pub := exec.Command("npm", npmArgs...)
	pub.Dir = dir
	pub.Env = os.Environ()
	pub.Stdout = os.Stdout
	pub.Stderr = os.Stderr
	if err := pub.Run(); err != nil {
		return fmt.Errorf("npm publish failed: %w", err)
	}
	return nil
}

// recordClientArtifact appends an npm-package artifact to an existing apx
// release record so the release + catalog link the API version to its client.
func recordClientArtifact(recordPath, name, version, status string) error {
	rec, err := publisher.ReadReleaseRecord(recordPath)
	if err != nil {
		return fmt.Errorf("reading release record %q: %w", recordPath, err)
	}
	if version == "" {
		version = "0.0.0"
	}
	rec.AddArtifact("npm-package", name, version, status)
	if err := publisher.WriteReleaseRecord(rec, recordPath); err != nil {
		return fmt.Errorf("updating release record %q: %w", recordPath, err)
	}
	return nil
}

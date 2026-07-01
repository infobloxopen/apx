package client

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/infobloxopen/apx/internal/ui"
)

// ngOpenAPIGenVersion pins the ng-openapi-gen tool version. Overridable later;
// hardcoded for now.
const ngOpenAPIGenVersion = "1.0.5"

func init() {
	Register(&angularGenerator{})
}

// angularGenerator produces a TypeScript/Angular client by orchestrating the
// ng-openapi-gen npm tool and emitting npm package scaffolding around its output.
type angularGenerator struct{}

func (g *angularGenerator) Name() string { return "typescript-angular" }

func (g *angularGenerator) Generate(ctx context.Context, gc GenerateContext) (Result, error) {
	// Preflight: the generator shells out to npx.
	if _, err := exec.LookPath("npx"); err != nil {
		return Result{}, fmt.Errorf("Node.js/npx required for the typescript-angular client generator; install Node >=18: %w", err)
	}

	specAbs, err := filepath.Abs(gc.SpecPath)
	if err != nil {
		return Result{}, fmt.Errorf("resolving spec path %q: %w", gc.SpecPath, err)
	}
	if _, err := os.Stat(specAbs); err != nil {
		return Result{}, fmt.Errorf("OpenAPI spec not found at %q: %w", specAbs, err)
	}

	if err := os.MkdirAll(gc.OutputDir, 0o755); err != nil {
		return Result{}, fmt.Errorf("creating output dir %q: %w", gc.OutputDir, err)
	}
	srcDir := filepath.Join(gc.OutputDir, "src")

	// Run the generator. `--index-file true` emits an index.ts barrel that
	// re-exports the generated services, models, and functions.
	ui.Info("Running ng-openapi-gen@%s ...", ngOpenAPIGenVersion)
	pkgArg := fmt.Sprintf("ng-openapi-gen@%s", ngOpenAPIGenVersion)
	cmd := exec.CommandContext(ctx, "npx", "--yes", pkgArg,
		"--input", specAbs,
		"--output", srcDir,
		"--index-file", "true",
	)
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return Result{}, fmt.Errorf("ng-openapi-gen failed: %w\n%s", err, tail(string(out), 40))
	}

	// ng-openapi-gen's barrel re-exports the ApiConfiguration class but omits the
	// provideApiConfiguration(rootUrl) provider helper it also emits. Re-export it
	// so consumers can wire the base URL from the package entry point in one line.
	if err := ensureProviderExported(srcDir); err != nil {
		return Result{}, fmt.Errorf("augmenting client barrel: %w", err)
	}

	pkgName := composePackageName(gc.Scope, gc.PackageName)
	version := gc.PackageVersion
	if version == "" {
		version = "0.0.0"
	}

	// Emit package scaffolding into OutputDir (NOT src/).
	files := make([]string, 0, 3)
	pkgJSONPath := filepath.Join(gc.OutputDir, "package.json")
	if err := writePackageJSON(pkgJSONPath, pkgName, version); err != nil {
		return Result{}, fmt.Errorf("writing package.json: %w", err)
	}
	files = append(files, pkgJSONPath)

	tsconfigPath := filepath.Join(gc.OutputDir, "tsconfig.json")
	if err := os.WriteFile(tsconfigPath, []byte(tsconfigJSON), 0o644); err != nil {
		return Result{}, fmt.Errorf("writing tsconfig.json: %w", err)
	}
	files = append(files, tsconfigPath)

	readmePath := filepath.Join(gc.OutputDir, "README.md")
	if err := os.WriteFile(readmePath, []byte(renderReadme(pkgName)), 0o644); err != nil {
		return Result{}, fmt.Errorf("writing README.md: %w", err)
	}
	files = append(files, readmePath)

	return Result{
		PackageDir:  gc.OutputDir,
		PackageName: pkgName,
		Files:       files,
	}, nil
}

// composePackageName combines an npm scope and a bare package name.
//
//   - If packageName already starts with "@", it is used as-is (already scoped).
//   - Else if scope is non-empty, the result is "<scope>/<packageName>", with the
//     scope normalized to a single leading "@" and no trailing slash.
//   - Else the packageName is used unscoped.
func composePackageName(scope, packageName string) string {
	packageName = strings.TrimSpace(packageName)
	scope = strings.TrimSpace(scope)

	if strings.HasPrefix(packageName, "@") {
		return packageName
	}
	if scope == "" {
		return packageName
	}
	scope = strings.TrimSuffix(scope, "/")
	scope = "@" + strings.TrimPrefix(scope, "@")
	return scope + "/" + packageName
}

// Pinned devDependency versions used to typecheck/build the generated package.
// These are only build-time tooling; consumers rely on peerDependencies instead.
const (
	angularDevVersion    = "^15.2.10"
	rxjsDevVersion       = "^7.8.0"
	typescriptDevVersion = "^5.4.0"
	zoneDevVersion       = "~0.13.3"
)

// packageJSON mirrors the shape of @infobloxopen/devedge-ufe-angular. Fields are
// ordered to produce a stable, readable file.
type packageJSON struct {
	Name             string            `json:"name"`
	Version          string            `json:"version"`
	Description      string            `json:"description"`
	License          string            `json:"license"`
	Type             string            `json:"type"`
	Main             string            `json:"main"`
	Module           string            `json:"module"`
	Types            string            `json:"types"`
	Exports          map[string]any    `json:"exports"`
	Files            []string          `json:"files"`
	SideEffects      bool              `json:"sideEffects"`
	PublishConfig    map[string]string `json:"publishConfig"`
	Scripts          map[string]string `json:"scripts"`
	PeerDependencies map[string]string `json:"peerDependencies"`
	DevDependencies  map[string]string `json:"devDependencies"`
}

func writePackageJSON(path, name, version string) error {
	pj := packageJSON{
		Name:        name,
		Version:     version,
		Description: fmt.Sprintf("Generated TypeScript/Angular API client for %s.", name),
		License:     "Apache-2.0",
		Type:        "module",
		Main:        "./dist/index.js",
		Module:      "./dist/index.js",
		Types:       "./dist/index.d.ts",
		Exports: map[string]any{
			".": map[string]string{
				"types":  "./dist/index.d.ts",
				"import": "./dist/index.js",
			},
		},
		Files:       []string{"dist"},
		SideEffects: false,
		PublishConfig: map[string]string{
			"access":   "public",
			"registry": "https://npm.pkg.github.com",
		},
		Scripts: map[string]string{
			"build":     "tsc -p tsconfig.json",
			"typecheck": "tsc -p tsconfig.json --noEmit",
		},
		PeerDependencies: map[string]string{
			"@angular/core":   ">=15",
			"@angular/common": ">=15",
			"rxjs":            ">=7",
		},
		// devDependencies provide the build toolchain (tsc) and satisfy the
		// generated code's peer imports so `npm run build` can typecheck the
		// package in isolation. Consumers use peerDependencies instead.
		DevDependencies: map[string]string{
			"@angular/core":   angularDevVersion,
			"@angular/common": angularDevVersion,
			"rxjs":            rxjsDevVersion,
			"zone.js":         zoneDevVersion,
			"typescript":      typescriptDevVersion,
		},
	}
	data, err := json.MarshalIndent(pj, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

// tsconfigJSON is a standalone tsconfig for the generated Angular client. It
// does NOT extend anything so the package compiles in isolation, and it enables
// the decorator support the generated @Injectable services need.
const tsconfigJSON = `{
  "compilerOptions": {
    "target": "ES2021",
    "module": "ESNext",
    "moduleResolution": "Bundler",
    "lib": ["ES2021", "DOM", "DOM.Iterable"],
    "declaration": true,
    "declarationMap": true,
    "sourceMap": true,
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "forceConsistentCasingInFileNames": true,
    "experimentalDecorators": true,
    "emitDecoratorMetadata": true,
    "useDefineForClassFields": false,
    "rootDir": "./src",
    "outDir": "./dist"
  },
  "include": ["src/**/*.ts"],
  "exclude": ["dist", "node_modules"]
}
`

func renderReadme(pkgName string) string {
	return fmt.Sprintf(`# %s

Generated TypeScript/Angular API client.

This package was produced by `+"`apx client generate`"+`, which orchestrates
[`+"`ng-openapi-gen`"+`](https://www.npmjs.com/package/ng-openapi-gen) and wraps the
generated sources in a buildable npm package.

## Build

`+"```sh"+`
npm install
npm run build   # tsc -> dist/ (with .d.ts declarations)
`+"```"+`

## Consume locally

Add a `+"`file:`"+` dependency pointing at this package directory:

`+"```json"+`
{
  "dependencies": {
    "%s": "file:../path/to/this/package"
  }
}
`+"```"+`

The generated services import `+"`@angular/core`"+`, `+"`@angular/common/http`"+`, and
`+"`rxjs`"+` as peer dependencies; provide those in the consuming application.

> Generated code — do not edit by hand. Regenerate with `+"`apx client generate`"+`.
`, pkgName, pkgName)
}

// ensureProviderExported re-exports provideApiConfiguration from the generated
// index.ts barrel. ng-openapi-gen emits the provider helper in
// api-configuration.ts but does not add it to the barrel, so consumers using the
// package's public entry point cannot import it to wire the base URL. This is
// idempotent: a no-op when the barrel is absent (index-file disabled), when the
// export is already present, or when the generator did not emit the helper.
func ensureProviderExported(srcDir string) error {
	indexPath := filepath.Join(srcDir, "index.ts")
	idx, err := os.ReadFile(indexPath)
	if err != nil {
		return nil // no barrel to augment
	}
	if strings.Contains(string(idx), "provideApiConfiguration") {
		return nil // already exported
	}

	apiCfg, err := os.ReadFile(filepath.Join(srcDir, "api-configuration.ts"))
	if err != nil || !strings.Contains(string(apiCfg), "export function provideApiConfiguration") {
		return nil // generator did not emit the helper; nothing to re-export
	}

	f, err := os.OpenFile(indexPath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString("export { provideApiConfiguration } from './api-configuration';\n")
	return err
}

// tail returns the last n lines of s, useful for surfacing the end of a failed
// command's output without dumping everything.
func tail(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) <= n {
		return strings.Join(lines, "\n")
	}
	return strings.Join(lines[len(lines)-n:], "\n")
}

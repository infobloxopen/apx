package client

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/infobloxopen/apx/internal/ui"
)

// Build compile-verifies the generated package described by res. A Builder-aware
// generator (e.g. the go generator: `go mod tidy` + `go build`) supplies its own
// build; generators without a Builder (e.g. typescript-angular) use the npm build
// path. The command layer MUST NOT publish a package whose Build failed.
//
// Centralized here (rather than in the command layer) so `generate --build`,
// `publish`, and `verify` all compile-verify identically.
func Build(ctx context.Context, gen Generator, res Result) error {
	if b, ok := gen.(Builder); ok {
		return b.Build(ctx, res)
	}
	return npmBuild(res.PackageDir)
}

// npmBuild runs `npm install` then `npm run build` in dir to prove the generated
// package compiles. Output streams to the process stdio so build failures are
// visible verbatim.
func npmBuild(dir string) error {
	if _, err := exec.LookPath("npm"); err != nil {
		return fmt.Errorf("npm required to build the client; install Node >=18: %w", err)
	}

	ui.Info("Running npm install in %s ...", dir)
	install := exec.Command("npm", "install")
	install.Dir = dir
	install.Env = os.Environ()
	install.Stdout = os.Stdout
	install.Stderr = os.Stderr
	if err := install.Run(); err != nil {
		return fmt.Errorf("npm install failed: %w", err)
	}

	ui.Info("Running npm run build in %s ...", dir)
	build := exec.Command("npm", "run", "build")
	build.Dir = dir
	build.Env = os.Environ()
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		return fmt.Errorf("npm run build failed: %w", err)
	}
	return nil
}

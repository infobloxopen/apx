// Copyright 2025 Infoblox Inc.
// SPDX-License-Identifier: Apache-2.0

package testhelpers

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// APXCommand wraps APX CLI command execution
type APXCommand struct {
	WorkDir string
	Env     []string
}

// NewAPXCommand creates a new APX command wrapper
func NewAPXCommand(workDir string) *APXCommand {
	return &APXCommand{
		WorkDir: workDir,
		Env:     []string{},
	}
}

// WithEnv adds environment variables to the command
func (a *APXCommand) WithEnv(key, value string) *APXCommand {
	a.Env = append(a.Env, fmt.Sprintf("%s=%s", key, value))
	return a
}

// Init runs apx init command
func (a *APXCommand) Init(ctx context.Context, repoType, name string, extraArgs ...string) (string, error) {
	args := []string{"init", repoType, name}
	args = append(args, extraArgs...)
	return a.run(ctx, args...)
}

// Release runs apx release command
func (a *APXCommand) Release(ctx context.Context, extraArgs ...string) (string, error) {
	args := append([]string{"release"}, extraArgs...)
	return a.run(ctx, args...)
}

// Add runs apx add command to add a dependency
func (a *APXCommand) Add(ctx context.Context, dependency string, extraArgs ...string) (string, error) {
	args := append([]string{"add", dependency}, extraArgs...)
	return a.run(ctx, args...)
}

// Gen runs apx gen command to generate code
func (a *APXCommand) Gen(ctx context.Context, language string, extraArgs ...string) (string, error) {
	args := append([]string{"gen", language}, extraArgs...)
	return a.run(ctx, args...)
}

// Sync runs apx sync command
func (a *APXCommand) Sync(ctx context.Context, extraArgs ...string) (string, error) {
	args := append([]string{"sync"}, extraArgs...)
	return a.run(ctx, args...)
}

// Lint runs apx lint command
func (a *APXCommand) Lint(ctx context.Context, extraArgs ...string) (string, error) {
	args := append([]string{"lint"}, extraArgs...)
	return a.run(ctx, args...)
}

// Breaking runs apx breaking command
func (a *APXCommand) Breaking(ctx context.Context, extraArgs ...string) (string, error) {
	args := append([]string{"breaking"}, extraArgs...)
	return a.run(ctx, args...)
}

// run executes the apx command with given arguments
func (a *APXCommand) run(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "apx", args...)
	cmd.Dir = a.WorkDir
	cmd.Env = append(cmd.Env, a.Env...)

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		return outputStr, fmt.Errorf("apx %s failed: %w\nOutput: %s",
			strings.Join(args, " "), err, outputStr)
	}

	return outputStr, nil
}

// RunRaw runs an arbitrary apx command with raw arguments
func (a *APXCommand) RunRaw(ctx context.Context, args ...string) (string, error) {
	return a.run(ctx, args...)
}

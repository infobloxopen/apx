package main

import (
	"fmt"
	"os"

	"github.com/infobloxopen/apx/cmd/apx/commands"
	gh "github.com/infobloxopen/apx/internal/github"
	"github.com/infobloxopen/apx/internal/config"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/infobloxopen/apx/pkg/githubauth"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"    //nolint:unused // Set by build
	date    = "unknown" //nolint:unused // Set by build
)

func main() {
	ui.InitializeFromEnv()
	// Wire the browser opener into the auth package so device flow
	// can open the verification URL automatically.
	githubauth.OpenBrowserFn = gh.OpenBrowser
	root := NewApp()
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitCode(err))
	}
}

// NewApp creates a new CLI application (cobra root command).
func NewApp() *cobra.Command {
	return commands.NewRootCmd(fmt.Sprintf("apx %s (%s) %s", version, commit, date))
}

// exitCode maps errors to exit codes
func exitCode(err error) int {
	if err == nil {
		return 0
	}

	// Handle config errors
	if config.IsValidationError(err) {
		return 6
	}

	// Default error
	return 1
}

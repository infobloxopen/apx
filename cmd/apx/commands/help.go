package commands

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/spf13/cobra"
)

// setColorizedHelp installs a colorized help function on cmd.
// Cobra walks the parent chain when looking up HelpFunc, so every subcommand
// inherits this automatically without needing per-command wiring.
func setColorizedHelp(cmd *cobra.Command) {
	cmd.SetHelpFunc(colorHelpFunc)
}

func colorHelpFunc(cmd *cobra.Command, _ []string) {
	// Honor --no-color even when PersistentPreRunE hasn't run yet
	// (e.g. `apx --no-color --help` triggers help before the pre-run hook).
	if noColor, err := cmd.Root().PersistentFlags().GetBool("no-color"); err == nil && noColor {
		ui.SetColorEnabled(false)
	}

	out := cmd.OutOrStdout()
	bold := color.New(color.Bold)
	cyan := color.New(color.FgCyan)

	// Description — long preferred, fall back to short.
	desc := cmd.Long
	if desc == "" {
		desc = cmd.Short
	}
	if desc != "" {
		fmt.Fprintln(out, strings.TrimRight(desc, " \n"))
		fmt.Fprintln(out)
	}

	// Usage line
	fmt.Fprintf(out, "%s\n", bold.Sprint("Usage:"))
	if cmd.Runnable() {
		fmt.Fprintf(out, "  %s\n", cmd.UseLine())
	}
	if cmd.HasAvailableSubCommands() {
		fmt.Fprintf(out, "  %s [command]\n", cmd.CommandPath())
	}

	// Aliases
	if len(cmd.Aliases) > 0 {
		fmt.Fprintln(out)
		fmt.Fprintf(out, "%s\n", bold.Sprint("Aliases:"))
		fmt.Fprintf(out, "  %s\n", cmd.NameAndAliases())
	}

	// Examples
	if cmd.HasExample() {
		fmt.Fprintln(out)
		fmt.Fprintf(out, "%s\n", bold.Sprint("Examples:"))
		fmt.Fprintln(out, cmd.Example)
	}

	// Available Commands
	if cmd.HasAvailableSubCommands() {
		fmt.Fprintln(out)
		fmt.Fprintf(out, "%s\n", bold.Sprint("Available Commands:"))
		pad := subcommandPadding(cmd)
		for _, sub := range cmd.Commands() {
			if sub.IsAvailableCommand() || sub.Name() == "help" {
				// Compute spacing manually — cyan.Sprint embeds ANSI codes that
				// would throw off %-*s width calculation.
				spacing := strings.Repeat(" ", max(0, pad-len(sub.Name())))
				fmt.Fprintf(out, "  %s%s  %s\n", cyan.Sprint(sub.Name()), spacing, sub.Short)
			}
		}
	}

	// Local Flags
	if cmd.HasAvailableLocalFlags() {
		fmt.Fprintln(out)
		fmt.Fprintf(out, "%s\n", bold.Sprint("Flags:"))
		fmt.Fprint(out, strings.TrimRight(cmd.LocalFlags().FlagUsages(), "\n"))
		fmt.Fprintln(out)
	}

	// Global / Inherited Flags
	if cmd.HasAvailableInheritedFlags() {
		fmt.Fprintln(out)
		fmt.Fprintf(out, "%s\n", bold.Sprint("Global Flags:"))
		fmt.Fprint(out, strings.TrimRight(cmd.InheritedFlags().FlagUsages(), "\n"))
		fmt.Fprintln(out)
	}

	// Additional help topics (commands marked as help topics, not runnable)
	if cmd.HasHelpSubCommands() {
		fmt.Fprintln(out)
		fmt.Fprintf(out, "%s\n", bold.Sprint("Additional help topics:"))
		pad := subcommandPadding(cmd)
		for _, sub := range cmd.Commands() {
			if sub.IsAdditionalHelpTopicCommand() {
				fmt.Fprintf(out, "  %-*s  %s\n", pad, sub.Name(), sub.Short)
			}
		}
	}

	// Footer
	if cmd.HasAvailableSubCommands() {
		fmt.Fprintln(out)
		fmt.Fprintf(out, "Use \"%s [command] --help\" for more information about a command.\n", cmd.CommandPath())
	}
}

// subcommandPadding returns the column width needed to align descriptions
// in the Available Commands block.
func subcommandPadding(cmd *cobra.Command) int {
	maxLen := 11 // Cobra's default minimum
	for _, sub := range cmd.Commands() {
		if (sub.IsAvailableCommand() || sub.IsAdditionalHelpTopicCommand()) && len(sub.Name()) > maxLen {
			maxLen = len(sub.Name())
		}
	}
	return maxLen
}

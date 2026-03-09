package commands

import (
	"fmt"

	"github.com/infobloxopen/apx/internal/config"
	"github.com/infobloxopen/apx/internal/policy"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/spf13/cobra"
)

func newPolicyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "policy",
		Short: "Policy operations",
	}
	cmd.AddCommand(newPolicyCheckCmd())
	return cmd
}

func newPolicyCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check [path]",
		Short: "Check policy compliance",
		Args:  cobra.MaximumNArgs(1),
		RunE:  policyCheckAction,
	}
}

func policyCheckAction(cmd *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	cfg, err := loadConfig(cmd)
	if err != nil {
		ui.Error("Failed to load config: %v", err)
		return err
	}

	return checkPolicy(cfg, path)
}

func checkPolicy(cfg *config.Config, path string) error {
	ui.Info("Checking policy compliance in %s...", path)

	result, err := policy.Check(cfg.Policy, path)
	if err != nil {
		ui.Error("Policy check failed: %v", err)
		return err
	}

	if result.Checked == 0 {
		ui.Info("No policy rules configured")
		return nil
	}

	ui.Info("Evaluated %d policy rule(s)", result.Checked)

	if result.Passed() {
		ui.Success("All policies are compliant")
		return nil
	}

	for _, v := range result.Violations {
		ui.Error("[%s] %s", v.Rule, v.Message)
	}

	return fmt.Errorf("policy check failed: %d violation(s) found", len(result.Violations))
}

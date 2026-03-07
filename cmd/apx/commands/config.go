package commands

import (
	"encoding/json"
	"fmt"

	"github.com/infobloxopen/apx/internal/config"
	"github.com/infobloxopen/apx/internal/ui"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configuration operations",
	}
	cmd.AddCommand(newConfigInitCmd())
	cmd.AddCommand(newConfigValidateCmd())
	cmd.AddCommand(newConfigMigrateCmd())
	return cmd
}

func newConfigInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize configuration file",
		RunE: func(cmd *cobra.Command, args []string) error {
			return config.Init()
		},
	}
}

func newConfigValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration file against the canonical schema",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, _ := cmd.Root().PersistentFlags().GetString("config")
			if configPath == "" {
				configPath = "apx.yaml"
			}
			jsonOut, _ := cmd.Root().PersistentFlags().GetBool("json")

			result, err := config.ValidateFile(configPath)
			if err != nil {
				ui.Error("Configuration validation failed: %v", err)
				return err
			}

			if jsonOut {
				data, _ := json.MarshalIndent(result, "", "  ")
				fmt.Fprintln(cmd.OutOrStdout(), string(data))
				if !result.Valid {
					return fmt.Errorf("validation failed")
				}
				return nil
			}

			// Render warnings
			for _, w := range result.Warnings {
				ui.Warning("%s", w.Error())
			}

			if !result.Valid {
				ui.Error("Validation failed (%d error(s))", len(result.Errors))
				for _, e := range result.Errors {
					ui.Error("  %s", e.Error())
				}
				return fmt.Errorf("validation failed with %d error(s)", len(result.Errors))
			}

			if len(result.Warnings) > 0 {
				ui.Success("Configuration is valid (%d warning(s))", len(result.Warnings))
			} else {
				ui.Success("Configuration is valid")
			}
			return nil
		},
	}
}

func newConfigMigrateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "Migrate configuration file to the current schema version",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, _ := cmd.Root().PersistentFlags().GetString("config")
			if configPath == "" {
				configPath = "apx.yaml"
			}
			jsonOut, _ := cmd.Root().PersistentFlags().GetBool("json")

			migrateResult, err := config.MigrateFile(configPath)
			if err != nil {
				ui.Error("Migration failed: %v", err)
				return err
			}

			if jsonOut {
				data, _ := json.MarshalIndent(migrateResult, "", "  ")
				fmt.Fprintln(cmd.OutOrStdout(), string(data))
				return nil
			}

			if !migrateResult.Migrated {
				ui.Success("apx.yaml is already at version %d (current). No migration needed.", migrateResult.ToVersion)
				return nil
			}

			ui.Info("Migrating apx.yaml from version %d to version %d...", migrateResult.FromVersion, migrateResult.ToVersion)
			if migrateResult.Backup != "" {
				ui.Info("  Backed up original to %s", migrateResult.Backup)
			}
			for _, c := range migrateResult.Changes {
				ui.Info("  %s: %s (%s)", c.Action, c.Field, c.Detail)
			}
			ui.Success("Migration complete. apx.yaml is now version %d.", migrateResult.ToVersion)
			return nil
		},
	}
}

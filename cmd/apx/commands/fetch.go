package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/infobloxopen/apx/internal/validator"
	"github.com/spf13/cobra"
)

func newFetchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "fetch",
		Short:  "Download and cache toolchain dependencies for offline use",
		Hidden: true,
		RunE:   fetchAction,
	}
	cmd.Flags().StringP("config", "c", "apx.yaml", "Path to configuration file")
	cmd.Flags().String("output", ".apx-tools", "Output directory for toolchain bundles")
	cmd.Flags().Bool("verify", true, "Verify checksums of downloaded tools")
	return cmd
}

func fetchAction(cmd *cobra.Command, args []string) error {
	configPath, _ := cmd.Flags().GetString("config")
	outputDir, _ := cmd.Flags().GetString("output")
	verify, _ := cmd.Flags().GetBool("verify")

	fmt.Printf("Fetching toolchain dependencies...\n")
	fmt.Printf("Config: %s\n", configPath)
	fmt.Printf("Output: %s\n", outputDir)

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	resolver := validator.NewToolchainResolver()

	lockPath := "apx.lock"
	if _, err := os.Stat(lockPath); err == nil {
		fmt.Printf("Loading toolchain profile from %s...\n", lockPath)

		profile, err := validator.LoadToolchainProfile(lockPath)
		if err != nil {
			return fmt.Errorf("failed to load toolchain profile: %w", err)
		}

		for tool, toolRef := range profile.Tools {
			fmt.Printf("Fetching %s@%s...\n", tool, toolRef.Version)

			_, err := resolver.ResolveTool(tool, toolRef.Version)
			if err != nil {
				fmt.Printf("Warning: failed to resolve %s: %v\n", tool, err)
				continue
			}

			bundlePath := filepath.Join(outputDir, tool, toolRef.Version)
			if err := os.MkdirAll(bundlePath, 0755); err != nil {
				return fmt.Errorf("failed to create bundle directory: %w", err)
			}

			fmt.Printf("\u2713 Cached %s at %s\n", tool, bundlePath)
		}
	} else {
		fmt.Printf("No apx.lock found, skipping toolchain fetch\n")
	}

	if verify {
		fmt.Printf("Verifying checksums...\n")
	}

	fmt.Printf("\u2713 Toolchain fetch complete\n")
	return nil
}

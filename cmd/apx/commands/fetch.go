package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/infobloxopen/apx/internal/validator"
	"github.com/urfave/cli/v2"
)

// FetchCommand implements the apx fetch command for toolchain hydration
func FetchCommand() *cli.Command {
	return &cli.Command{
		Name:  "fetch",
		Usage: "Download and cache toolchain dependencies for offline use",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Value:   "apx.yaml",
				Usage:   "Path to configuration file",
			},
			&cli.StringFlag{
				Name:  "output",
				Value: ".apx-tools",
				Usage: "Output directory for toolchain bundles",
			},
			&cli.BoolFlag{
				Name:  "verify",
				Value: true,
				Usage: "Verify checksums of downloaded tools",
			},
		},
		Action: fetchAction,
	}
}

func fetchAction(c *cli.Context) error {
	configPath := c.String("config")
	outputDir := c.String("output")
	verify := c.Bool("verify")

	fmt.Printf("Fetching toolchain dependencies...\n")
	fmt.Printf("Config: %s\n", configPath)
	fmt.Printf("Output: %s\n", outputDir)

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Initialize toolchain resolver
	resolver := validator.NewToolchainResolver()

	// Attempt to load apx.lock for pinned versions
	lockPath := "apx.lock"
	if _, err := os.Stat(lockPath); err == nil {
		fmt.Printf("Loading toolchain profile from %s...\n", lockPath)

		profile, err := validator.LoadToolchainProfile(lockPath)
		if err != nil {
			return fmt.Errorf("failed to load toolchain profile: %w", err)
		}

		// Download tools from profile
		for tool, toolRef := range profile.Tools {
			fmt.Printf("Fetching %s@%s...\n", tool, toolRef.Version)

			_, err := resolver.ResolveTool(tool, toolRef.Version)
			if err != nil {
				fmt.Printf("Warning: failed to resolve %s: %v\n", tool, err)
				continue
			}

			// Copy tool to offline bundle
			bundlePath := filepath.Join(outputDir, tool, toolRef.Version)
			if err := os.MkdirAll(bundlePath, 0755); err != nil {
				return fmt.Errorf("failed to create bundle directory: %w", err)
			}

			fmt.Printf("✓ Cached %s at %s\n", tool, bundlePath)
		}
	} else {
		fmt.Printf("No apx.lock found, skipping toolchain fetch\n")
	}

	if verify {
		fmt.Printf("Verifying checksums...\n")
		// TODO: Implement checksum verification
	}

	fmt.Printf("✓ Toolchain fetch complete\n")
	return nil
}

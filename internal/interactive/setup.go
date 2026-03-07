package interactive

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/infobloxopen/apx/internal/detector"
	"github.com/infobloxopen/apx/internal/ui"
)

// RunSetup guides the user through interactive configuration
func RunSetup(defaults *detector.ProjectDefaults, kind, modulePath string) (string, string, error) {
	ui.Info("\U0001f680 Welcome to APX initialization!")
	ui.Info("Let's set up your configuration with some questions...")
	ui.Info("")

	// Schema type selection (if not provided)
	if kind == "" {
		err := huh.NewSelect[string]().
			Title("What type of schema do you want to create?").
			Description("Choose the schema format that best fits your needs").
			Options(
				huh.NewOption("proto", "proto"),
				huh.NewOption("openapi", "openapi"),
				huh.NewOption("avro", "avro"),
				huh.NewOption("jsonschema", "jsonschema"),
				huh.NewOption("parquet", "parquet"),
			).
			Value(&kind).
			Run()
		if err != nil {
			return "", "", fmt.Errorf("failed to get schema type: %w", err)
		}
	}

	// Module path (if not provided)
	if modulePath == "" {
		var defaultModulePath string
		switch kind {
		case "proto":
			defaultModulePath = "com.example.service.v1"
		case "openapi":
			defaultModulePath = "my-api"
		case "avro":
			defaultModulePath = "com.example.events"
		case "jsonschema":
			defaultModulePath = "com.example.schema"
		case "parquet":
			defaultModulePath = "com.example.data"
		}

		err := huh.NewInput().
			Title("Module path/name:").
			Description("This will be used as the namespace/package for your schema").
			Value(&modulePath).
			Placeholder(defaultModulePath).
			Run()
		if err != nil {
			return "", "", fmt.Errorf("failed to get module path: %w", err)
		}
		if modulePath == "" {
			modulePath = defaultModulePath
		}
	}

	ui.Info("")
	ui.Info("\U0001f4cb Schema Configuration:")
	ui.Info("   Type: %s", kind)
	ui.Info("   Module: %s", modulePath)
	ui.Info("")

	// Organization name
	err := huh.NewInput().
		Title("Organization name:").
		Description("This will be used in generated configurations and tooling").
		Value(&defaults.Org).
		Placeholder(defaults.Org).
		Run()
	if err != nil {
		return "", "", fmt.Errorf("failed to get organization name: %w", err)
	}

	// Repository name
	err = huh.NewInput().
		Title("Repository name:").
		Description("The name of your API repository").
		Value(&defaults.Repo).
		Placeholder(defaults.Repo).
		Run()
	if err != nil {
		return "", "", fmt.Errorf("failed to get repository name: %w", err)
	}

	// Target languages
	err = huh.NewMultiSelect[string]().
		Title("Target languages (select all that apply):").
		Description("APX will generate code for these languages").
		Options(
			huh.NewOption("go", "go"),
			huh.NewOption("python", "python"),
			huh.NewOption("java", "java"),
		).
		Value(&defaults.Languages).
		Run()
	if err != nil {
		return "", "", fmt.Errorf("failed to get target languages: %w", err)
	}

	ui.Info("")
	ui.Success("Configuration complete! \U0001f389")
	return kind, modulePath, nil
}

// PromptForString prompts the user for a string value with a default
func PromptForString(message, defaultValue string, result *string) error {
	*result = defaultValue
	return huh.NewInput().
		Title(message).
		Value(result).
		Placeholder(defaultValue).
		Run()
}

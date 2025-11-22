package interactive

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/infobloxopen/apx/internal/detector"
	"github.com/infobloxopen/apx/internal/ui"
)

// RunSetup guides the user through interactive configuration
func RunSetup(defaults *detector.ProjectDefaults, kind, modulePath string) (string, string, error) {
	ui.Info("🚀 Welcome to APX initialization!")
	ui.Info("Let's set up your configuration with some questions...")
	ui.Info("")

	// Schema type selection (if not provided)
	if kind == "" {
		schemaOptions := []string{"proto", "openapi", "avro", "jsonschema", "parquet"}
		schemaPrompt := &survey.Select{
			Message: "What type of schema do you want to create?",
			Options: schemaOptions,
			Help:    "Choose the schema format that best fits your needs",
		}
		if err := survey.AskOne(schemaPrompt, &kind); err != nil {
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

		modulePrompt := &survey.Input{
			Message: "Module path/name:",
			Default: defaultModulePath,
			Help:    "This will be used as the namespace/package for your schema",
		}
		if err := survey.AskOne(modulePrompt, &modulePath); err != nil {
			return "", "", fmt.Errorf("failed to get module path: %w", err)
		}
	}

	ui.Info("")
	ui.Info("📋 Schema Configuration:")
	ui.Info("   Type: %s", kind)
	ui.Info("   Module: %s", modulePath)
	ui.Info("")

	// Organization name
	orgPrompt := &survey.Input{
		Message: "Organization name:",
		Default: defaults.Org,
		Help:    "This will be used in generated configurations and tooling",
	}
	if err := survey.AskOne(orgPrompt, &defaults.Org); err != nil {
		return "", "", fmt.Errorf("failed to get organization name: %w", err)
	}

	// Repository name
	repoPrompt := &survey.Input{
		Message: "Repository name:",
		Default: defaults.Repo,
		Help:    "The name of your API repository",
	}
	if err := survey.AskOne(repoPrompt, &defaults.Repo); err != nil {
		return "", "", fmt.Errorf("failed to get repository name: %w", err)
	}

	// Target languages
	languageOptions := []string{"go", "python", "java"}
	languagePrompt := &survey.MultiSelect{
		Message: "Target languages (select all that apply):",
		Options: languageOptions,
		Default: defaults.Languages,
		Help:    "APX will generate code for these languages",
	}
	if err := survey.AskOne(languagePrompt, &defaults.Languages); err != nil {
		return "", "", fmt.Errorf("failed to get target languages: %w", err)
	}

	ui.Info("")
	ui.Success("Configuration complete! 🎉")
	return kind, modulePath, nil
}

// PromptForString prompts the user for a string value with a default
func PromptForString(message, defaultValue string, result *string) error {
	prompt := &survey.Input{
		Message: message,
		Default: defaultValue,
	}
	return survey.AskOne(prompt, result)
}

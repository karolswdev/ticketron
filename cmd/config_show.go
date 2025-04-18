package cmd

import (
	"fmt"
	"io" // Added for output writer

	"github.com/spf13/cobra"

	"github.com/karolswdev/ticketron/internal/config"
)

// Constants for keyring interaction (assuming they exist in config package or are defined here)
// If these are defined in config package, use config.KeyringService, config.KeyringUser
const keyringService = "ticketron"
const keyringUser = "llm_api_key"

// configShowCmd represents the show command
var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current Ticketron configuration",
	Long: `Displays the currently loaded configuration values
from config files and environment variables.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get the provider instance
		provider, err := GetProvider()
		if err != nil {
			// Log is not available here, return error for main to handle
			return fmt.Errorf("failed to get service provider: %w", err)
		}

		// Get the output writer from the command
		writer := cmd.OutOrStdout()

		// Execute the core logic
		return configShowRunE(provider.Config, provider.Keyring, writer)
	},
}

// configShowRunE contains the core logic for the 'config show' command.
func configShowRunE(cfgProvider ConfigProvider, keyringClient KeyringClient, writer io.Writer) error {
	cfg, err := cfgProvider.LoadConfig()
	if err != nil {
		// Return error instead of logging fatal
		return fmt.Errorf("error loading configuration: %w", err)
	}

	fmt.Fprintln(writer, "Current Ticketron Configuration:")
	// Removed the line attempting to print config file path as it wasn't part of original output and function doesn't exist
	fmt.Fprintf(writer, "  MCP Server URL: %s\n", cfg.MCPServerURL)
	fmt.Fprintf(writer, "  LLM Provider:   %s\n", cfg.LLM.Provider)
	// Display provider-specific settings
	switch cfg.LLM.Provider {
	case "openai":
		fmt.Fprintf(writer, "    OpenAI Model: %s\n", cfg.LLM.OpenAI.ModelName)
		if cfg.LLM.OpenAI.BaseURL != "" {
			fmt.Fprintf(writer, "    OpenAI BaseURL: %s\n", cfg.LLM.OpenAI.BaseURL)
		}
	// Add cases for other providers like "anthropic", "ollama" here when implemented
	default:
		fmt.Fprintf(writer, "    (No specific settings shown for provider '%s')\n", cfg.LLM.Provider)
	}

	// Check if API key exists using the injected KeyringClient
	_, err = keyringClient.GetAPIKey(keyringService, keyringUser) // Use constants
	apiKeyStatus := "Set (use 'tix config set-key' to change)"
	if err != nil {
		if err == config.ErrAPIKeyNotFound {
			apiKeyStatus = "Not Set (use 'tix config set-key' to set)"
		} else {
			// Return error instead of logging warning
			// We still want to show the rest of the config, so maybe just update status
			apiKeyStatus = fmt.Sprintf("Status Unknown (error checking keychain/env: %v)", err)
			// Optionally return the error if checking the key is critical:
			// return fmt.Errorf("could not check API key status: %w", err)
		}
	}
	fmt.Fprintf(writer, "  LLM API Key:    %s\n", apiKeyStatus) // Display status, not the key itself

	return nil // Indicate success
}

func init() {
	configCmd.AddCommand(configShowCmd) // Use the renamed variable
}

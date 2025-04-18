package cmd

import (
	"errors"
	"fmt"
	"io" // Added for io.Writer

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	// Keyring interaction now happens via the KeyringClient interface
)

// Constants for keyring service and user name (should match config/config.go)
const (
	keyringServiceName = "ticketron"
	keyringUserName    = "openai_api_key"
)

// setKeyCmd represents the set-key command
var setKeyCmd = &cobra.Command{
	Use:   "set-key [api-key]",
	Short: "Stores the OpenAI API key securely in the OS keychain",
	Long: `Stores the OpenAI API key securely in the operating system's keychain or keyring.
This is the recommended way to configure the API key for Ticketron.
The key will be associated with the service 'ticketron' and user 'openai_api_key'.`,
	Args: cobra.ExactArgs(1), // Requires exactly one argument: the API key
	// RunE will be set in init() after getting the provider
}

// configSetKeyRun contains the core logic for the set-key command.
// It accepts dependencies (keyring client, writer) for testability.
func configSetKeyRun(kc KeyringClient, writer io.Writer, apiKey string) error {
	if apiKey == "" {
		return errors.New("API key cannot be empty")
	}

	log.Info().Msgf("Attempting to store API key in keychain for service '%s'...", keyringServiceName)

	err := kc.Set(keyringServiceName, keyringUserName, apiKey)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to store API key in keychain")
		// Don't write to writer on error, return error for cobra to handle
		return fmt.Errorf("failed to store API key in keychain: %w", err)
	}

	log.Info().Msg("API key stored successfully in keychain.")
	fmt.Fprintln(writer, "API key stored successfully.") // Use injected writer
	return nil
}

func init() {
	// Get the provider
	provider, err := GetProvider()
	if err != nil {
		// Handle provider initialization error - log and make command return error
		log.Error().Err(err).Msg("Failed to initialize provider for set-key command; command will be disabled.")
		setKeyCmd.RunE = func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("command disabled due to provider initialization error: %w", err)
		}
	} else {
		// Set RunE using the provider
		setKeyCmd.RunE = func(cmd *cobra.Command, args []string) error {
			// Get the writer from the command
			writer := cmd.OutOrStdout()
			// Get the API key from args
			apiKey := args[0]
			// Call the actual logic function with injected dependencies
			return configSetKeyRun(provider.Keyring, writer, apiKey)
		}
	}

	configCmd.AddCommand(setKeyCmd)
}

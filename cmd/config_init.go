package cmd

import (
	"fmt"
	"io" // Added for io.Writer

	"github.com/rs/zerolog/log" // Import log package
	"github.com/spf13/cobra"
	// Removed direct import of "github.com/karolswdev/ticketron/internal/config"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Ticketron configuration",
	Long: `Creates the default configuration directory and files if they don't exist.
This command ensures that the necessary configuration structure is in place
for Ticketron to function correctly.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get the provider
		provider, err := GetProvider()
		if err != nil {
			log.Error().Err(err).Msg("Failed to get service provider")
			return fmt.Errorf("failed to get service provider: %w", err)
		}
		// Execute the actual command logic
		return configInitRunE(provider.Config, cmd.OutOrStdout(), cmd, args)
	},
}

func init() {
	configCmd.AddCommand(initCmd)
}

// configInitRunE contains the core logic for the config init command.
// It accepts dependencies for testability.
func configInitRunE(configProvider ConfigProvider, writer io.Writer, cmd *cobra.Command, args []string) error {
	log.Info().Msg("Initializing configuration...")
	// Use the injected provider to create files
	// The configDir parameter is currently ignored by the provider implementation,
	// but included here for potential future use or interface consistency.
	// We can pass an empty string or determine the actual dir if needed later.
	err := configProvider.CreateDefaultConfigFiles("")
	if err != nil {
		log.Error().Err(err).Msg("Failed to initialize configuration files")
		return fmt.Errorf("failed to initialize configuration: %w", err)
	}
	log.Info().Msg("Configuration initialization complete.")
	// Use the injected writer for output
	fmt.Fprintln(writer, "Configuration directory and default files ensured.")
	return nil
}

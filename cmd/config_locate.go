package cmd

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/spf13/cobra"
)

// configLocateRunE contains the core logic for the config locate command.
// It uses dependency injection for testability.
func configLocateRunE(cfgProvider ConfigProvider, out io.Writer) error {
	configDir, err := cfgProvider.EnsureConfigDir()
	if err != nil {
		// Return the error instead of logging fatally
		return fmt.Errorf("error ensuring config directory: %w", err)
	}

	// Use the injected writer for output
	fmt.Fprintf(out, "Configuration directory: %s\n", configDir)
	fmt.Fprintln(out, "Expected configuration files:")
	fmt.Fprintf(out, "- %s\n", filepath.Join(configDir, "config.yaml"))
	fmt.Fprintf(out, "- %s\n", filepath.Join(configDir, "links.yaml"))
	fmt.Fprintf(out, "- %s\n", filepath.Join(configDir, "system_prompt.txt"))
	fmt.Fprintf(out, "- %s\n", filepath.Join(configDir, "context.md"))

	return nil // Indicate success
}

// locateCmd represents the locate command
var locateCmd = &cobra.Command{
	Use:   "locate",
	Short: "Locate Ticketron configuration files",
	Long: `Displays the paths to the configuration files being used by Ticketron.
This command helps you find where Ticketron is looking for its settings.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get the provider which includes the ConfigProvider
		provider, err := GetProvider()
		if err != nil {
			// Return error to Cobra for handling
			return fmt.Errorf("failed to initialize provider: %w", err)
		}

		// Execute the core logic, passing dependencies
		// Use cmd.OutOrStdout() to get the correct output stream (stdout or test buffer)
		return configLocateRunE(provider.Config, cmd.OutOrStdout())
	},
}

func init() {
	configCmd.AddCommand(locateCmd)
}

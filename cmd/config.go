package cmd

import (
	"github.com/spf13/cobra"
)

// configCmd represents the config command group
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage Ticketron configuration",
	Long: `Provides commands to show, locate, and manage Ticketron configuration files.
This command itself does not perform any action but serves as a parent for subcommands.`,
	// No Run function needed for a parent command
}

func init() {
	rootCmd.AddCommand(configCmd)
}

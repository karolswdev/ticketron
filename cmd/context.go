package cmd

import (
	"fmt"
	"os" // Needed for os.IsNotExist, os.Getenv, os.Stdin, os.Stdout, os.Stderr, os.OpenFile, os.O_APPEND, os.O_CREATE, os.O_WRONLY
	"os/exec"
	"path/filepath"
	"runtime" // Needed for runtime.GOOS

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Manage the persistent LLM context file (~/.ticketron/context.md)",
	Long: `Provides subcommands to show, edit, or add entries to the
context.md file used to provide persistent context to the LLM.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Ensure config directory exists before any context command runs
		provider, err := GetProvider() // Use the main provider factory
		if err != nil {
			// Log the error here as GetProvider might fail before logger is fully set up elsewhere
			log.Error().Err(err).Msg("Failed to get service provider in context command")
			return fmt.Errorf("failed to initialize services: %w", err)
		}
		_, err = provider.Config.EnsureConfigDir() // Access Config field
		return err
	},
}

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Show the content of the context file",
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Debug().Msg("Executing context show command")

		provider, err := GetProvider()
		if err != nil {
			log.Error().Err(err).Msg("Failed to get service provider")
			return fmt.Errorf("failed to initialize services: %w", err)
		}

		contextContent, err := provider.Config.LoadContext()
		if err != nil {
			// Handle file not found specifically
			if os.IsNotExist(err) {
				log.Warn().Msg("Context file (~/.ticketron/context.md) not found.")
				fmt.Println("Context file does not exist yet.")
				return nil // Not a fatal error for 'show'
			}
			// Handle other potential errors
			log.Error().Err(err).Msg("Failed to load context file")
			return fmt.Errorf("failed to read context file: %w", err)
		}

		fmt.Println(contextContent) // Print the content to stdout
		return nil
	},
}

var editCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit the context file using $EDITOR",
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Debug().Msg("Executing context edit command")

		provider, err := GetProvider()
		if err != nil {
			log.Error().Err(err).Msg("Failed to get service provider")
			return fmt.Errorf("failed to initialize services: %w", err)
		}

		// EnsureConfigDir also returns the path, which we need here
		configDir, err := provider.Config.EnsureConfigDir()
		if err != nil {
			log.Error().Err(err).Msg("Failed to ensure config directory exists")
			return fmt.Errorf("failed to ensure config directory: %w", err)
		}
		contextFilePath := filepath.Join(configDir, "context.md")
		log.Debug().Str("path", contextFilePath).Msg("Context file path determined")

		// Determine the editor
		editor := os.Getenv("EDITOR")
		if editor == "" {
			log.Debug().Msg("$EDITOR not set, using default editor for OS")
			if runtime.GOOS == "windows" {
				editor = "notepad"
			} else {
				editor = "vim" // Sensible default for Linux/macOS
			}
		}
		log.Debug().Str("editor", editor).Msg("Using editor")

		// Prepare the command
		editorCmd := exec.Command(editor, contextFilePath)
		editorCmd.Stdin = os.Stdin
		editorCmd.Stdout = os.Stdout
		editorCmd.Stderr = os.Stderr

		// Run the editor
		log.Debug().Msg("Launching editor...")
		err = editorCmd.Run()
		if err != nil {
			log.Error().Err(err).Str("editor", editor).Msg("Editor command failed")
			return fmt.Errorf("failed to run editor '%s': %w", editor, err)
		}

		log.Info().Msg("Editor finished.")
		return nil
	},
}

var addCmd = &cobra.Command{
	Use:   "add [entry]",
	Short: "Add a new entry (line) to the context file",
	Args:  cobra.ExactArgs(1), // Requires exactly one argument
	RunE: func(cmd *cobra.Command, args []string) error {
		entry := args[0]
		log.Debug().Str("entry", entry).Msg("Executing context add command")

		provider, err := GetProvider()
		if err != nil {
			log.Error().Err(err).Msg("Failed to get service provider")
			return fmt.Errorf("failed to initialize services: %w", err)
		}

		configDir, err := provider.Config.EnsureConfigDir()
		if err != nil {
			log.Error().Err(err).Msg("Failed to ensure config directory exists")
			return fmt.Errorf("failed to ensure config directory: %w", err)
		}
		contextFilePath := filepath.Join(configDir, "context.md")
		log.Debug().Str("path", contextFilePath).Msg("Context file path determined")

		// Open file in append mode (create if doesn't exist)
		f, err := os.OpenFile(contextFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Error().Err(err).Str("path", contextFilePath).Msg("Failed to open context file for appending")
			return fmt.Errorf("failed to open context file: %w", err)
		}
		defer f.Close()

		// Add entry as a new line
		formattedEntry := fmt.Sprintf("%s\n", entry)
		if _, err := f.WriteString(formattedEntry); err != nil {
			log.Error().Err(err).Str("path", contextFilePath).Msg("Failed to write entry to context file")
			return fmt.Errorf("failed to write to context file: %w", err)
		}

		log.Info().Str("path", contextFilePath).Msg("Entry successfully added to context file")
		fmt.Println("Entry added to context file.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(contextCmd)
	contextCmd.AddCommand(showCmd)
	contextCmd.AddCommand(editCmd)
	contextCmd.AddCommand(addCmd)
}

// Note: The GetConfigProvider helper is removed as we now use GetProvider directly.

package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// version is set during build time (e.g., via ldflags)
// Default is "dev" for local development.
var version = "dev"

var (
	logLevel string
	// Log is the globally configured zerolog logger instance used throughout the cmd package.
	// It's initialized in rootCmd's PersistentPreRunE based on the --log-level flag.
	Log zerolog.Logger
)

// configureLogger sets up the global zerolog logger based on the logLevel flag.
// This is extracted to be reusable by both the package-level rootCmd and NewRootCmd.
func configureLogger(levelStr string) error {
	// Configure logger
	output := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}
	log.Logger = log.Output(output) // Use zerolog's global logger temporarily for setup

	// Set global log level
	level, err := zerolog.ParseLevel(strings.ToLower(levelStr))
	if err != nil {
		log.Warn().Msgf("Invalid log level '%s', defaulting to 'info'", levelStr)
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	// Assign the configured logger to the package variable
	Log = log.Logger.With().Timestamp().Logger() // Add timestamp to all logs

	Log.Debug().Msgf("Log level set to '%s'", level.String())
	return nil
}

// persistentPreRunLogic contains the logic for PersistentPreRunE, reusable by NewRootCmd.
func persistentPreRunLogic(cmd *cobra.Command, args []string) error {
	// Handle --version flag
	showVersion, _ := cmd.Flags().GetBool("version") // Error ignored, default is false
	if showVersion {
		fmt.Println(version)
		os.Exit(0) // Exit after showing version, as Cobra does not handle this automatically in PersistentPreRunE
	}
	// Configure logger using the bound logLevel variable
	return configureLogger(logLevel)
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "tix",
	Short: "Ticketron CLI - Fire-and-forget JIRA issue creation",
	Long: `Ticketron (tix) is a CLI tool designed to quickly create JIRA issues
based on brief user input, leveraging LLMs for detail generation and
an MCP server for JIRA interaction.`,
	PersistentPreRunE: persistentPreRunLogic, // Use the extracted logic
}

// Execute is the main entry point for the Cobra CLI application.
// It parses command-line arguments, executes the appropriate command (rootCmd or one of its subcommands),
// handles flag parsing, and manages error reporting. This function is typically called directly from main.main().
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		// Ensure logger is initialized even if PersistentPreRunE failed early
		if Log.GetLevel() == zerolog.Disabled {
			_ = configureLogger("info") // Use default level if logger wasn't set up
		}
		Log.Error().Err(err).Msg("Command execution failed") // Use logger for errors
		os.Exit(1)
	}
}

// NewRootCmd creates a new instance of the root command, configured for testing or embedding.
// It mirrors the setup of the package-level rootCmd.
func NewRootCmd() *cobra.Command {
	// Create a new command instance mirroring rootCmd
	newCmd := &cobra.Command{
		Use:   "tix",
		Short: "Ticketron CLI - Fire-and-forget JIRA issue creation",
		Long: `Ticketron (tix) is a CLI tool designed to quickly create JIRA issues
based on brief user input, leveraging LLMs for detail generation and
an MCP server for JIRA interaction.`,
		// Note: PersistentPreRunE needs careful handling if it relies on package vars
		// directly modified by flags bound to the *other* rootCmd instance.
		// Re-binding flags to local vars for this instance is safer.
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Get flags directly from this command instance
			lvl, _ := cmd.Flags().GetString("log-level")
			showVersion, _ := cmd.Flags().GetBool("version")

			if showVersion {
				fmt.Println(version)
				os.Exit(0)
			}
			// Configure logger using the flag value from *this* command
			return configureLogger(lvl)
		},
	}

	// Define flags specifically for this new command instance
	// Use local variables to avoid conflicts with package-level flag bindings
	var instanceLogLevel string
	newCmd.PersistentFlags().StringVar(&instanceLogLevel, "log-level", "info", "Set log level (debug, info, warn, error, fatal, panic)")
	newCmd.PersistentFlags().Bool("version", false, "Show application version")
	newCmd.PersistentFlags().StringP("output", "o", "text", "Output format (text|json)")

	// Add subcommands (ensure subcommands are also initialized correctly if needed)
	// We need to add the *initialized* subcommand variables from their respective files.
	newCmd.AddCommand(configCmd) // Assuming configCmd is initialized in config.go's init()
	newCmd.AddCommand(createCmd) // Assuming createCmd is initialized in create.go's init()
	newCmd.AddCommand(searchCmd) // Assuming searchCmd is initialized in search.go's init()
	newCmd.AddCommand(completionCmd)

	return newCmd
}

// completionCmd represents the completion command
var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion script",
	Long: `To load completions:

Bash:
  $ source <(tix completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ tix completion bash > /etc/bash_completion.d/tix
  # macOS:
  $ tix completion bash > /usr/local/etc/bash_completion.d/tix

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it. You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ tix completion zsh > "${fpath[1]}/_tix"

  # You will need to start a new shell for this setup to take effect.

Fish:
  $ tix completion fish | source

  # To load completions for each session, execute once:
  $ tix completion fish > ~/.config/fish/completions/tix.fish

PowerShell:
  PS> tix completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> tix completion powershell > tix.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs), // Updated: Use MatchAll(ExactArgs(1), OnlyValidArgs)
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletion(os.Stdout) // Use the main rootCmd for generation
		case "zsh":
			return rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			return rootCmd.GenFishCompletion(os.Stdout, true) // includeDesc true
		case "powershell":
			return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
		default:
			// This case should theoretically not be reached due to Args validation,
			// but included for robustness.
			return fmt.Errorf("unsupported shell type %q", args[0])
		}
	},
}

func init() {
	// Define flags for the package-level rootCmd
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "Set log level (debug, info, warn, error, fatal, panic)")
	rootCmd.PersistentFlags().Bool("version", false, "Show application version")
	rootCmd.PersistentFlags().StringP("output", "o", "text", "Output format (text|json)")

	// Add child commands to the package-level rootCmd
	// Subcommands like createCmd, searchCmd, configCmd are added via their own init() functions.
	rootCmd.AddCommand(completionCmd)
}

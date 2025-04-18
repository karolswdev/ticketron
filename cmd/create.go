package cmd

import (
	"bufio" // Added for interactive confirmation
	"context"
	"encoding/json" // Added for JSON output
	"errors"        // Added for errors.Is
	"fmt"
	"io" // Added for io.Writer
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/karolswdev/ticketron/internal/config"
	"github.com/karolswdev/ticketron/internal/llm"
	"github.com/karolswdev/ticketron/internal/mcpclient"
)

// --- Concrete Implementations of Interfaces ---

// NOTE: defaultConfigProvider moved to providers.go

// defaultLLMProcessor removed - using llm.Client via Provider now

// NOTE: defaultMCPClient and newDefaultMCPClient moved to providers.go

// DefaultProjectMapper implements the ProjectMapper interface. Exported for tests.
type DefaultProjectMapper struct{}

func (m *DefaultProjectMapper) MapSuggestionToKey(suggestion string, linksCfg *config.LinksConfig) (string, *config.ProjectLink, error) {
	if linksCfg == nil || linksCfg.Projects == nil {
		return "", nil, fmt.Errorf("links configuration is nil or empty")
	}
	for i := range linksCfg.Projects {
		link := &linksCfg.Projects[i] // Get pointer
		if strings.EqualFold(suggestion, link.Name) {
			Log.Debug().Str("suggestion", suggestion).Str("key", link.Key).Msg("Mapped project name to key")
			return link.Key, link, nil
		}
	}

	homeDir, _ := os.UserHomeDir()                                     // Best effort
	expectedPath := filepath.Join(homeDir, ".ticketron", "links.yaml") // Best effort path for logging
	Log.Error().Str("suggestion", suggestion).Str("expected_links_path", expectedPath).Msg("Mapping failed")
	// Return the specific sentinel error, wrapping the suggestion for context in the calling function if needed.
	return "", nil, config.ErrProjectMappingFailed
}

// DefaultIssueTypeResolver implements the IssueTypeResolver interface. Exported for tests.
type DefaultIssueTypeResolver struct{}

const defaultIssueType = "Task" // Hardcoded default

func (r *DefaultIssueTypeResolver) Resolve(flagType string, matchedProjectLink *config.ProjectLink, projectKey string) string {
	if flagType != "" {
		Log.Debug().Str("issue_type", flagType).Msg("Using issue type from --type flag")
		return flagType
	}

	if matchedProjectLink != nil && matchedProjectLink.DefaultIssueType != "" {
		Log.Debug().Str("project_key", projectKey).Str("issue_type", matchedProjectLink.DefaultIssueType).Msg("Using default issue type from links.yaml")
		return matchedProjectLink.DefaultIssueType
	}

	Log.Debug().Str("project_key", projectKey).Str("issue_type", defaultIssueType).Msg("Using hardcoded default issue type 'Task'")
	return defaultIssueType
}

// --- Helper Functions for createCmdRunner.Run ---

// loadedConfigs holds all necessary configuration data.
type loadedConfigs struct {
	appConfig    *config.AppConfig
	linksConfig  *config.LinksConfig
	systemPrompt string
	contextData  string
}

// loadAllConfigs loads all required configuration files.
func loadAllConfigs(cp ConfigProvider) (*loadedConfigs, error) {
	Log.Debug().Msg("Loading all configurations...")
	cfg, err := cp.LoadConfig()
	if err != nil {
		Log.Error().Err(err).Msg("Failed to load main configuration file (config.yaml)")
		// Add user message based on error type using switch
		switch {
		case errors.Is(err, config.ErrConfigRead), errors.Is(err, config.ErrConfigParse):
			fmt.Fprintln(os.Stderr, "Error reading or parsing config.yaml. Please check its format and permissions.")
		case errors.Is(err, config.ErrConfigDirCreate), errors.Is(err, config.ErrConfigDirStat), errors.Is(err, config.ErrConfigDirNotDir):
			fmt.Fprintln(os.Stderr, "Error accessing configuration directory. Please check permissions.")
		default:
			fmt.Fprintln(os.Stderr, "An unexpected error occurred loading config.yaml.")
		}
		fmt.Fprintln(os.Stderr, "You might need to run 'tix config init'.")
		return nil, err // Return original error
	}

	linksCfg, err := cp.LoadLinks()
	if err != nil {
		Log.Error().Err(err).Msg("Failed to load links configuration file (links.yaml)")
		switch {
		case errors.Is(err, config.ErrLinksRead), errors.Is(err, config.ErrLinksParse):
			fmt.Fprintln(os.Stderr, "Error reading or parsing links.yaml. Please check its format and permissions.")
			fmt.Fprintln(os.Stderr, "You might need to run 'tix config init' to create a default.")
		default:
			fmt.Fprintln(os.Stderr, "An unexpected error occurred loading links.yaml.")
		}
		return nil, err // Return original error
	}

	systemPrompt, err := cp.LoadSystemPrompt()
	if err != nil {
		Log.Error().Err(err).Msg("Failed to load system prompt file (system_prompt.txt)")
		switch {
		case errors.Is(err, config.ErrSystemPromptRead):
			fmt.Fprintln(os.Stderr, "Error reading system_prompt.txt. Please check its permissions.")
			fmt.Fprintln(os.Stderr, "You might need to run 'tix config init' to create a default.")
		default:
			fmt.Fprintln(os.Stderr, "An unexpected error occurred loading system_prompt.txt.")
		}
		return nil, err // Return original error
	}

	contextData, err := cp.LoadContext()
	if err != nil {
		Log.Error().Err(err).Msg("Failed to load context data file (context.md)")
		switch {
		case errors.Is(err, config.ErrContextRead):
			fmt.Fprintln(os.Stderr, "Error reading context.md. Please check its permissions.")
			fmt.Fprintln(os.Stderr, "You might need to run 'tix config init' to create a default.")
		default:
			fmt.Fprintln(os.Stderr, "An unexpected error occurred loading context.md.")
		}
		return nil, err // Return original error
	}

	Log.Debug().Msg("All configurations loaded successfully.")
	return &loadedConfigs{
		appConfig:    cfg,
		linksConfig:  linksCfg,
		systemPrompt: systemPrompt,
		contextData:  contextData,
	}, nil
}

// confirmInteractively prompts the user for confirmation if interactive mode is enabled.
// Returns true if the user confirms or if interactive mode is off, false if the user aborts.
// Returns an error only if reading user input fails.
func confirmInteractively(cmd *cobra.Command, request mcpclient.CreateIssueRequest) (proceed bool, err error) {
	interactive, _ := cmd.Flags().GetBool("interactive")
	if !interactive {
		return true, nil // Proceed if not interactive
	}

	fmt.Println("\n--- Issue Details ---")
	fmt.Printf("Project Key: %s\n", request.ProjectKey)
	fmt.Printf("Issue Type:  %s\n", request.IssueType)
	fmt.Printf("Summary:     %s\n", request.Summary)
	fmt.Printf("Description:\n%s\n", request.Description)
	fmt.Println("---------------------")
	fmt.Print("Create this issue? [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		Log.Error().Err(err).Msg("Failed to read user input for confirmation")
		fmt.Println("\nError reading input:", err)
		return false, err // Return error if input reading fails
	}

	cleanedInput := strings.ToLower(strings.TrimSpace(input))
	if cleanedInput != "y" && cleanedInput != "yes" {
		Log.Info().Msg("User aborted issue creation.")
		fmt.Println("Aborted.")
		return false, nil // User aborted, no error
	}

	Log.Debug().Msg("User confirmed issue creation.")
	return true, nil // User confirmed
}

// formatOutput formats the successful creation response based on the output flag.
// Updated signature to accept io.Writer
func formatOutput(cmd *cobra.Command, resp *mcpclient.CreateIssueResponse, out io.Writer) error {
	outputFormat, _ := cmd.Flags().GetString("output")
	Log.Debug().Str("format", outputFormat).Msg("Processing output for created issue")

	if outputFormat == "json" {
		Log.Debug().Msg("Marshalling created issue to JSON")
		jsonData, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			Log.Error().Err(err).Msg("Failed to marshal created issue to JSON")
			// Return error, but log that the issue was created
			return fmt.Errorf("issue created successfully (Key: %s), but failed to format result as JSON: %w", resp.Key, err)
		}
		fmt.Fprintln(out, string(jsonData)) // Use fmt.Fprintln with out
	} else {
		// Default to text output
		Log.Debug().Msg("Formatting created issue as text")
		// Use fmt.Fprintf with out
		fmt.Fprintf(out, "Successfully created JIRA issue:\nKey: %s\nURL: %s\n", resp.Key, resp.Self)
	}
	return nil
}

// --- Command Runner ---

// createCmdRunner holds the dependencies for the create command.
type createCmdRunner struct {
	configProvider    ConfigProvider
	llmClient         llm.Client // Use the llm.Client interface
	mcpClient         MCPClient  // Use the MCPClient interface directly
	projectMapper     ProjectMapper
	issueTypeResolver IssueTypeResolver
}

// newCreateCmdRunner creates a new runner, fetching dependencies from the central Provider.
func newCreateCmdRunner() (*createCmdRunner, error) {
	// Fetch dependencies from the central provider
	provider, err := GetProvider()
	if err != nil {
		// Log the error here as we can't return it easily to Cobra's init
		Log.Error().Err(err).Msg("Failed to initialize dependency provider in newCreateCmdRunner")
		// Return a partially initialized runner or handle error differently?
		// For now, return the error to prevent command setup if provider fails.
		return nil, fmt.Errorf("failed to initialize dependencies: %w", err)
	}

	// Check if essential providers are nil (e.g., if GetProvider logged warnings but returned)
	if provider.Config == nil {
		return nil, fmt.Errorf("config provider is nil after initialization")
	}
	// LLM and MCP clients might be nil if config/API keys are missing, commands should handle this.

	return &createCmdRunner{
		configProvider:    provider.Config,
		llmClient:         provider.LLM,                // Get from central provider
		mcpClient:         provider.MCP,                // Get from central provider
		projectMapper:     &DefaultProjectMapper{},     // Use exported type
		issueTypeResolver: &DefaultIssueTypeResolver{}, // Use exported type
	}, nil
}

// NewCreateCmdRunnerForTest creates a runner with explicitly provided dependencies for testing.
func NewCreateCmdRunnerForTest(cp ConfigProvider, llmClient llm.Client, mcpClient MCPClient, pm ProjectMapper, ir IssueTypeResolver) *createCmdRunner {
	return &createCmdRunner{
		configProvider:    cp,
		llmClient:         llmClient,
		mcpClient:         mcpClient,
		projectMapper:     pm, // Assume pm is *DefaultProjectMapper or compatible
		issueTypeResolver: ir, // Assume ir is *DefaultIssueTypeResolver or compatible
	}
}

// Run executes the logic for the create command using injected dependencies.
func (r *createCmdRunner) Run(cmd *cobra.Command, args []string) error {
	// Load configurations using helper
	loadedCfgs, err := loadAllConfigs(r.configProvider)
	if err != nil {
		// Specific user messages added in loadAllConfigs
		// Logged there too, just return the error for Cobra
		return err
	}

	// --- LLM Interaction ---
	userInput := strings.Join(args, " ")
	ctx := context.Background() // Create context for LLM and MCP calls

	// Check if LLM Client was initialized
	if r.llmClient == nil {
		err := fmt.Errorf("LLM client not initialized. Check configuration (provider, API key)")
		Log.Error().Err(err).Msg("LLM client is nil in createCmdRunner.Run")
		fmt.Fprintln(cmd.ErrOrStderr(), "Error: LLM client not initialized.")
		fmt.Fprintln(cmd.ErrOrStderr(), "Please check your LLM provider configuration and API key setup ('tix config show', 'tix config set-key').")
		return err
	}

	// Call LLM Client
	Log.Debug().Msg("Calling LLM client to generate ticket details...")
	llmResponse, err := r.llmClient.GenerateTicketDetails(ctx, userInput, loadedCfgs.systemPrompt, loadedCfgs.contextData)
	if err != nil {
		Log.Error().Err(err).Msg("LLM client GenerateTicketDetails failed")
		// Provide user feedback based on error type using switch
		switch {
		case errors.Is(err, config.ErrAPIKeyNotFound):
			fmt.Fprintln(cmd.ErrOrStderr(), "Error: LLM API key not found.")
			fmt.Fprintf(cmd.ErrOrStderr(), "Please store it using 'tix config set-key <your-key>' or set the %s environment variable.\n", config.EnvAPIKeyName)
		case errors.Is(err, llm.ErrLLMCompletion):
			fmt.Fprintf(cmd.ErrOrStderr(), "Error communicating with the LLM API: %v\n", err)
			fmt.Fprintln(cmd.ErrOrStderr(), "Please check your network connection and API key/endpoint configuration.")
		case errors.Is(err, llm.ErrLLMResponseParse), errors.Is(err, llm.ErrLLMResponseJSONFind), errors.Is(err, llm.ErrLLMResponseJSONUnmarshal), errors.Is(err, llm.ErrLLMResponseMissingField):
			fmt.Fprintf(cmd.ErrOrStderr(), "Error processing the response from the LLM: %v\n", err)
			fmt.Fprintln(cmd.ErrOrStderr(), "The LLM might have returned an unexpected format. Check logs for details.")
		default:
			fmt.Fprintf(cmd.ErrOrStderr(), "An unexpected error occurred during LLM processing: %v\n", err)
		}
		return err // Return the original error
	}
	Log.Info().Msg("LLM processing successful.") // Simplified log message

	// --- Map Project Name Suggestion ---
	mappedProjectKey, matchedProjectLink, err := r.projectMapper.MapSuggestionToKey(llmResponse.ProjectNameSuggestion, loadedCfgs.linksConfig)
	if err != nil {
		switch {
		case errors.Is(err, config.ErrProjectMappingFailed):
			fmt.Fprintf(cmd.ErrOrStderr(), "Error: Could not map LLM's project suggestion '%s' to a known project key.\n", llmResponse.ProjectNameSuggestion)
			fmt.Fprintln(cmd.ErrOrStderr(), "Please check your ~/.ticketron/links.yaml file or the LLM's output.")
		default:
			fmt.Fprintf(cmd.ErrOrStderr(), "An unexpected error occurred during project mapping: %v\n", err)
		}
		// Logged in MapSuggestionToKey, just return
		return err
	}

	// --- Determine Final Issue Type ---
	issueTypeFlag, _ := cmd.Flags().GetString("type") // Ignore error, default is ""
	finalIssueType := r.issueTypeResolver.Resolve(issueTypeFlag, matchedProjectLink, mappedProjectKey)
	Log.Debug().Str("final_issue_type", finalIssueType).Msg("Determined final issue type")

	// --- MCP Client Interaction ---
	Log.Debug().Msg("Preparing to call MCP server...")

	// Check if MCP Client was initialized
	if r.mcpClient == nil {
		err := fmt.Errorf("MCP client not initialized. Check MCP server URL configuration")
		Log.Error().Err(err).Msg("MCP client is nil in createCmdRunner.Run")
		fmt.Fprintln(cmd.ErrOrStderr(), "Error: MCP client not initialized.")
		fmt.Fprintln(cmd.ErrOrStderr(), "Please check the 'mcp_server_url' in your configuration ('tix config show').")
		return err
	}
	// Use the injected MCP client directly: r.mcpClient

	// Prepare CreateIssue Request
	request := mcpclient.CreateIssueRequest{
		ProjectKey:  mappedProjectKey,
		Summary:     llmResponse.Summary,
		Description: llmResponse.Description,
		IssueType:   finalIssueType,
	}
	Log.Debug().Interface("mcp_request", request).Msg("Prepared MCP request")

	// --- Interactive Confirmation ---
	proceed, err := confirmInteractively(cmd, request)
	if err != nil {
		// Error reading input
		return err
	}
	if !proceed {
		// User aborted
		return nil // Graceful exit
	}

	// Call CreateIssue
	Log.Debug().Msg("Creating JIRA issue via MCP...")
	resp, err := r.mcpClient.CreateIssue(ctx, request) // Use r.mcpClient
	if err != nil {
		Log.Error().Err(err).Msg("Failed to create JIRA issue via MCP")
		// Provide user feedback based on MCP client errors using switch
		switch {
		case errors.Is(err, mcpclient.ErrRequestExecute):
			fmt.Fprintf(cmd.ErrOrStderr(), "Error connecting to the MCP server: %v\n", err)
			fmt.Fprintln(cmd.ErrOrStderr(), "Please ensure the MCP server is running and the URL is correct.")
		case errors.Is(err, mcpclient.ErrMCPServerError):
			fmt.Fprintf(cmd.ErrOrStderr(), "MCP server returned an error: %v\n", err) // Error includes server message
		case errors.Is(err, mcpclient.ErrMCPServerErrorUnparseable):
			fmt.Fprintf(cmd.ErrOrStderr(), "MCP server returned an error with an unparseable response body: %v\n", err)
		case errors.Is(err, mcpclient.ErrResponseDecode):
			fmt.Fprintf(cmd.ErrOrStderr(), "Failed to decode the success response from the MCP server: %v\n", err)
		default:
			fmt.Fprintf(cmd.ErrOrStderr(), "An unexpected error occurred while creating the issue via MCP: %v\n", err)
		}
		return err // Return original error
	}

	// Handle Success Response
	Log.Info().Str("issue_key", resp.Key).Str("issue_url", resp.Self).Msg("Successfully created JIRA issue")

	// Handle output format using helper - pass cmd's output writer
	if err := formatOutput(cmd, resp, cmd.OutOrStdout()); err != nil {
		return err
	}

	return nil // Return nil on success
}

// --- Cobra Command Definition ---

// Flags for the create command (still needed for Cobra)
var (
	issueType       string // Bound variable for the flag
	issueSummary    string // Not used by core logic anymore, but kept for potential future use/consistency
	projectKey      string // Not used by core logic anymore
	description     string // Not used by core logic anymore
	interactiveFlag bool   // Added for interactive confirmation
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create [your issue description here...]",
	Short: "Create a new JIRA issue from a description",
	Long: `Creates a new JIRA issue by processing the provided description
using an LLM and interacting with the configured Jira MCP server.`,
	Args: cobra.MinimumNArgs(1), // Require at least one argument for the description
	// RunE will be set in init()
}

func init() {
	rootCmd.AddCommand(createCmd)

	// Initialize the runner with dependencies
	runner, err := newCreateCmdRunner()
	if err != nil {
		// If provider init fails, we can't set up the command.
		// Log the error globally and exit or panic.
		// Using panic here because this happens during init().
		panic(fmt.Sprintf("Failed to initialize create command runner: %v", err))
	}

	// Set the RunE function for the command
	createCmd.RunE = runner.Run // Assign the Run method of the runner instance

	// Add flags for create command
	// Note: We bind to package-level vars, but the runner reads flags directly via cmd.Flags().GetString()
	createCmd.Flags().StringVarP(&issueType, "type", "t", "", "Specify the JIRA issue type (e.g., Task, Bug) - overrides LLM suggestion and defaults")
	// Keep other flags for potential future direct use or help text, even if not used by core LLM flow
	createCmd.Flags().StringVarP(&issueSummary, "summary", "s", "", "[Optional] Specify the issue summary directly (currently unused by core logic)")
	createCmd.Flags().StringVarP(&projectKey, "project", "p", "", "[Optional] Specify the JIRA project key directly (currently unused by core logic)")
	createCmd.Flags().StringVarP(&description, "description", "d", "", "[Optional] Specify the issue description directly (currently unused by core logic)")
	createCmd.Flags().BoolVarP(&interactiveFlag, "interactive", "i", false, "Prompt for confirmation before creating the issue.") // Added flag
}

package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	openai "github.com/sashabaranov/go-openai" // Added openai import
	keyring "github.com/zalando/go-keyring"

	"github.com/karolswdev/ticketron/internal/config"
	"github.com/karolswdev/ticketron/internal/llm" // Added llm import
	"github.com/karolswdev/ticketron/internal/mcpclient"
)

// --- Concrete Implementations of Shared Interfaces ---

// defaultConfigProvider implements the ConfigProvider interface using the actual config package functions.
// Exported for potential use in tests directly.
type DefaultConfigProvider struct{}

func (p *DefaultConfigProvider) LoadConfig() (*config.AppConfig, error) {
	return config.LoadConfig("") // Pass empty string for default behavior
}

func (p *DefaultConfigProvider) LoadLinks() (*config.LinksConfig, error) {
	// LoadLinks returns LinksConfig, not *LinksConfig. Adjusting interface might be better,
	// but for now, we return a pointer to the loaded struct.
	links, err := config.LoadLinks("") // Pass empty string for default behavior
	if err != nil {
		return nil, err
	}
	return &links, nil // Corrected: Use & instead of &amp;
}

func (p *DefaultConfigProvider) LoadSystemPrompt() (string, error) {
	return config.LoadSystemPrompt("") // Pass empty string for default behavior
}

func (p *DefaultConfigProvider) LoadContext() (string, error) {
	return config.LoadContext("") // Pass empty string for default behavior
}

func (p *DefaultConfigProvider) GetAPIKey() (string, error) {
	return config.GetAPIKey()
}

// CreateDefaultConfigFiles calls the underlying config function to create default files.
// It ignores the configDir parameter as the underlying function determines the path.
func (p *DefaultConfigProvider) CreateDefaultConfigFiles(configDir string) error {
	// Ignore configDir, call the function that handles directory creation internally
	return config.CreateDefaultConfigFiles("") // Pass empty string for default behavior
}

// EnsureConfigDir calls the underlying config function to ensure the config directory exists.
func (p *DefaultConfigProvider) EnsureConfigDir() (string, error) {
	return config.EnsureConfigDir("") // Pass empty string for default behavior
}

// defaultMCPClient implements the MCPClient interface.
type defaultMCPClient struct {
	client *mcpclient.Client // Store the actual MCP client instance (pointer)
}

func newDefaultMCPClient(cfg *config.AppConfig) (MCPClient, error) {
	// Check for MCP Server URL before creating client
	if cfg.MCPServerURL == "" {
		homeDir, _ := os.UserHomeDir() // Best effort
		expectedPath := filepath.Join(homeDir, ".ticketron", "config.yaml")
		err := fmt.Errorf("MCP server URL is missing")
		// Log is not available here directly, consider returning a more specific error
		// or having the caller log it. For now, just return the error.
		// Error is returned, logging should happen in the caller (e.g., RunE)
		// Log.Error().Err(err).Str("path", expectedPath).Msg("Ensure 'mcp_server_url' is set in config or via TICKETRON_MCP_SERVER_URL env var")
		return nil, fmt.Errorf("%w: Ensure 'mcp_server_url' is set in %s or via TICKETRON_MCP_SERVER_URL env var", err, expectedPath)
	}

	c, err := mcpclient.New(cfg)
	if err != nil {
		// Error is returned, logging should happen in the caller (e.g., RunE)
		// Log.Error().Err(err).Msg("Failed to initialize MCP client")
		return nil, fmt.Errorf("failed to initialize MCP client: %w", err) // Wrap error
	}
	Log.Debug().Msg("MCP Client created successfully.") // Uncommented and kept as Debug
	return &defaultMCPClient{client: c}, nil            // Corrected: Use & instead of &amp;
}

func (m *defaultMCPClient) CreateIssue(ctx context.Context, req mcpclient.CreateIssueRequest) (*mcpclient.CreateIssueResponse, error) {
	return m.client.CreateIssue(ctx, req)
}

// SearchIssues calls the underlying client's SearchIssues method.
func (m *defaultMCPClient) SearchIssues(ctx context.Context, req mcpclient.SearchIssuesRequest) (*mcpclient.SearchIssuesResponse, error) {
	return m.client.SearchIssues(ctx, req)
}

// DefaultMCPClientWrapper wraps the concrete mcpclient.Client to satisfy the MCPClient interface for testing.
// Exported for use in tests.
type DefaultMCPClientWrapper struct {
	Client *mcpclient.Client
}

func (w *DefaultMCPClientWrapper) CreateIssue(ctx context.Context, req mcpclient.CreateIssueRequest) (*mcpclient.CreateIssueResponse, error) {
	if w.Client == nil {
		return nil, fmt.Errorf("wrapped mcpclient.Client is nil")
	}
	return w.Client.CreateIssue(ctx, req)
}

func (w *DefaultMCPClientWrapper) SearchIssues(ctx context.Context, req mcpclient.SearchIssuesRequest) (*mcpclient.SearchIssuesResponse, error) {
	if w.Client == nil {
		return nil, fmt.Errorf("wrapped mcpclient.Client is nil")
	}
	return w.Client.SearchIssues(ctx, req)
}

// --- Keyring Client Implementation ---

// defaultKeyringClient implements the KeyringClient interface using the actual keyring package.
type defaultKeyringClient struct {
	// No fields needed for the basic Set operation using the package directly.
}

// Set calls the underlying keyring package's Set function.
func (k *defaultKeyringClient) Set(service, user, password string) error {
	// Note: Consider adding configuration for the keyring backend if needed.
	// For now, it uses the default backend determined by the library.
	return keyring.Set(service, user, password)
}

// GetAPIKey calls the underlying config function to retrieve the API key.
// Note: The service and user parameters are currently unused by the underlying
// config.GetAPIKey function but are kept for interface compatibility.
func (k *defaultKeyringClient) GetAPIKey(service, user string) (string, error) {
	// TODO: Consider using service/user if the underlying implementation changes.
	return config.GetAPIKey()
}

// --- Central Provider ---

// Provider serves as a central dependency injection container, aggregating the various
// service interfaces (like ConfigProvider, MCPClient, KeyringClient) required by
// the application's commands. This structure simplifies passing dependencies down
// the call stack and facilitates mocking during testing.
type Provider struct {
	Config  ConfigProvider
	MCP     MCPClient
	Keyring KeyringClient
	LLM     llm.Client // Added LLM client interface
}

// GetProvider is the factory function responsible for initializing and returning a
// fully configured Provider instance. It sets up the concrete implementations for
// each required service interface (e.g., defaultConfigProvider, defaultMCPClient,
// defaultKeyringClient). It handles loading the initial application configuration
// and attempts to initialize the MCP client if configured. Errors during critical
// initialization steps (like loading AppConfig) are returned. Warnings may be logged
// for non-critical failures (like MCP client init if URL is missing).
func GetProvider() (*Provider, error) {
	// Initialize Config Provider
	cfgProvider := &DefaultConfigProvider{} // Corrected: Use & instead of &amp;
	appCfg, err := cfgProvider.LoadConfig()
	if err != nil {
		// Error is returned, logging should happen in the caller (e.g., RunE)
		// Log.Error().Err(err).Msg("Failed to load application config during provider initialization")
		return nil, fmt.Errorf("failed to load application config: %w", err)
	}

	// Initialize MCP Client (conditionally based on config)
	var mcpClient MCPClient
	// Only initialize MCP if URL is present, otherwise commands needing it will fail later if they try to use a nil client.
	// Commands that don't need MCP should still function.
	if appCfg.MCPServerURL != "" {
		mcpClient, err = newDefaultMCPClient(appCfg)
		if err != nil {
			Log.Warn().Err(err).Msg("Failed to initialize MCP client during provider setup. MCP operations might fail.") // Uncommented and kept as Warn
			// Consider returning the error if MCP is essential for most operations:
			// return nil, fmt.Errorf("failed to initialize MCP client: %w", err)
		}
	} else {
		Log.Debug().Msg("MCP Server URL not configured. MCP client not initialized.") // Uncommented and kept as Debug
	}

	// Initialize Keyring Client
	keyringClient := &defaultKeyringClient{} // Corrected: Use & instead of &amp;

	// Initialize LLM Client based on config
	var llmClient llm.Client
	apiKey, keyErr := cfgProvider.GetAPIKey()           // Get API key once
	if keyErr != nil && appCfg.LLM.Provider != "mock" { // Allow mock provider without key
		// Log warning but don't fail provider init; commands needing LLM will fail later
		Log.Warn().Err(keyErr).Msg("Failed to get LLM API key during provider setup. LLM operations might fail.")
	}

	switch appCfg.LLM.Provider {
	case "openai":
		if keyErr == nil { // Only proceed if key was retrieved
			Log.Debug().Str("provider", "openai").Msg("Initializing OpenAI LLM client")
			// Log the key being used (MASK SENSITIVE PARTS IN REAL LOGS if necessary, but ok for test dummy key)
			Log.Debug().Str("apiKeyUsed", apiKey).Msg("API Key retrieved for OpenAI client")
			openAIConfig := openai.DefaultConfig(apiKey)
			if appCfg.LLM.OpenAI.BaseURL != "" {
				openAIConfig.BaseURL = appCfg.LLM.OpenAI.BaseURL
				Log.Debug().Str("baseURLUsed", openAIConfig.BaseURL).Msg("Using custom OpenAI BaseURL")
			} else {
				Log.Debug().Msg("Using default OpenAI BaseURL")
			}
			openaiSdkClient := openai.NewClientWithConfig(openAIConfig)
			llmClient, err = llm.NewOpenAIClient(openaiSdkClient, appCfg.LLM.OpenAI.ModelName)
			if err != nil {
				Log.Warn().Err(err).Msg("Failed to initialize OpenAI client. LLM operations might fail.")
				// Don't return error here, let commands fail if they need LLM
			}
		} else {
			Log.Warn().Msg("OpenAI provider selected but API key retrieval failed. LLM client not initialized.")
		}
	// case "anthropic": // Placeholder
	// 	Log.Warn().Msg("Anthropic provider not yet implemented.")
	// case "ollama": // Placeholder
	// 	Log.Warn().Msg("Ollama provider not yet implemented.")
	case "mock": // Allow a mock provider for testing potentially
		Log.Info().Msg("Using mock LLM provider (if implemented).")
		// llmClient = NewMockLLMClient() // Example
	default:
		Log.Warn().Str("provider", appCfg.LLM.Provider).Msg("Unsupported LLM provider specified in config. LLM client not initialized.")
	}

	// Construct and return the Provider
	provider := &Provider{ // Corrected: Use & instead of &amp;
		Config:  cfgProvider,
		MCP:     mcpClient, // This might be nil if URL wasn't set or init failed
		Keyring: keyringClient,
		LLM:     llmClient, // Assign the initialized LLM client (might be nil)
	}

	Log.Debug().Msg("Service Provider initialized successfully.") // Uncommented and kept as Debug
	return provider, nil
}

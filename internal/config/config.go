package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/zalando/go-keyring"
	"gopkg.in/yaml.v3"

	"github.com/spf13/viper"
)

const (
	// DefaultConfigFileName is the standard name for the main configuration file.
	DefaultConfigFileName = "config.yaml"
	// DefaultLinksFileName is the standard name for the project links file.
	DefaultLinksFileName = "links.yaml"
	// DefaultPromptFileName is the standard name for the system prompt file.
	DefaultPromptFileName = "system_prompt.txt"
	// DefaultContextFileName is the standard name for the context file.
	DefaultContextFileName = "context.md"
	// DefaultConfigDirName is the standard name for the configuration directory within the user's home directory.
	DefaultConfigDirName = ".ticketron"
	// ConfigDirEnvVar is the environment variable used to override the default configuration directory path.
	ConfigDirEnvVar = "TICKETRON_CONFIG_DIR"
)

// EnsureConfigDir checks if the configuration directory exists, creating it if necessary.
// It prioritizes baseDir if provided. If baseDir is empty, it checks the TICKETRON_CONFIG_DIR
// environment variable. If the environment variable is also empty or unset, it defaults to ~/.ticketron.
// The function ensures the path exists and is a directory with appropriate permissions (0700).
// It returns the validated configuration directory path or an error if creation/validation fails.
func EnsureConfigDir(baseDir string) (string, error) {
	var configDirPath string
	var err error

	if baseDir != "" {
		// Use provided baseDir
		configDirPath = baseDir
		log.Debug().Str("path", configDirPath).Msg("Using provided base directory path")
	} else {
		// Check environment variable if baseDir is empty
		envDir := os.Getenv(ConfigDirEnvVar)
		if envDir != "" {
			configDirPath = envDir
			log.Debug().Str("path", configDirPath).Str("env_var", ConfigDirEnvVar).Msg("Using config directory path from environment variable")
		} else {
			// Default behavior: use ~/.ticketron
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("failed to get user home directory: %w", err)
			}
			configDirPath = filepath.Join(homeDir, DefaultConfigDirName)
			log.Debug().Str("path", configDirPath).Msg("Using default config directory path")
		}
	}

	info, err := os.Stat(configDirPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Info().Str("path", configDirPath).Msg("Config directory does not exist, attempting to create")
			// Directory does not exist, create it
			if mkdirErr := os.MkdirAll(configDirPath, 0700); mkdirErr != nil {
				log.Error().Err(mkdirErr).Str("path", configDirPath).Msg("Failed to create config directory")
				return "", fmt.Errorf("%w: %w", ErrConfigDirCreate, mkdirErr) // Use sentinel error
			}
			// Creation successful
			log.Info().Str("path", configDirPath).Msg("Successfully created config directory")
			return configDirPath, nil
		}
		// Another error occurred during stat
		log.Error().Err(err).Str("path", configDirPath).Msg("Failed to stat config directory path")
		return "", fmt.Errorf("%w: %w", ErrConfigDirStat, err) // Use sentinel error
	}

	// Path exists, check if it's a directory
	if !info.IsDir() {
		log.Error().Str("path", configDirPath).Msg("Config path exists but is not a directory")
		return "", ErrConfigDirNotDir // Use sentinel error
	}

	// Path exists and is a directory
	log.Debug().Str("path", configDirPath).Msg("Config directory exists and is a directory")
	return configDirPath, nil
}

// OpenAIConfig holds configuration specific to the OpenAI provider.
type OpenAIConfig struct {
	ModelName string `mapstructure:"model_name"`
	BaseURL   string `mapstructure:"base_url"` // Optional custom base URL
	// APIKey is handled separately via keyring/env var (GetAPIKey) for now
}

// LLMConfig holds configuration specific to the Language Model provider selection
// and common settings. Provider-specific settings are nested.
type LLMConfig struct {
	Provider string       `mapstructure:"provider"` // e.g., "openai", "anthropic", "ollama"
	OpenAI   OpenAIConfig `mapstructure:"openai"`
	// Add other providers like AnthropicConfig, OllamaConfig here later
}

// AppConfig holds the overall application configuration.
type AppConfig struct {
	MCPServerURL string    `mapstructure:"mcp_server_url"`
	LLM          LLMConfig `mapstructure:"llm"` // Embed the new LLMConfig
}

// LoadConfig loads the application configuration from the config file (e.g., ~/.ticketron/config.yaml or baseDir/config.yaml),
// environment variables (TICKETRON_*), and sets defaults.
// If baseDir is empty, it uses the default ~/.ticketron.
func LoadConfig(baseDir string) (*AppConfig, error) {
	configDir, err := EnsureConfigDir(baseDir) // EnsureConfigDir now respects TICKETRON_CONFIG_DIR
	if err != nil {
		// Error already logged in EnsureConfigDir
		return nil, fmt.Errorf("failed to ensure config directory: %w", err)
	}

	v := viper.New()

	// Set default values
	v.SetDefault("mcp_server_url", "http://localhost:8080")
	v.SetDefault("llm.provider", "openai")          // Default to openai
	v.SetDefault("llm.openai.model_name", "gpt-4o") // Default OpenAI model
	v.SetDefault("llm.openai.base_url", "")         // Default OpenAI base_url
	// No default for API key - use GetAPIKey() for retrieval

	// Configure Viper to read the config file
	configPath := filepath.Join(configDir, DefaultConfigFileName) // Define configPath here
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(configDir) // This now correctly points to the temp dir in tests
	log.Debug().Str("path", configPath).Msg("Attempting to load config file")

	// Configure environment variable overrides
	v.SetEnvPrefix("TICKETRON")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_")) // map nested keys like llm.model_name to TICKETRON_LLM_MODEL_NAME

	// Attempt to read the config file
	err = v.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if it's just not found
			log.Warn().Str("path", configPath).Msg("Config file not found. Using defaults and environment variables.")
		} else {
			// Config file was found but another error was produced
			log.Error().Err(err).Str("path", configPath).Msg("Failed to read config file")
			return nil, fmt.Errorf("%w: %w", ErrConfigRead, err) // Use sentinel error
		}
	} else {
		log.Debug().Str("path", configPath).Msg("Read config file successfully") // Log success only if ReadInConfig doesn't error
	}

	// Unmarshal the config into the struct
	var cfg AppConfig
	err = v.Unmarshal(&cfg)
	if err != nil {
		log.Error().Err(err).Str("path", configPath).Msg("Failed to unmarshal config file")
		return nil, fmt.Errorf("%w: %w", ErrConfigParse, err) // Use sentinel error
	}
	log.Debug().Str("path", configPath).Interface("config", cfg).Msg("Unmarshalled config successfully")

	return &cfg, nil
}

// ProjectLink defines the structure for a single project mapping.
type ProjectLink struct {
	Name             string `yaml:"name"`                         // User-friendly name/alias (case-insensitive match target)
	Key              string `yaml:"key"`                          // The actual JIRA project key
	DefaultIssueType string `yaml:"default_issue_type,omitempty"` // Optional default issue type
}

// LinksConfig holds the list of project links.
type LinksConfig struct {
	Projects []ProjectLink `yaml:"projects"`
}

// LoadLinks loads the project link configurations from the links file (e.g., ~/.ticketron/links.yaml or baseDir/links.yaml).
// It returns an empty LinksConfig if the file doesn't exist.
// It returns an error if the file exists but cannot be read or parsed.
// If baseDir is empty, it uses the default ~/.ticketron.
func LoadLinks(baseDir string) (LinksConfig, error) {
	var cfg LinksConfig // Initialize empty struct

	configDir, err := EnsureConfigDir(baseDir)
	if err != nil {
		// Error already logged in EnsureConfigDir
		return cfg, fmt.Errorf("failed to ensure config directory for links: %w", err)
	}

	linksPath := filepath.Join(configDir, DefaultLinksFileName)
	log.Debug().Str("path", linksPath).Msg("Attempting to load links file")

	fileBytes, err := os.ReadFile(linksPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Warn().Str("path", linksPath).Msg("Links file not found, returning empty links config")
			// File doesn't exist, which is acceptable. Return empty config.
			return cfg, nil
		}
		// Other error reading the file
		log.Error().Err(err).Str("path", linksPath).Msg("Failed to read links file")
		return cfg, fmt.Errorf("%w: %w", ErrLinksRead, err) // Use sentinel error
	}
	log.Debug().Str("path", linksPath).Int("bytes", len(fileBytes)).Msg("Read links file successfully")

	// File exists, attempt to parse it
	err = yaml.Unmarshal(fileBytes, &cfg)
	if err != nil {
		log.Error().Err(err).Str("path", linksPath).Msg("Failed to parse links file")
		return cfg, fmt.Errorf("%w: %w", ErrLinksParse, err) // Use sentinel error
	}
	log.Debug().Str("path", linksPath).Interface("links", cfg).Msg("Parsed links file successfully")

	// Ensure Projects slice is not nil if the file was empty or contained no projects
	if cfg.Projects == nil {
		cfg.Projects = []ProjectLink{}
	}

	return cfg, nil
}

// LoadSystemPrompt loads the system prompt text from the prompt file (e.g., ~/.ticketron/system_prompt.txt or baseDir/system_prompt.txt).
// It returns an empty string if the file doesn't exist.
// It returns an error if the file exists but cannot be read.
// If baseDir is empty, it uses the default ~/.ticketron.
func LoadSystemPrompt(baseDir string) (string, error) {
	configDir, err := EnsureConfigDir(baseDir)
	if err != nil {
		// Error already logged in EnsureConfigDir
		return "", fmt.Errorf("failed to ensure config directory for system prompt: %w", err)
	}

	promptPath := filepath.Join(configDir, DefaultPromptFileName)
	log.Debug().Str("path", promptPath).Msg("Attempting to load system prompt file")

	fileBytes, err := os.ReadFile(promptPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Warn().Str("path", promptPath).Msg("System prompt file not found, returning empty string")
			// File doesn't exist, which is acceptable. Return empty string.
			return "", nil
		}
		// Other error reading the file
		log.Error().Err(err).Str("path", promptPath).Msg("Failed to read system prompt file")
		return "", fmt.Errorf("%w: %w", ErrSystemPromptRead, err) // Use sentinel error
	}
	log.Debug().Str("path", promptPath).Int("bytes", len(fileBytes)).Msg("Read system prompt file successfully")

	// File exists and was read successfully
	return string(fileBytes), nil
}

// LoadContext loads the context text from the context file (e.g., ~/.ticketron/context.md or baseDir/context.md).
// It returns an empty string if the file doesn't exist.
// It returns an error if the file exists but cannot be read.
// If baseDir is empty, it uses the default ~/.ticketron.
func LoadContext(baseDir string) (string, error) {
	configDir, err := EnsureConfigDir(baseDir)
	if err != nil {
		// Error already logged in EnsureConfigDir
		return "", fmt.Errorf("failed to ensure config directory for context: %w", err)
	}

	contextPath := filepath.Join(configDir, DefaultContextFileName)
	log.Debug().Str("path", contextPath).Msg("Attempting to load context file")

	fileBytes, err := os.ReadFile(contextPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Warn().Str("path", contextPath).Msg("Context file not found, returning empty string")
			// File doesn't exist, which is acceptable. Return empty string.
			return "", nil
		}
		// Other error reading the file
		log.Error().Err(err).Str("path", contextPath).Msg("Failed to read context file")
		return "", fmt.Errorf("%w: %w", ErrContextRead, err) // Use sentinel error
	}
	log.Debug().Str("path", contextPath).Int("bytes", len(fileBytes)).Msg("Read context file successfully")

	// File exists and was read successfully
	return string(fileBytes), nil
}

// --- Default File Creation ---

const defaultConfigYAML = `# User-specific configuration for the Ticketron CLI (tix)
# Located at ~/.ticketron/config.yaml

# URL for the Jira MCP server used for interacting with Jira.
mcp_server_url: "http://localhost:8080" # Default, user should change if needed

# Configuration for the Large Language Model (LLM) used by Ticketron.
llm:
  # Specify the LLM provider to use ("openai", "anthropic", "ollama", etc.)
  provider: "openai"

  # Settings specific to the OpenAI provider
  openai:
    # Name or identifier of the OpenAI model to use.
    model_name: "gpt-4o" # Example: gpt-4, gpt-4o, gpt-3.5-turbo
    # Optional: Specify a custom base URL for the OpenAI API (e.g., for proxies)
    # base_url: ""

  # Example for Anthropic (add when implemented)
  # anthropic:
  #   model_name: "claude-3-opus-20240229"
  #   # api_key: Handled via keyring/env var

  # Example for Ollama (add when implemented)
  # ollama:
  #   model_name: "llama3"
  #   base_url: "http://localhost:11434" # Default Ollama URL

`

const defaultLinksYAML = `# ~/.ticketron/links.yaml
# Defines mappings between user-friendly project aliases and JIRA project keys.
# Also allows specifying a default issue type per project.
projects:
  - name: "My Project Alias" # User-friendly name used for matching (case-insensitive)
    key: "PROJ"             # The actual JIRA project key
    default_issue_type: "Task" # Optional: Default issue type for this project
  - name: "Backend Team"
    key: "BE"
  # Add more projects as needed
`

const defaultSystemPromptTXT = `You are an expert assistant specialized in creating JIRA tickets from user requests.
Your task is to process the user's input, which may include their raw request and potentially additional context provided separately, and generate a structured JIRA issue.

Input:
- User's natural language request.
- Optional: Additional context (e.g., from a context.md file).

Output:
Generate ONLY a JSON object containing the following fields:
- "project_alias": A suggested JIRA project alias (lowercase string, derived from the user request or context, matching an alias defined in the system's links.yaml).
- "summary": A concise and informative JIRA issue summary based on the user request.
- "description": A detailed JIRA issue description, formatted using markdown where appropriate, elaborating on the user request.
- "issue_type": The suggested JIRA issue type (e.g., "Task", "Bug", "Story", "Epic").

CRITICAL: Your response MUST contain ONLY the valid JSON object. Do not include any introductory text, explanations, apologies, markdown formatting around the JSON block (like json), or any other text outside the JSON structure itself.
`

const defaultContextMD = `# LLM Context for Ticketron JIRA Ticket Generation
# -------------------------------------------------
# This file provides optional, persistent context to the LLM.
# Populate the sections below with details relevant to your current work
# to help the LLM generate more accurate and context-aware JIRA tickets.
# Lines starting with '#' are comments and will be ignored.

## Current Focus / Active Projects
# Add high-level goals, active epics, sprints, or project names.
# Example:
# - Currently focused on the "User Authentication" epic (PROJ-123).
# - Sprint goal: Complete backend integration for SSO.
# - Project Alpha: Refactoring the reporting module.


## Key Technologies / Components
# List relevant technologies, services, repositories, or code components.
# Example:
# - Backend: Go (Gin framework), PostgreSQL
# - Frontend: React, TypeScript
# - Services: AuthSvc, NotificationSvc
# - Repo: github.com/example/project-alpha


## Common Acronyms / Jargon
# Define project-specific terms, acronyms, or team names.
# Example:
# - SSO: Single Sign-On
# - K8s: Kubernetes
# - The "Phoenix Team" is responsible for deployment automation.
# - TKT: Ticketron


## Recent Activity / Decisions
# Note down recent significant events, decisions, or blockers.
# Example:
# - Decision (2023-10-27): Decided to use Kafka for event streaming instead of RabbitMQ.
# - Blocked: Waiting for API keys for the external payment gateway.
# - Completed: Deployed v1.2.0 to staging environment yesterday.
`

// writeFileIfNotExists checks if a file exists. If not, it writes the provided content.
func writeFileIfNotExists(filePath string, content string, perm os.FileMode) error {
	_, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Info().Str("path", filePath).Msg("File does not exist, attempting to write default content")
			// File does not exist, write it
			errWrite := os.WriteFile(filePath, []byte(content), perm)
			if errWrite != nil {
				log.Error().Err(errWrite).Str("path", filePath).Msg("Failed to write default file content")
				return fmt.Errorf("%w: %w", ErrDefaultFileWrite, errWrite) // Use sentinel error
			}
			log.Info().Str("path", filePath).Msg("Successfully wrote default file content") // Replaced fmt.Printf
			return nil                                                                      // File created successfully
		}
		// Another error occurred during stat
		log.Error().Err(err).Str("path", filePath).Msg("Failed to stat file path")
		return fmt.Errorf("%w: %w", ErrDefaultFileStat, err) // Use sentinel error
	}
	// File already exists
	log.Debug().Str("path", filePath).Msg("File already exists, no action needed")
	return nil
}

// CreateDefaultConfigFiles ensures the configuration directory exists (using default or baseDir)
// and creates default configuration files (config.yaml, links.yaml, system_prompt.txt, context.md)
// within that directory if they do not already exist.
// If baseDir is empty, it uses the default ~/.ticketron.
func CreateDefaultConfigFiles(baseDir string) error {
	configDir, err := EnsureConfigDir(baseDir)
	if err != nil {
		// Error already logged in EnsureConfigDir
		return fmt.Errorf("failed to ensure config directory: %w", err)
	}

	filesToCreate := []struct {
		name    string
		content string
		perm    os.FileMode
	}{
		{DefaultConfigFileName, defaultConfigYAML, 0600},
		{DefaultLinksFileName, defaultLinksYAML, 0600},
		{DefaultPromptFileName, defaultSystemPromptTXT, 0644},
		{DefaultContextFileName, defaultContextMD, 0644},
	}

	for _, file := range filesToCreate {
		filePath := filepath.Join(configDir, file.name)
		log.Debug().Str("file", file.name).Msg("Ensuring default file") // Added log
		err := writeFileIfNotExists(filePath, file.content, file.perm)
		if err != nil {
			// Error already includes file path and specific write/stat error
			return err
		}
	}

	return nil
}

// --- API Key Handling ---

const (
	keyringServiceName = "ticketron"
	keyringUserName    = "openai_api_key" // Or a more generic name if supporting multiple LLMs later
	// EnvAPIKeyName defines the environment variable name used to look up the LLM API key
	// as a fallback if it's not found in the OS keychain.
	EnvAPIKeyName = "TICKETRON_LLM_API_KEY" // Exported constant
)

// ErrAPIKeyNotFound is returned when the API key cannot be found in any source.
var ErrAPIKeyNotFound = errors.New("OpenAI API key not found in OS keychain or environment variable " + EnvAPIKeyName)

// GetAPIKey retrieves the OpenAI API key.
// It first tries the OS keychain/keyring using the service "ticketron" and user "openai_api_key".
// If not found there, it checks the environment variable TICKETRON_LLM_API_KEY.
// If found in either location, it returns the key.
// If not found in either, it returns ErrAPIKeyNotFound.
func GetAPIKey() (string, error) {
	// 1. Try Keychain
	log.Debug().Str("service", keyringServiceName).Str("user", keyringUserName).Msg("Attempting to get API key from keychain")
	key, err := keyring.Get(keyringServiceName, keyringUserName)
	if err == nil {
		// Found in keychain
		log.Debug().Msg("API key retrieved successfully (from keychain)")
		return key, nil
	}

	// Check if the error is simply "not found"
	if !errors.Is(err, keyring.ErrNotFound) {
		// A different error occurred with the keychain (permissions, etc.)
		log.Error().Err(err).Str("service", keyringServiceName).Str("user", keyringUserName).Msg("Error reading key from keychain")
		return "", fmt.Errorf("%w: %w", ErrKeyringGet, err) // Use sentinel error
	}

	// 2. Try Environment Variable (keychain returned ErrNotFound)
	log.Warn().Str("service", keyringServiceName).Str("user", keyringUserName).Msgf("API key not found in keychain, checking environment variable %s", EnvAPIKeyName) // Added log
	key = os.Getenv(EnvAPIKeyName)
	if key != "" {
		// Found in environment variable
		log.Debug().Msg("API key retrieved successfully (from env var)")
		return key, nil
	}

	// 3. Not found anywhere
	log.Error().Str("env_var", EnvAPIKeyName).Msg("API key not found in environment variable either.") // Added log
	return "", ErrAPIKeyNotFound
}

// SetAPIKey stores the OpenAI API key securely in the OS keychain/keyring.
func SetAPIKey(apiKey string) error {
	log.Debug().Str("service", keyringServiceName).Str("user", keyringUserName).Msg("Attempting to set API key in keychain")
	err := keyring.Set(keyringServiceName, keyringUserName, apiKey)
	if err != nil {
		log.Error().Err(err).Str("service", keyringServiceName).Str("user", keyringUserName).Msg("Failed to set API key in keychain")
		return fmt.Errorf("%w: %w", ErrKeyringSet, err) // Use sentinel error
	}
	log.Info().Str("service", keyringServiceName).Str("user", keyringUserName).Msg("API key stored successfully in keychain")
	return nil
}

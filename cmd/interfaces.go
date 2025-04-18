package cmd

import (
	"context"

	"github.com/karolswdev/ticketron/internal/config"
	"github.com/karolswdev/ticketron/internal/mcpclient"
)

// ConfigProvider defines an interface for components that load various configuration
// aspects of the Ticketron application, such as the main config, project links,
// prompts, context, and API keys. It also includes methods for managing the
// configuration directory and default files. This abstraction allows for easier testing
// by mocking configuration loading behavior.
type ConfigProvider interface {
	LoadConfig() (*config.AppConfig, error)
	LoadLinks() (*config.LinksConfig, error)
	LoadSystemPrompt() (string, error)
	LoadContext() (string, error)
	GetAPIKey() (string, error)
	CreateDefaultConfigFiles(configDir string) error // Added for config init
	EnsureConfigDir() (string, error)                // Added for config locate
}

// LLMProcessor interface removed, use llm.Client directly.

// MCPClient defines an interface for components that communicate with the
// Jira MCP (Model Context Protocol) server. It abstracts the operations of
// creating and searching for Jira issues via the MCP API.
type MCPClient interface {
	CreateIssue(ctx context.Context, req mcpclient.CreateIssueRequest) (*mcpclient.CreateIssueResponse, error)
	SearchIssues(ctx context.Context, req mcpclient.SearchIssuesRequest) (*mcpclient.SearchIssuesResponse, error)
}

// ProjectMapper defines an interface for components that can map a project name
// suggestion (potentially from the LLM) to a valid Jira project key using the
// loaded LinksConfig. It returns the mapped key and the specific ProjectLink matched.
type ProjectMapper interface {
	MapSuggestionToKey(suggestion string, links *config.LinksConfig) (projectKey string, matchedLink *config.ProjectLink, err error)
}

// IssueTypeResolver defines an interface for components that determine the final
// issue type to be used for ticket creation, considering the type provided via command flag,
// the default type specified in a matched ProjectLink, and a hardcoded default.
type IssueTypeResolver interface {
	Resolve(flagType string, projectLink *config.ProjectLink, defaultType string) string
}

// KeyringClient defines an interface for components that interact with the
// operating system's secure credential store (keychain/keyring). It abstracts
// the operations of setting and retrieving secrets, specifically the LLM API key.
type KeyringClient interface {
	Set(service, user, password string) error
	GetAPIKey(service, user string) (string, error) // Added for config show
	// Add Delete later if needed by other commands
}

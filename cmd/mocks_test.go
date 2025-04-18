package cmd

import (
	"context" // Added for MCPClient interface

	"github.com/stretchr/testify/mock"

	// Corrected import paths
	"github.com/karolswdev/ticketron/internal/config" // Correct path
	// Added llm import
	"github.com/karolswdev/ticketron/internal/mcpclient" // Correct path
)

// --- Mock ConfigLoader ---

type MockConfigLoader struct {
	mock.Mock // Implements ConfigProvider
}

// --- Mock ConfigProvider ---
// Renamed MockConfigLoader to MockConfigProvider for clarity
type MockConfigProvider struct {
	mock.Mock
}

// LoadConfig matches ConfigProvider interface
func (m *MockConfigProvider) LoadConfig() (*config.AppConfig, error) {
	args := m.Called()
	cfg, _ := args.Get(0).(*config.AppConfig) // Use AppConfig
	return cfg, args.Error(1)
}

// LoadLinks matches ConfigProvider interface
func (m *MockConfigProvider) LoadLinks() (*config.LinksConfig, error) {
	args := m.Called()
	cfg, _ := args.Get(0).(*config.LinksConfig)
	return cfg, args.Error(1)
}

// LoadSystemPrompt matches ConfigProvider interface
func (m *MockConfigProvider) LoadSystemPrompt() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

// LoadContext matches ConfigProvider interface
func (m *MockConfigProvider) LoadContext() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

// GetAPIKey matches ConfigProvider interface
func (m *MockConfigProvider) GetAPIKey() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

// CreateDefaultConfigFiles matches ConfigProvider interface signature
func (m *MockConfigProvider) CreateDefaultConfigFiles(configDir string) error {
	args := m.Called(configDir)
	return args.Error(0)
}

// EnsureConfigDir matches ConfigProvider interface
func (m *MockConfigProvider) EnsureConfigDir() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

// --- Mock MCPClient ---

type MockMCPClient struct {
	mock.Mock // Implements MCPClient
}

// CreateIssue matches MCPClient interface
func (m *MockMCPClient) CreateIssue(ctx context.Context, req mcpclient.CreateIssueRequest) (*mcpclient.CreateIssueResponse, error) {
	args := m.Called(ctx, req)
	resp, _ := args.Get(0).(*mcpclient.CreateIssueResponse)
	return resp, args.Error(1)
}

// SearchIssues matches MCPClient interface
func (m *MockMCPClient) SearchIssues(ctx context.Context, req mcpclient.SearchIssuesRequest) (*mcpclient.SearchIssuesResponse, error) {
	args := m.Called(ctx, req)
	resp, _ := args.Get(0).(*mcpclient.SearchIssuesResponse)
	return resp, args.Error(1)
}

// MockLLMClient moved to mocks.go

// --- Mock KeyringClient ---

type MockKeyringClient struct {
	mock.Mock // Implements KeyringClient
}

// Set matches KeyringClient interface
func (m *MockKeyringClient) Set(service, user, password string) error {
	args := m.Called(service, user, password)
	return args.Error(0)
}

// GetAPIKey matches KeyringClient interface (corrected name and params)
func (m *MockKeyringClient) GetAPIKey(service, user string) (string, error) {
	args := m.Called(service, user)
	return args.String(0), args.Error(1)
}

// Removed Delete as it's not in the current interface definition

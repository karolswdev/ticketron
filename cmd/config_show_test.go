package cmd

import (
	"bytes"
	"errors"
	"testing"

	"github.com/karolswdev/ticketron/internal/config" // Use correct import path
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	// "github.com/zalando/go-keyring" // Not needed directly, using config.ErrAPIKeyNotFound
)

func TestConfigShowCmd_Success(t *testing.T) {
	mockProvider := new(MockConfigProvider)
	mockKeyring := new(MockKeyringClient)
	var out bytes.Buffer

	// Setup mock expectations
	testConfig := &config.AppConfig{
		MCPServerURL: "http://mcp.example.com",
		LLM:          config.LLMConfig{Provider: "openai", OpenAI: config.OpenAIConfig{ModelName: "gpt-test"}}, // Updated config structure
		// Add other fields as needed for output verification
	}
	mockProvider.On("LoadConfig").Return(testConfig, nil)
	// Use correct keyring user constant from config_show.go
	mockKeyring.On("GetAPIKey", keyringService, keyringUser).Return("test-key-****-end", nil)

	// cmd := &cobra.Command{} // Not needed

	// Call configShowRunE directly
	err := configShowRunE(mockProvider, mockKeyring, &out)

	// Assertions
	assert.NoError(t, err)
	assert.Contains(t, out.String(), "Current Ticketron Configuration:")         // Exact title
	assert.Contains(t, out.String(), "  MCP Server URL: http://mcp.example.com") // Exact format with spaces
	assert.Contains(t, out.String(), "  LLM Provider:   openai")                 // Check provider
	assert.Contains(t, out.String(), "    OpenAI Model: gpt-test")               // Check model name
	// Check output based on config_show.go logic
	assert.Contains(t, out.String(), "  LLM API Key:    Set (use 'tix config set-key' to change)") // Exact format and status
	mockProvider.AssertExpectations(t)
	mockKeyring.AssertExpectations(t)
}

func TestConfigShowCmd_ConfigLoadError(t *testing.T) {
	mockProvider := new(MockConfigProvider)
	mockKeyring := new(MockKeyringClient)
	var out bytes.Buffer

	// Setup mock expectation for LoadConfig to return an error
	expectedErr := errors.New("failed to read config file")
	mockProvider.On("LoadConfig").Return((*config.AppConfig)(nil), expectedErr)

	// cmd := &cobra.Command{} // Not needed

	// Call configShowRunE directly
	err := configShowRunE(mockProvider, mockKeyring, &out)

	// Assertions
	assert.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
	assert.Contains(t, err.Error(), "error loading configuration:") // Exact wrapped error message
	assert.Empty(t, out.String())
	mockProvider.AssertExpectations(t)
	mockKeyring.AssertNotCalled(t, "GetAPIKey", mock.Anything, mock.Anything) // Keyring should not be called
}

func TestConfigShowCmd_KeyringGetError_NotFound(t *testing.T) {
	mockProvider := new(MockConfigProvider)
	mockKeyring := new(MockKeyringClient)
	var out bytes.Buffer

	// Setup mock expectations
	testConfig := &config.AppConfig{
		MCPServerURL: "http://mcp.example.com",
		LLM:          config.LLMConfig{Provider: "openai", OpenAI: config.OpenAIConfig{ModelName: "gpt-test"}}, // Updated config structure
	}
	mockProvider.On("LoadConfig").Return(testConfig, nil)
	// Simulate keyring.ErrNotFound
	// Use correct keyring user constant
	mockKeyring.On("GetAPIKey", keyringService, keyringUser).Return("", config.ErrAPIKeyNotFound) // Simulate specific error

	// cmd := &cobra.Command{} // Not needed

	// Call configShowRunE directly
	err := configShowRunE(mockProvider, mockKeyring, &out)

	// Assertions
	assert.NoError(t, err) // Should not return an error, just indicate not found
	assert.Contains(t, out.String(), "Current Ticketron Configuration:")
	assert.Contains(t, out.String(), "  MCP Server URL: http://mcp.example.com")
	assert.Contains(t, out.String(), "  LLM Provider:   openai")                                    // Check provider
	assert.Contains(t, out.String(), "    OpenAI Model: gpt-test")                                  // Check model name
	assert.Contains(t, out.String(), "  LLM API Key:    Not Set (use 'tix config set-key' to set)") // Exact format and status
	mockProvider.AssertExpectations(t)
	mockKeyring.AssertExpectations(t)
}

func TestConfigShowCmd_KeyringGetError_Other(t *testing.T) {
	mockProvider := new(MockConfigProvider)
	mockKeyring := new(MockKeyringClient)
	var out bytes.Buffer

	// Setup mock expectations
	testConfig := &config.AppConfig{
		MCPServerURL: "http://mcp.example.com",
		LLM:          config.LLMConfig{Provider: "openai", OpenAI: config.OpenAIConfig{ModelName: "gpt-test"}}, // Updated config structure
	}
	mockProvider.On("LoadConfig").Return(testConfig, nil)
	// Simulate a different keyring error
	expectedErr := errors.New("keyring daemon unavailable")
	// Use correct keyring user constant
	mockKeyring.On("GetAPIKey", keyringService, keyringUser).Return("", expectedErr)

	// cmd := &cobra.Command{} // Not needed

	// Call configShowRunE directly
	err := configShowRunE(mockProvider, mockKeyring, &out)

	// Assertions
	assert.NoError(t, err) // Should not return an error, just indicate the error status
	assert.Contains(t, out.String(), "Current Ticketron Configuration:")
	assert.Contains(t, out.String(), "  MCP Server URL: http://mcp.example.com")
	assert.Contains(t, out.String(), "  LLM Provider:   openai")                                                                   // Check provider
	assert.Contains(t, out.String(), "    OpenAI Model: gpt-test")                                                                 // Check model name
	assert.Contains(t, out.String(), "  LLM API Key:    Status Unknown (error checking keychain/env: keyring daemon unavailable)") // Exact format and status
	mockProvider.AssertExpectations(t)
	mockKeyring.AssertExpectations(t)
}

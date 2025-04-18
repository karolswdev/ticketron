//go:build integration

package integration

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog" // Added for log level setting

	"github.com/karolswdev/ticketron/cmd" // Import the cmd package
	"github.com/karolswdev/ticketron/internal/config"
)

// mockLLMServer creates a mock HTTP server simulating the LLM API.
// It takes a handler function to define the mock response.
func mockLLMServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

// mockMCPServer creates a mock HTTP server simulating the jira-mcp-server API.
// It takes a handler function to define the mock response.
func mockMCPServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

// setupTestEnvironment creates a temporary directory for configuration files,
// writes a basic config.yaml pointing to mock servers, and returns the path
// to the temporary directory and a cleanup function.
func setupTestEnvironment(t *testing.T, llmURL, mcpURL string) (string, func()) {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "ticketron-integration-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create a basic config file pointing to the mock servers
	// Ensure precise YAML indentation
	configContent := fmt.Sprintf(`
mcp_server_url: %s
llm:
  provider: "openai"
  openai:
    model_name: "test-model"
    base_url: "%s"
`, mcpURL, llmURL) // Provide both URLs, llmURL might be empty

	configPath := filepath.Join(tempDir, config.DefaultConfigFileName)
	err = os.WriteFile(configPath, []byte(configContent), 0600)
	if err != nil {
		os.RemoveAll(tempDir) // Clean up if write fails
		t.Fatalf("Failed to write temp config file: %v", err)
	}

	// Set environment variables for API keys (can be dummy values for tests)
	// These are not directly used by config loading but might be by underlying libs if not mocked
	// os.Setenv("TICKETRON_TEST_OPENAI_API_KEY", "test-openai-key")
	// os.Setenv("TICKETRON_TEST_MCP_API_KEY", "test-mcp-key")

	// Set the TICKETRON_CONFIG_DIR environment variable to point to the temp dir
	originalConfigDir := os.Getenv(config.ConfigDirEnvVar) // Use exported constant
	os.Setenv(config.ConfigDirEnvVar, tempDir)             // Use exported constant

	cleanup := func() {
		os.RemoveAll(tempDir)
		// os.Unsetenv("TICKETRON_TEST_OPENAI_API_KEY")
		// os.Unsetenv("TICKETRON_TEST_MCP_API_KEY")
		// Restore original config dir env var if it was set
		if originalConfigDir != "" {
			os.Setenv(config.ConfigDirEnvVar, originalConfigDir) // Use exported constant
		} else {
			os.Unsetenv(config.ConfigDirEnvVar) // Use exported constant
		}
	}

	return tempDir, cleanup
}

// executeTixCommand runs the ticketron root command with given arguments in-process.
// It captures stdout and stderr. The TICKETRON_CONFIG_DIR environment variable
// must be set correctly before calling this function (e.g., by setupTestEnvironment).
func executeTixCommand(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	// Set global log level to Debug for integration tests
	originalLevel := zerolog.GlobalLevel()
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	// Restore original level after test execution
	t.Cleanup(func() { zerolog.SetGlobalLevel(originalLevel) })

	// Capture stdout and stderr
	var outBuf, errBuf bytes.Buffer

	// Get the root command
	rootCmd := cmd.NewRootCmd() // Assuming NewRootCmd() initializes the command tree

	// Set output streams
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(&errBuf)

	// Set command arguments (os.Args[0] is the command name, followed by actual args)
	rootCmd.SetArgs(args)

	// Execute the command
	// Use context.Background() for simple execution
	ctx := context.Background()
	execErr := rootCmd.ExecuteContext(ctx)

	// Return captured output and error
	return outBuf.String(), errBuf.String(), execErr
}

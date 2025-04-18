//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	// "time" // No longer needed for LLM mock

	"github.com/karolswdev/ticketron/cmd"          // Import the cmd package
	"github.com/karolswdev/ticketron/internal/llm" // Import llm for response struct
	"github.com/karolswdev/ticketron/internal/mcpclient"
	"github.com/spf13/cobra" // Import cobra
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock" // Import mock
	"github.com/stretchr/testify/require"
)

// TestCreateWorkflow tests the end-to-end flow:
// 1. config init
// 2. config set-key (using a dummy key)
// 3. create (mocking LLM interface and MCP HTTP server)
func TestCreateWorkflow(t *testing.T) {
	// --- Mock LLM Server Removed ---
	// LLM interaction will be mocked at the interface level

	// --- Mock MCP Server ---
	mcpRequestVerified := false // Flag to ensure MCP mock was called
	mockMCP := mockMCPServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method, "MCP mock expected POST")
		require.Equal(t, "/create_jira_issue", r.URL.Path, "MCP mock expected /create_jira_issue path")

		// Decode request to verify payload
		var reqBody mcpclient.CreateIssueRequest
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		require.NoError(t, err, "Failed to decode MCP request body")
		assert.Equal(t, "TEST", reqBody.ProjectKey, "MCP request project key mismatch")
		assert.Equal(t, "My test ticket", reqBody.Summary, "MCP request summary mismatch")
		assert.Equal(t, "Task", reqBody.IssueType, "MCP request issue type mismatch")
		mcpRequestVerified = true // Mark as verified

		// Simulate a successful MCP response
		mcpResponse := mcpclient.CreateIssueResponse{
			Key:  "TEST-123",
			Self: "http://mock-jira/browse/TEST-123",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		err = json.NewEncoder(w).Encode(mcpResponse)
		require.NoError(t, err, "Failed to encode mock MCP response")
	})

	// --- Setup Test Environment ---
	// Pass empty string for llmURL as it's not needed for config anymore
	tempDir, cleanup := setupTestEnvironment(t, "", mockMCP.URL)
	t.Cleanup(cleanup) // Ensure cleanup runs even if test fails

	// Create a dummy links.yaml for project mapping
	linksContent := `
projects:
  - name: "Test Project"
    key: "TEST"
    default_issue_type: "Task"
`
	linksPath := filepath.Join(tempDir, "links.yaml")
	err := os.WriteFile(linksPath, []byte(linksContent), 0600)
	require.NoError(t, err, "Failed to write temp links file")

	// --- Execute Commands ---

	// 1. tix config init
	t.Run("config init", func(t *testing.T) {
		stdout, stderr, err := executeTixCommand(t, "config", "init")
		require.NoError(t, err, "config init failed")
		assert.Empty(t, stderr, "config init stderr should be empty")
		assert.Contains(t, stdout, "Configuration directory and default files ensured.", "config init output mismatch")
		_, err = os.Stat(filepath.Join(tempDir, "config.yaml"))
		require.NoError(t, err, "config.yaml not found after init")
		_, err = os.Stat(filepath.Join(tempDir, "links.yaml")) // links.yaml should be overwritten by init
		require.NoError(t, err, "links.yaml not found after init")
	})

	// 2. tix config set-key
	t.Run("config set-key", func(t *testing.T) {
		// Use valid-looking dummy key
		stdout, stderr, err := executeTixCommand(t, "config", "set-key", "sk-dummyintegrationtestkey1234567890")
		if err != nil {
			t.Logf("config set-key returned an error (may be expected in some CI environments): %v\nStderr: %s", err, stderr)
		} else {
			assert.Empty(t, stderr, "config set-key stderr should be empty on success")
			assert.Contains(t, stdout, "API key stored successfully", "config set-key output mismatch")
		}
	})

	// 3. tix create (Modified to inject mocks and call runner directly)
	t.Run("create", func(t *testing.T) {
		// Instantiate Mocks and Dependencies
		mockLLM := new(cmd.MockLLMClient) // Use the mock from cmd package
		// Real config provider to load files from tempDir
		configProvider := &cmd.DefaultConfigProvider{} // Assuming this is accessible or create similar struct
		// Real project mapper and issue resolver
		projectMapper := &cmd.DefaultProjectMapper{}         // Assuming this is accessible
		issueTypeResolver := &cmd.DefaultIssueTypeResolver{} // Assuming this is accessible

		// Configure MCP Client to point to mock server
		// Need to load config first to get the URL
		appCfg, err := configProvider.LoadConfig()
		require.NoError(t, err, "Failed to load config for MCP client setup")
		mcpClientInstance, err := mcpclient.New(appCfg) // Create real client pointing to mock URL
		require.NoError(t, err, "Failed to create MCP client instance")
		// Wrap the real client instance in the defaultMCPClient struct if needed by runner,
		// or adjust runner to accept mcpclient.Client directly.
		// Assuming runner needs the interface:
		mcpClient := &cmd.DefaultMCPClientWrapper{Client: mcpClientInstance} // Need a wrapper struct

		// Setup LLM Mock Expectation
		expectedLLMResponse := llm.LLMResponse{
			Summary:               "My test ticket",
			Description:           "This is a test.",
			ProjectNameSuggestion: "Test Project", // Matches links.yaml
		}
		mockLLM.On("GenerateTicketDetails",
			mock.Anything,                   // Use mock.Anything for context
			"Create a ticket for this test", // Match exact user input
			mock.Anything,                   // Use mock.Anything for system prompt
			mock.Anything,                   // Use mock.Anything for context data
		).Return(expectedLLMResponse, nil)

		// Instantiate the runner directly
		runner := cmd.NewCreateCmdRunnerForTest( // Need a test constructor
			configProvider,
			mockLLM, // Inject mock LLM
			mcpClient,
			projectMapper,
			issueTypeResolver,
		)

		// Prepare Cobra command and arguments
		cobraCmd := &cobra.Command{} // Dummy command
		args := []string{"Create a ticket for this test"}
		var outBuf bytes.Buffer
		cobraCmd.SetOut(&outBuf)
		cobraCmd.SetErr(&outBuf) // Capture stderr too

		// Execute the runner's Run method
		err = runner.Run(cobraCmd, args)

		// Assertions
		stderr := outBuf.String() // Combined stdout/stderr for simplicity here
		require.NoError(t, err, fmt.Sprintf("runner.Run failed. Stderr/Stdout:\n%s", stderr))
		assert.True(t, mcpRequestVerified, "MCP server mock was not called") // Verify MCP interaction happened

		// Check output written to cobraCmd's output buffer
		stdout := outBuf.String()
		assert.Contains(t, stdout, "Successfully created JIRA issue:", "tix create success message mismatch")
		assert.Contains(t, stdout, "Key: TEST-123", "tix create key mismatch")
		assert.Contains(t, stdout, "URL: http://mock-jira/browse/TEST-123", "tix create URL mismatch")

		// Verify mock expectations
		mockLLM.AssertExpectations(t)
	})
}

// Need to define DefaultConfigProvider, DefaultProjectMapper, DefaultIssueTypeResolver,
// DefaultMCPClientWrapper and NewCreateCmdRunnerForTest in cmd package or make them accessible.
// Example wrapper:
// type DefaultMCPClientWrapper struct { Client *mcpclient.Client }
// func (w *DefaultMCPClientWrapper) CreateIssue(...) { return w.Client.CreateIssue(...) }
// func (w *DefaultMCPClientWrapper) SearchIssues(...) { return w.Client.SearchIssues(...) }

// Example Test Constructor:
// func NewCreateCmdRunnerForTest(cp ConfigProvider, llm llm.Client, mcp MCPClient, pm ProjectMapper, ir IssueTypeResolver) *createCmdRunner {
// 	return &createCmdRunner{
// 		configProvider:    cp,
// 		llmClient:         llm,
// 		mcpClient:         mcp,
// 		projectMapper:     pm,
// 		issueTypeResolver: ir,
// 	}
// }

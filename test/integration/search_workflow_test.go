//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings" // Added for TSV tests
	"testing"

	"github.com/karolswdev/ticketron/cmd"                // Import cmd package to access flags
	"github.com/karolswdev/ticketron/internal/mcpclient" // For response struct
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3" // Added for YAML tests
)

// mockSearchResponse is the standard response used by the mock MCP server for search tests.
var mockSearchResponse = mcpclient.SearchIssuesResponse{
	StartAt:    0,
	MaxResults: 20,
	Total:      2,
	Issues: []mcpclient.Issue{
		{
			Key:  "PROJ-456",
			ID:   "10001",
			Self: "http://mock-jira/browse/PROJ-456",
			Fields: mcpclient.IssueFields{
				Summary:     "Fix urgent bug in login",
				Status:      mcpclient.Status{Name: "Open"},
				IssueType:   mcpclient.IssueType{Name: "Bug"},
				Description: "Detailed description for PROJ-456.",
			},
		},
		{
			Key:  "PROJ-457",
			ID:   "10002",
			Self: "http://mock-jira/browse/PROJ-457",
			Fields: mcpclient.IssueFields{
				Summary:     "Another urgent bug report",
				Status:      mcpclient.Status{Name: "Open"},
				IssueType:   mcpclient.IssueType{Name: "Task"},
				Description: "Second issue\nwith newline.",
			},
		},
	},
}

// commonSearchTestSetup sets up the mock MCP server and test environment for search tests.
func commonSearchTestSetup(t *testing.T) (cleanup func()) {
	t.Helper()
	// --- Mock MCP Server ---
	mockMCP := mockMCPServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method, "MCP mock expected POST")
		require.Equal(t, "/search_jira_issues", r.URL.Path, "MCP mock expected /search_jira_issues path")

		var reqBody mcpclient.SearchIssuesRequest
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		require.NoError(t, err, "Failed to decode MCP search request body")

		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(mockSearchResponse)
		require.NoError(t, err, "Failed to encode mock MCP search response")
	})

	// --- Setup Test Environment ---
	_, cleanupEnv := setupTestEnvironment(t, "", mockMCP.URL) // No LLM needed

	// --- Initial Config Init ---
	// Run config init once for the setup
	_, _, err := executeTixCommand(t, "config", "init")
	require.NoError(t, err, "config init failed during setup")

	return cleanupEnv // Return the cleanup function from setupTestEnvironment
}

// resetSearchFlags resets flags on a temporary command instance.
// This is crucial to prevent state leakage between tests.
func resetSearchFlags(t *testing.T) {
	t.Helper()
	// Create a temporary root command to access the search subcommand's flags
	tempRootCmd := cmd.NewRootCmd()
	searchCmd, _, err := tempRootCmd.Find([]string{"search"})
	require.NoError(t, err, "Failed to find search command for flag reset")
	// Reset the specific flag causing issues
	searchCmd.Flags().Set("output-fields", "")
	// Reset other potentially problematic flags if needed
	searchCmd.Flags().Set("output", "")
}

// TestSearchWorkflow_DefaultText tests the default text output.
func TestSearchWorkflow_DefaultText(t *testing.T) {
	cleanup := commonSearchTestSetup(t)
	t.Cleanup(cleanup)
	resetSearchFlags(t) // Reset flags before test
	searchJQL := "status = Open"

	stdout, stderr, err := executeTixCommand(t, "search", searchJQL)
	require.NoError(t, err, fmt.Sprintf("tix search failed. Stderr:\n%s", stderr))
	assert.Empty(t, stderr, "tix search stderr should be empty")

	assert.Contains(t, stdout, "Found 2 issues", "Search output missing total count")
	assert.Contains(t, stdout, "PROJ-456", "Search output missing issue key PROJ-456")
	assert.Contains(t, stdout, "Fix urgent bug in login", "Search output missing summary for PROJ-456")
	assert.Contains(t, stdout, "PROJ-457", "Search output missing issue key PROJ-457")
	assert.Contains(t, stdout, "Another urgent bug report", "Search output missing summary for PROJ-457")
}

// TestSearchWorkflow_JSONFull tests the full JSON output.
func TestSearchWorkflow_JSONFull(t *testing.T) {
	cleanup := commonSearchTestSetup(t)
	t.Cleanup(cleanup)
	resetSearchFlags(t) // Reset flags before test
	searchJQL := "status = Open"

	stdout, stderr, err := executeTixCommand(t, "search", "--output", "json", searchJQL)
	require.NoError(t, err, fmt.Sprintf("tix search --output json failed. Stderr:\n%s", stderr))
	assert.Empty(t, stderr, "tix search --output json stderr should be empty")

	var searchResult mcpclient.SearchIssuesResponse
	err = json.Unmarshal([]byte(stdout), &searchResult)
	require.NoError(t, err, "Failed to unmarshal JSON output from tix search")
	assert.Equal(t, 2, searchResult.Total, "JSON output total mismatch")
	require.Len(t, searchResult.Issues, 2, "JSON output issues count mismatch")
	assert.Equal(t, "PROJ-456", searchResult.Issues[0].Key, "JSON output issue 0 key mismatch")
	assert.Equal(t, "PROJ-457", searchResult.Issues[1].Key, "JSON output issue 1 key mismatch")
}

// TestSearchWorkflow_JSONFields tests filtered JSON output.
func TestSearchWorkflow_JSONFields(t *testing.T) {
	cleanup := commonSearchTestSetup(t)
	t.Cleanup(cleanup)
	resetSearchFlags(t) // Reset flags before test
	searchJQL := "status = Open"
	fields := "key,fields.status.name,fields.summary"

	stdout, stderr, err := executeTixCommand(t, "search", "-o", "json", "-f", fields, searchJQL)
	require.NoError(t, err, fmt.Sprintf("tix search -o json -f failed. Stderr:\n%s", stderr))
	assert.Empty(t, stderr, "tix search -o json -f stderr should be empty")

	var result []map[string]interface{}
	err = json.Unmarshal([]byte(stdout), &result)
	require.NoError(t, err, "Failed to unmarshal filtered JSON output")
	require.Len(t, result, 2, "Filtered JSON output issues count mismatch")

	// Check first issue
	assert.Equal(t, "PROJ-456", result[0]["key"])
	assert.Equal(t, "Open", result[0]["fields.status.name"])
	assert.Equal(t, "Fix urgent bug in login", result[0]["fields.summary"])
	_, idExists := result[0]["id"]
	assert.False(t, idExists, "ID field should not exist in filtered JSON")

	// Check second issue
	assert.Equal(t, "PROJ-457", result[1]["key"])
	assert.Equal(t, "Open", result[1]["fields.status.name"])
	assert.Equal(t, "Another urgent bug report", result[1]["fields.summary"])
}

// TestSearchWorkflow_YAMLFull tests the default YAML output (list of issues).
func TestSearchWorkflow_YAMLFull(t *testing.T) {
	cleanup := commonSearchTestSetup(t)
	t.Cleanup(cleanup)
	resetSearchFlags(t) // Reset flags before test
	searchJQL := "status = Open"

	stdout, stderr, err := executeTixCommand(t, "search", "-o", "yaml", searchJQL)
	require.NoError(t, err, fmt.Sprintf("tix search -o yaml failed. Stderr:\n%s", stderr))
	assert.Empty(t, stderr, "tix search -o yaml stderr should be empty")

	// Expect a list of issues. Unmarshal into the actual Issue type.
	var result []mcpclient.Issue
	err = yaml.Unmarshal([]byte(stdout), &result)
	require.NoError(t, err, "Failed to unmarshal full YAML output into []mcpclient.Issue")
	require.Len(t, result, 2, "Full YAML output issues count mismatch")

	// Check content using struct fields
	assert.Equal(t, "PROJ-456", result[0].Key, "Full YAML output issue 0 key mismatch")
	assert.Equal(t, "Fix urgent bug in login", result[0].Fields.Summary, "Full YAML output issue 0 summary mismatch")
	assert.Equal(t, "Open", result[0].Fields.Status.Name, "Full YAML output issue 0 status mismatch")
	assert.Equal(t, "PROJ-457", result[1].Key, "Full YAML output issue 1 key mismatch")
	assert.Equal(t, "Another urgent bug report", result[1].Fields.Summary, "Full YAML output issue 1 summary mismatch")
}

// TestSearchWorkflow_YAMLFields tests filtered YAML output.
func TestSearchWorkflow_YAMLFields(t *testing.T) {
	cleanup := commonSearchTestSetup(t)
	t.Cleanup(cleanup)
	resetSearchFlags(t) // Reset flags before test
	searchJQL := "status = Open"
	fields := "key,fields.issuetype.name"

	stdout, stderr, err := executeTixCommand(t, "search", "-o", "yaml", "-f", fields, searchJQL)
	require.NoError(t, err, fmt.Sprintf("tix search -o yaml -f failed. Stderr:\n%s", stderr))
	assert.Empty(t, stderr, "tix search -o yaml -f stderr should be empty")

	var result []map[string]interface{}
	err = yaml.Unmarshal([]byte(stdout), &result)
	require.NoError(t, err, "Failed to unmarshal filtered YAML output")
	require.Len(t, result, 2)
	assert.Equal(t, "PROJ-456", result[0]["key"])
	assert.Equal(t, "Bug", result[0]["fields.issuetype.name"])
	assert.Equal(t, "PROJ-457", result[1]["key"])
	assert.Equal(t, "Task", result[1]["fields.issuetype.name"])
	_, statusExists := result[0]["fields.status.name"]
	assert.False(t, statusExists, "Status field should not exist in filtered YAML")
}

// TestSearchWorkflow_TSVDefault tests the default TSV output.
func TestSearchWorkflow_TSVDefault(t *testing.T) {
	cleanup := commonSearchTestSetup(t)
	t.Cleanup(cleanup)
	resetSearchFlags(t) // Reset flags before test
	searchJQL := "status = Open"

	stdout, stderr, err := executeTixCommand(t, "search", "-o", "tsv", searchJQL)
	require.NoError(t, err, fmt.Sprintf("tix search -o tsv failed. Stderr:\n%s", stderr))
	assert.Empty(t, stderr, "tix search -o tsv stderr should be empty")

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	require.Len(t, lines, 3, "TSV default output should have 3 lines (header + 2 data)")

	// Check header contains expected fields
	assert.Contains(t, lines[0], "key", "TSV header missing key")
	assert.Contains(t, lines[0], "fields.summary", "TSV header missing summary")
	assert.Contains(t, lines[0], "fields.status.name", "TSV header missing status")
	assert.Contains(t, lines[0], "fields.issuetype.name", "TSV header missing issuetype")

	// Check line 1 contains expected values
	assert.Contains(t, lines[1], "PROJ-456", "TSV line 1 missing key")
	assert.Contains(t, lines[1], "Fix urgent bug in login", "TSV line 1 missing summary")
	assert.Contains(t, lines[1], "Open", "TSV line 1 missing status")
	assert.Contains(t, lines[1], "Bug", "TSV line 1 missing issuetype")

	// Check line 2 contains expected values
	assert.Contains(t, lines[2], "PROJ-457", "TSV line 2 missing key")
	assert.Contains(t, lines[2], "Another urgent bug report", "TSV line 2 missing summary")
	assert.Contains(t, lines[2], "Open", "TSV line 2 missing status") // Status is Open for both in mock data
	assert.Contains(t, lines[2], "Task", "TSV line 2 missing issuetype")
}

// TestSearchWorkflow_TSVFields tests filtered TSV output.
func TestSearchWorkflow_TSVFields(t *testing.T) {
	cleanup := commonSearchTestSetup(t)
	t.Cleanup(cleanup)
	resetSearchFlags(t) // Reset flags before test
	searchJQL := "status = Open"
	fields := "key,fields.description"

	stdout, stderr, err := executeTixCommand(t, "search", "-o", "tsv", "-f", fields, searchJQL)
	require.NoError(t, err, fmt.Sprintf("tix search -o tsv -f failed. Stderr:\n%s", stderr))
	assert.Empty(t, stderr, "tix search -o tsv -f stderr should be empty")

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	require.Len(t, lines, 3, "TSV fields output should have 3 lines")
	assert.Equal(t, "key\tfields.description", lines[0])
	assert.Equal(t, "PROJ-456\tDetailed description for PROJ-456.", lines[1])
	// Check newline sanitization
	assert.Equal(t, "PROJ-457\tSecond issue with newline.", lines[2])
}

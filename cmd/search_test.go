package cmd

import (
	"bytes"
	"encoding/json" // Added for JSON tests
	"errors"
	"strings" // Added for TSV tests
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gopkg.in/yaml.v3" // Added for YAML tests

	"github.com/karolswdev/ticketron/internal/mcpclient"
)

// --- Mocks (Defined in mocks_test.go) ---
// MockConfigProvider
// MockMCPClient
// MockKeyringClient

// Helper to create a mock MCP response
func createMockSearchResponse() *mcpclient.SearchIssuesResponse {
	return &mcpclient.SearchIssuesResponse{
		StartAt:    0,
		MaxResults: 20,
		Total:      2,
		Issues: []mcpclient.Issue{
			{
				Key:  "TEST-1",
				ID:   "10001",
				Self: "http://jira.example.com/rest/api/2/issue/10001",
				Fields: mcpclient.IssueFields{
					Summary:     "Found issue 1 with details",
					Status:      mcpclient.Status{Name: "Open"},
					IssueType:   mcpclient.IssueType{Name: "Bug"},
					Description: "This is the first test issue.",
				},
			},
			{
				Key:  "TEST-2",
				ID:   "10002",
				Self: "http://jira.example.com/rest/api/2/issue/10002",
				Fields: mcpclient.IssueFields{
					Summary:     "Found issue 2",
					Status:      mcpclient.Status{Name: "In Progress"},
					IssueType:   mcpclient.IssueType{Name: "Task"},
					Description: "Second issue\nwith newline.", // Add newline for TSV test
				},
			},
		},
	}
}

// Helper to setup command flags for tests
func setupSearchCmdFlags(cmd *cobra.Command, outputFormat, outputFields string) {
	cmd.Flags().String("jql", "", "JQL query string")
	cmd.Flags().Int("max-results", 20, "Maximum number of results to return")
	cmd.Flags().StringP("output", "o", "", "Output format (text, json, yaml, tsv)")
	cmd.Flags().StringP("output-fields", "f", "", "Comma-separated fields for structured output")

	// Set flag values for the test
	if outputFormat != "" {
		cmd.Flags().Set("output", outputFormat)
	}
	if outputFields != "" {
		cmd.Flags().Set("output-fields", outputFields)
	}
}

func TestSearchCmd_Success_Text(t *testing.T) {
	mockProvider := new(MockConfigProvider)
	mockMCP := new(MockMCPClient)
	var out bytes.Buffer

	mockResponse := createMockSearchResponse()
	mockMCP.On("SearchIssues", mock.Anything, mock.AnythingOfType("mcpclient.SearchIssuesRequest")).Return(mockResponse, nil)

	cmd := &cobra.Command{}
	setupSearchCmdFlags(cmd, "text", "") // Explicitly text
	args := []string{"test query"}

	err := searchRunE(mockProvider, mockMCP, &out, cmd, args)

	assert.NoError(t, err)
	output := out.String()
	assert.Contains(t, output, "Found 2 issues:")
	assert.Contains(t, output, "- TEST-1 - Open - Found issue 1 with details")
	assert.Contains(t, output, "- TEST-2 - In Progress - Found issue 2")
	mockProvider.AssertExpectations(t)
	mockMCP.AssertExpectations(t)
	mockProvider.AssertNotCalled(t, "LoadConfig")
}

func TestSearchCmd_Success_JSON_Full(t *testing.T) {
	mockProvider := new(MockConfigProvider)
	mockMCP := new(MockMCPClient)
	var out bytes.Buffer

	mockResponse := createMockSearchResponse()
	mockMCP.On("SearchIssues", mock.Anything, mock.AnythingOfType("mcpclient.SearchIssuesRequest")).Return(mockResponse, nil)

	cmd := &cobra.Command{}
	setupSearchCmdFlags(cmd, "json", "") // JSON, no fields specified
	args := []string{"test query"}

	err := searchRunE(mockProvider, mockMCP, &out, cmd, args)

	assert.NoError(t, err)
	var result mcpclient.SearchIssuesResponse
	err = json.Unmarshal(out.Bytes(), &result)
	assert.NoError(t, err, "Output should be valid JSON parsable into SearchIssuesResponse")
	assert.Equal(t, 2, len(result.Issues))
	assert.Equal(t, "TEST-1", result.Issues[0].Key)
	assert.Equal(t, "Found issue 1 with details", result.Issues[0].Fields.Summary)
	assert.Equal(t, "Open", result.Issues[0].Fields.Status.Name)
	mockProvider.AssertExpectations(t)
	mockMCP.AssertExpectations(t)
}

func TestSearchCmd_Success_JSON_Fields(t *testing.T) {
	mockProvider := new(MockConfigProvider)
	mockMCP := new(MockMCPClient)
	var out bytes.Buffer

	mockResponse := createMockSearchResponse()
	mockMCP.On("SearchIssues", mock.Anything, mock.AnythingOfType("mcpclient.SearchIssuesRequest")).Return(mockResponse, nil)

	cmd := &cobra.Command{}
	// Test with specific fields, including nested and non-existent
	setupSearchCmdFlags(cmd, "json", "key, fields.summary, fields.status.name, nonExistentField")
	args := []string{"test query"}

	err := searchRunE(mockProvider, mockMCP, &out, cmd, args)

	assert.NoError(t, err)
	var result []map[string]interface{} // Expecting a slice of maps
	err = json.Unmarshal(out.Bytes(), &result)
	assert.NoError(t, err, "Output should be valid JSON parsable into []map[string]interface{}")
	assert.Equal(t, 2, len(result))

	// Check first issue
	assert.Equal(t, "TEST-1", result[0]["key"])
	assert.Equal(t, "Found issue 1 with details", result[0]["fields.summary"])
	assert.Equal(t, "Open", result[0]["fields.status.name"])
	assert.Nil(t, result[0]["nonExistentField"], "Non-existent fields should be null")

	// Check second issue
	assert.Equal(t, "TEST-2", result[1]["key"])
	assert.Equal(t, "Found issue 2", result[1]["fields.summary"])
	assert.Equal(t, "In Progress", result[1]["fields.status.name"])
	assert.Nil(t, result[1]["nonExistentField"])

	mockProvider.AssertExpectations(t)
	mockMCP.AssertExpectations(t)
}

func TestSearchCmd_Success_YAML_Full(t *testing.T) {
	mockProvider := new(MockConfigProvider)
	mockMCP := new(MockMCPClient)
	var out bytes.Buffer

	mockResponse := createMockSearchResponse()
	mockMCP.On("SearchIssues", mock.Anything, mock.AnythingOfType("mcpclient.SearchIssuesRequest")).Return(mockResponse, nil)

	cmd := &cobra.Command{}
	setupSearchCmdFlags(cmd, "yaml", "") // YAML, no fields
	args := []string{"test query"}

	err := searchRunE(mockProvider, mockMCP, &out, cmd, args)

	assert.NoError(t, err)
	// Expect a slice of issues for default YAML output
	var result []mcpclient.Issue
	err = yaml.Unmarshal(out.Bytes(), &result)
	assert.NoError(t, err, "Output should be valid YAML parsable into []mcpclient.Issue")
	assert.Equal(t, 2, len(result), "Default YAML should contain 2 issues")
	assert.Equal(t, "TEST-1", result[0].Key)
	assert.Equal(t, "Found issue 1 with details", result[0].Fields.Summary) // Check nested fields directly
	assert.Equal(t, "Open", result[0].Fields.Status.Name)
	assert.Equal(t, "TEST-2", result[1].Key)
	assert.Equal(t, "Found issue 2", result[1].Fields.Summary)
	assert.Equal(t, "In Progress", result[1].Fields.Status.Name)

	mockProvider.AssertExpectations(t)
	mockMCP.AssertExpectations(t)
}

func TestSearchCmd_Success_YAML_Fields(t *testing.T) {
	mockProvider := new(MockConfigProvider)
	mockMCP := new(MockMCPClient)
	var out bytes.Buffer

	mockResponse := createMockSearchResponse()
	mockMCP.On("SearchIssues", mock.Anything, mock.AnythingOfType("mcpclient.SearchIssuesRequest")).Return(mockResponse, nil)

	cmd := &cobra.Command{}
	setupSearchCmdFlags(cmd, "yaml", "key, fields.issuetype.name") // YAML with fields
	args := []string{"test query"}

	err := searchRunE(mockProvider, mockMCP, &out, cmd, args)

	assert.NoError(t, err)
	var result []map[string]interface{} // Expecting slice of maps
	err = yaml.Unmarshal(out.Bytes(), &result)
	assert.NoError(t, err, "Output should be valid YAML parsable into []map[string]interface{}")
	assert.Equal(t, 2, len(result))
	assert.Equal(t, "TEST-1", result[0]["key"])
	assert.Equal(t, "Bug", result[0]["fields.issuetype.name"])
	assert.Equal(t, "TEST-2", result[1]["key"])
	assert.Equal(t, "Task", result[1]["fields.issuetype.name"])
	_, summaryExists := result[0]["fields.summary"]
	assert.False(t, summaryExists, "Summary field should not exist in filtered YAML output")

	mockProvider.AssertExpectations(t)
	mockMCP.AssertExpectations(t)
}

func TestSearchCmd_Success_TSV_Default(t *testing.T) {
	mockProvider := new(MockConfigProvider)
	mockMCP := new(MockMCPClient)
	var out bytes.Buffer

	mockResponse := createMockSearchResponse()
	mockMCP.On("SearchIssues", mock.Anything, mock.AnythingOfType("mcpclient.SearchIssuesRequest")).Return(mockResponse, nil)

	cmd := &cobra.Command{}
	setupSearchCmdFlags(cmd, "tsv", "") // TSV, default fields
	args := []string{"test query"}

	err := searchRunE(mockProvider, mockMCP, &out, cmd, args)

	assert.NoError(t, err)
	output := out.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.Equal(t, 3, len(lines), "Should have header + 2 data lines")
	// Reverted expected default fields to lowercase JSON style
	expectedHeader := "key\tfields.summary\tfields.status.name\tfields.issuetype.name"
	assert.Equal(t, expectedHeader, lines[0])
	assert.Equal(t, "TEST-1\tFound issue 1 with details\tOpen\tBug", lines[1])
	// Check newline sanitization
	assert.Equal(t, "TEST-2\tFound issue 2\tIn Progress\tTask", lines[2])

	mockProvider.AssertExpectations(t)
	mockMCP.AssertExpectations(t)
}

func TestSearchCmd_Success_TSV_Fields(t *testing.T) {
	mockProvider := new(MockConfigProvider)
	mockMCP := new(MockMCPClient)
	var out bytes.Buffer

	mockResponse := createMockSearchResponse()
	mockMCP.On("SearchIssues", mock.Anything, mock.AnythingOfType("mcpclient.SearchIssuesRequest")).Return(mockResponse, nil)

	cmd := &cobra.Command{}
	setupSearchCmdFlags(cmd, "tsv", "key, fields.description, fields.status.name") // TSV, specific fields
	args := []string{"test query"}

	err := searchRunE(mockProvider, mockMCP, &out, cmd, args)

	assert.NoError(t, err)
	output := out.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.Equal(t, 3, len(lines), "Should have header + 2 data lines")
	assert.Equal(t, "key\tfields.description\tfields.status.name", lines[0])
	assert.Equal(t, "TEST-1\tThis is the first test issue.\tOpen", lines[1])
	// Check newline sanitization in description
	assert.Equal(t, "TEST-2\tSecond issue with newline.\tIn Progress", lines[2])

	mockProvider.AssertExpectations(t)
	mockMCP.AssertExpectations(t)
}

func TestSearchCmd_ConfigError(t *testing.T) {
	// This test remains skipped as config loading happens in the RunE wrapper, not searchRunE.
	t.Skip("Skipping TestSearchCmd_ConfigError as config loading happens before searchRunE")
}

func TestSearchCmd_MCPError(t *testing.T) {
	mockProvider := new(MockConfigProvider)
	mockMCP := new(MockMCPClient)
	var out bytes.Buffer

	mcpErr := errors.New("mcp communication error")
	mockMCP.On("SearchIssues", mock.Anything, mock.Anything).Return((*mcpclient.SearchIssuesResponse)(nil), mcpErr)

	cmd := &cobra.Command{}
	setupSearchCmdFlags(cmd, "", "") // Default flags
	args := []string{"test query"}

	err := searchRunE(mockProvider, mockMCP, &out, cmd, args)

	assert.Error(t, err)
	assert.ErrorIs(t, err, mcpErr)
	assert.Empty(t, out.String()) // No output on MCP error
	mockProvider.AssertNotCalled(t, "LoadConfig")
	mockMCP.AssertExpectations(t)
}

func TestSearchCmd_NoArgs(t *testing.T) {
	mockProvider := new(MockConfigProvider)
	mockMCP := new(MockMCPClient)
	var out bytes.Buffer

	cmd := &cobra.Command{}
	setupSearchCmdFlags(cmd, "", "") // Default flags
	args := []string{}               // No arguments

	err := searchRunE(mockProvider, mockMCP, &out, cmd, args)

	assert.Error(t, err)
	assert.EqualError(t, err, "no JQL query provided")
	assert.Empty(t, out.String())
	mockProvider.AssertNotCalled(t, "LoadConfig")
	mockMCP.AssertNotCalled(t, "SearchIssues", mock.Anything, mock.Anything)
}

// Test for JSON marshalling error (less likely but good practice)
func TestSearchCmd_JSONMarshalError(t *testing.T) {
	t.Skip("Skipping JSON marshal error test due to complexity of inducing the error")
}

// Test for YAML marshalling error
func TestSearchCmd_YAMLMarshalError(t *testing.T) {
	t.Skip("Skipping YAML marshal error test due to complexity of inducing the error")
}

// Test getValueByPath helper function directly
func TestGetValueByPath(t *testing.T) {
	type Nested struct {
		Value string `json:"nestedValue"`
	}
	type TestStruct struct {
		Key    string  `json:"key"`
		Nested *Nested `json:"nested"`
		Map    map[string]interface{}
		SkipMe string `json:"-"`
		NoTag  string
	}

	data := TestStruct{
		Key:    "value1",
		Nested: &Nested{Value: "nestedValue1"},
		Map: map[string]interface{}{
			"mapKey1": "mapValue1",
			"nestedMap": map[string]string{
				"innerKey": "innerValue",
			},
		},
		SkipMe: "should be skipped",
		NoTag:  "noTagValue",
	}

	testCases := []struct {
		name     string
		path     string
		expected interface{}
		found    bool
	}{
		{"Simple Field", "Key", "value1", true},
		{"Simple Field Case Insensitive", "key", "value1", true},
		{"Nested Field", "Nested.Value", "nestedValue1", true},
		{"Nested Field Case Insensitive", "nested.value", "nestedValue1", true},
		{"Map Field", "Map.mapKey1", "mapValue1", true},
		{"Map Field Case Insensitive", "map.mapkey1", "mapValue1", true},
		{"Nested Map Field", "Map.nestedMap.innerKey", "innerValue", true},
		{"JSON Tag Field", "key", "value1", true},                             // Matches JSON tag 'key'
		{"Nested JSON Tag Field", "nested.nestedValue", "nestedValue1", true}, // Matches JSON tag 'nestedValue'
		{"Skipped JSON Tag", "SkipMe", nil, false},
		{"Field with No Tag", "NoTag", "noTagValue", true},
		{"Non-existent Field", "NonExistent", nil, false},
		{"Non-existent Nested Field", "Nested.NonExistent", nil, false},
		{"Non-existent Map Key", "Map.NonExistent", nil, false},
		{"Traverse Nil Pointer", "Nested.Value", nil, false}, // Test with nil pointer
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			localData := data // Copy data for modification
			if tc.name == "Traverse Nil Pointer" {
				localData.Nested = nil // Modify data for this specific test case
			}
			val, found := getValueByPath(localData, tc.path)
			assert.Equal(t, tc.found, found)
			if tc.found {
				// Use assert.Equal for direct comparison, handle potential type differences if necessary
				assert.Equal(t, tc.expected, val)
			} else {
				assert.Nil(t, val)
			}
		})
	}
}

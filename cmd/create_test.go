package cmd

import (
	"bytes"
	"errors" // Keep for potential error mocking
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/karolswdev/ticketron/internal/config"
	"github.com/karolswdev/ticketron/internal/llm"
	"github.com/karolswdev/ticketron/internal/mcpclient"
)

// --- Mocks (Shared mocks are now in mocks_test.go) ---

// MockConfigProvider is defined in mocks_test.go
// MockMCPClient is defined in mocks_test.go

// MockLLMClient is now defined in mocks_test.go

// MockMCPClient is defined in mocks_test.go

// MockProjectMapper is a mock implementation of the ProjectMapper interface
type MockProjectMapper struct {
	mock.Mock
}

func (m *MockProjectMapper) MapSuggestionToKey(suggestion string, links *config.LinksConfig) (projectKey string, matchedLink *config.ProjectLink, err error) {
	args := m.Called(suggestion, links)
	var linkPtr *config.ProjectLink
	if linkArg := args.Get(1); linkArg != nil {
		linkPtr = linkArg.(*config.ProjectLink)
	}
	return args.String(0), linkPtr, args.Error(2)
}

// MockIssueTypeResolver is a mock implementation of the IssueTypeResolver interface
type MockIssueTypeResolver struct {
	mock.Mock
}

// Resolve signature updated to match interface (string, *config.ProjectLink, string)
func (m *MockIssueTypeResolver) Resolve(flagType string, projectLink *config.ProjectLink, defaultType string) string {
	args := m.Called(flagType, projectLink, defaultType)
	return args.String(0)
}

// --- Helper ---

// executeCreateCmd simulates running the create command by injecting mocks into the runner
func executeCreateCmd(
	loader ConfigProvider,
	llmClient llm.Client, // Use llm.Client interface
	mcpClient MCPClient,
	projectMapper ProjectMapper,
	issueTypeResolver IssueTypeResolver,
	args []string,
	flags map[string]string,
) (string, error) {

	// Removed mockMCPProvider factory function

	runner := &createCmdRunner{
		configProvider:    loader,
		llmClient:         llmClient, // Assign llm.Client
		mcpClient:         mcpClient, // Assign MCPClient directly
		projectMapper:     projectMapper,
		issueTypeResolver: issueTypeResolver,
	}

	cmd := createCmd

	cmd.ResetFlags()
	createCmd.Flags().StringVarP(&issueType, "type", "t", "", "Specify the JIRA issue type")
	createCmd.Flags().StringVarP(&issueSummary, "summary", "s", "", "[Optional] Specify the issue summary directly")
	createCmd.Flags().StringVarP(&projectKey, "project", "p", "", "[Optional] Specify the JIRA project key directly")
	createCmd.Flags().StringVarP(&description, "description", "d", "", "[Optional] Specify the issue description directly")

	for key, val := range flags {
		cmd.Flags().Set(key, val)
	}

	buf := new(bytes.Buffer)
	// Don't capture stdout/stderr for unit tests unless specifically needed
	// cmd.SetOut(buf)
	// cmd.SetErr(buf)

	err := runner.Run(cmd, args)
	output := buf.String() // Output will likely be empty now

	return output, err
}

// --- Tests ---

func TestCreateCmdRunE_Success(t *testing.T) {
	Log = zerolog.Nop()

	mockProvider := new(MockConfigProvider)
	mockLLM := new(MockLLMClient) // Use new mock name
	mockMCP := new(MockMCPClient)
	mockMapper := new(MockProjectMapper)
	mockResolver := new(MockIssueTypeResolver)

	testAppConfig := &config.AppConfig{ // Use mockProvider
		MCPServerURL: "http://mcp.example.com",
		LLM:          config.LLMConfig{Provider: "openai", OpenAI: config.OpenAIConfig{ModelName: "gpt-4"}}, // Updated config structure
	}
	mockProvider.On("LoadConfig").Return(testAppConfig, nil)

	testLinksConfig := &config.LinksConfig{
		Projects: []config.ProjectLink{
			{Name: "Test Project", Key: "TEST", DefaultIssueType: "Task"},
		},
	}
	mockProvider.On("LoadLinks").Return(testLinksConfig, nil)
	mockProvider.On("LoadSystemPrompt").Return("System prompt content", nil)
	mockProvider.On("LoadContext").Return("Context content", nil)

	expectedLLMResponse := llm.LLMResponse{
		Summary:               "Generated Title",
		Description:           "Generated Description",
		ProjectNameSuggestion: "Test Project",
	}
	// Removed AppConfig from mock call
	mockLLM.On("GenerateTicketDetails", mock.AnythingOfType("context.backgroundCtx"), "Test Summary", "System prompt content", "Context content").Return(expectedLLMResponse, nil)

	matchedLinkPtr := &testLinksConfig.Projects[0]
	mockMapper.On("MapSuggestionToKey", "Test Project", testLinksConfig).Return("TEST", matchedLinkPtr, nil)

	// Corrected Resolve call signature - Expecting project key ("TEST") as 3rd arg based on current create.go behavior
	mockResolver.On("Resolve", "", matchedLinkPtr, "TEST").Return("Task") // Expect empty flag, project link, project key ("TEST")

	expectedMCPRequest := mcpclient.CreateIssueRequest{
		ProjectKey:  "TEST",
		IssueType:   "Task",
		Summary:     "Generated Title",
		Description: "Generated Description",
	}
	mockMCP.On("CreateIssue", mock.AnythingOfType("context.backgroundCtx"), expectedMCPRequest).Return(&mcpclient.CreateIssueResponse{Key: "TEST-123", ID: "10001", Self: "http://jira.example.com/browse/TEST-123"}, nil)

	args := []string{"Test Summary"}
	flags := map[string]string{}
	_, err := executeCreateCmd(mockProvider, mockLLM, mockMCP, mockMapper, mockResolver, args, flags) // Ignore output

	assert.NoError(t, err)

	mockProvider.AssertExpectations(t)
	mockLLM.AssertExpectations(t)
	mockMapper.AssertExpectations(t)
	mockResolver.AssertExpectations(t)
	mockMCP.AssertExpectations(t)
}

func TestCreateCmdRunE_LoadConfigError(t *testing.T) {
	Log = zerolog.Nop()

	mockProvider := new(MockConfigProvider)
	mockLLM := new(MockLLMClient) // Use new mock name
	mockMapper := new(MockProjectMapper)
	mockResolver := new(MockIssueTypeResolver)

	expectedError := errors.New("config load error")
	mockProvider.On("LoadConfig").Return(nil, expectedError)

	args := []string{"Test Summary"}
	flags := map[string]string{}
	_, err := executeCreateCmd(mockProvider, mockLLM, nil, mockMapper, mockResolver, args, flags)

	assert.Error(t, err)
	assert.ErrorContains(t, err, expectedError.Error()) // Check for wrapped error message

	mockProvider.AssertCalled(t, "LoadConfig")
	mockProvider.AssertNumberOfCalls(t, "LoadConfig", 1)
	mockProvider.AssertNotCalled(t, "LoadLinks")
	mockProvider.AssertNotCalled(t, "LoadSystemPrompt")
	mockProvider.AssertNotCalled(t, "LoadContext")
	// Removed AppConfig from mock assertion arguments
	mockLLM.AssertNotCalled(t, "GenerateTicketDetails", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockMapper.AssertNotCalled(t, "MapSuggestionToKey", mock.Anything, mock.Anything)
	mockResolver.AssertNotCalled(t, "Resolve", mock.Anything, mock.Anything, mock.Anything)
}

func TestCreateCmdRunE_LoadLinksError(t *testing.T) {
	Log = zerolog.Nop()

	mockProvider := new(MockConfigProvider)
	mockLLM := new(MockLLMClient) // Use new mock name
	mockMapper := new(MockProjectMapper)
	mockResolver := new(MockIssueTypeResolver)

	mockProvider.On("LoadConfig").Return(&config.AppConfig{
		MCPServerURL: "http://mcp.example.com",
		LLM:          config.LLMConfig{Provider: "openai", OpenAI: config.OpenAIConfig{ModelName: "gpt-4"}}, // Updated config structure
	}, nil)

	expectedError := errors.New("links load error")
	mockProvider.On("LoadLinks").Return(nil, expectedError)

	args := []string{"Test Summary"}
	flags := map[string]string{}
	_, err := executeCreateCmd(mockProvider, mockLLM, nil, mockMapper, mockResolver, args, flags)

	assert.Error(t, err)
	assert.ErrorContains(t, err, expectedError.Error()) // Check for wrapped error message

	mockProvider.AssertCalled(t, "LoadConfig")
	mockProvider.AssertCalled(t, "LoadLinks")
	mockProvider.AssertNumberOfCalls(t, "LoadConfig", 1)
	mockProvider.AssertNumberOfCalls(t, "LoadLinks", 1)
	mockProvider.AssertNotCalled(t, "LoadSystemPrompt")
	mockProvider.AssertNotCalled(t, "LoadContext")
	// Removed AppConfig from mock assertion arguments
	mockLLM.AssertNotCalled(t, "GenerateTicketDetails", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockMapper.AssertNotCalled(t, "MapSuggestionToKey", mock.Anything, mock.Anything)
	mockResolver.AssertNotCalled(t, "Resolve", mock.Anything, mock.Anything, mock.Anything)
}

func TestCreateCmdRunE_LoadSystemPromptError(t *testing.T) {
	Log = zerolog.Nop()

	mockProvider := new(MockConfigProvider)
	mockLLM := new(MockLLMClient) // Use new mock name
	mockMapper := new(MockProjectMapper)
	mockResolver := new(MockIssueTypeResolver)

	mockProvider.On("LoadConfig").Return(&config.AppConfig{
		MCPServerURL: "http://mcp.example.com",
		LLM:          config.LLMConfig{Provider: "openai", OpenAI: config.OpenAIConfig{ModelName: "gpt-4"}}, // Updated config structure
	}, nil)
	mockProvider.On("LoadLinks").Return(&config.LinksConfig{ /* ... */ }, nil) // Need valid links

	expectedError := errors.New("prompt load error")
	mockProvider.On("LoadSystemPrompt").Return("", expectedError)

	args := []string{"Test Summary"}
	flags := map[string]string{}
	_, err := executeCreateCmd(mockProvider, mockLLM, nil, mockMapper, mockResolver, args, flags)

	assert.Error(t, err)
	assert.ErrorContains(t, err, expectedError.Error()) // Check for wrapped error message

	mockProvider.AssertCalled(t, "LoadConfig")
	mockProvider.AssertCalled(t, "LoadLinks")
	mockProvider.AssertCalled(t, "LoadSystemPrompt")
	mockProvider.AssertNumberOfCalls(t, "LoadConfig", 1)
	mockProvider.AssertNumberOfCalls(t, "LoadLinks", 1)
	mockProvider.AssertNumberOfCalls(t, "LoadSystemPrompt", 1)
	mockProvider.AssertNotCalled(t, "LoadContext")
	// Removed AppConfig from mock assertion arguments
	mockLLM.AssertNotCalled(t, "GenerateTicketDetails", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockMapper.AssertNotCalled(t, "MapSuggestionToKey", mock.Anything, mock.Anything)
	mockResolver.AssertNotCalled(t, "Resolve", mock.Anything, mock.Anything, mock.Anything)
}

func TestCreateCmdRunE_LoadContextError(t *testing.T) {
	Log = zerolog.Nop()

	mockProvider := new(MockConfigProvider)
	mockLLM := new(MockLLMClient) // Use new mock name
	mockMapper := new(MockProjectMapper)
	mockResolver := new(MockIssueTypeResolver)

	mockProvider.On("LoadConfig").Return(&config.AppConfig{
		MCPServerURL: "http://mcp.example.com",
		LLM:          config.LLMConfig{Provider: "openai", OpenAI: config.OpenAIConfig{ModelName: "gpt-4"}}, // Updated config structure
	}, nil)
	mockProvider.On("LoadLinks").Return(&config.LinksConfig{ /* ... */ }, nil)
	mockProvider.On("LoadSystemPrompt").Return("System prompt content", nil)

	expectedError := errors.New("context load error")
	mockProvider.On("LoadContext").Return("", expectedError)

	args := []string{"Test Summary"}
	flags := map[string]string{}
	_, err := executeCreateCmd(mockProvider, mockLLM, nil, mockMapper, mockResolver, args, flags)

	assert.Error(t, err)
	assert.ErrorContains(t, err, expectedError.Error()) // Check for wrapped error message

	mockProvider.AssertCalled(t, "LoadConfig")
	mockProvider.AssertCalled(t, "LoadLinks")
	mockProvider.AssertCalled(t, "LoadSystemPrompt")
	mockProvider.AssertCalled(t, "LoadContext")
	mockProvider.AssertNumberOfCalls(t, "LoadConfig", 1)
	mockProvider.AssertNumberOfCalls(t, "LoadLinks", 1)
	mockProvider.AssertNumberOfCalls(t, "LoadSystemPrompt", 1)
	mockProvider.AssertNumberOfCalls(t, "LoadContext", 1)
	// Removed AppConfig from mock assertion arguments
	mockLLM.AssertNotCalled(t, "GenerateTicketDetails", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockMapper.AssertNotCalled(t, "MapSuggestionToKey", mock.Anything, mock.Anything)
	mockResolver.AssertNotCalled(t, "Resolve", mock.Anything, mock.Anything, mock.Anything)
}

func TestCreateCmdRunE_LLMGenerateError(t *testing.T) {
	Log = zerolog.Nop()

	mockProvider := new(MockConfigProvider)
	mockLLM := new(MockLLMClient) // Use new mock name
	mockMapper := new(MockProjectMapper)
	mockResolver := new(MockIssueTypeResolver)

	testAppConfig := &config.AppConfig{
		MCPServerURL: "http://mcp.example.com",
		LLM:          config.LLMConfig{Provider: "openai", OpenAI: config.OpenAIConfig{ModelName: "gpt-4"}}, // Updated config structure
	}
	mockProvider.On("LoadConfig").Return(testAppConfig, nil)
	mockProvider.On("LoadLinks").Return(&config.LinksConfig{ /* ... */ }, nil)
	mockProvider.On("LoadSystemPrompt").Return("System prompt content", nil)
	mockProvider.On("LoadContext").Return("Context content", nil)

	expectedError := errors.New("llm generate error")
	// Removed AppConfig from mock call
	mockLLM.On("GenerateTicketDetails", mock.AnythingOfType("context.backgroundCtx"), "Test Summary", "System prompt content", "Context content").Return(llm.LLMResponse{}, expectedError)

	args := []string{"Test Summary"}
	flags := map[string]string{}
	_, err := executeCreateCmd(mockProvider, mockLLM, nil, mockMapper, mockResolver, args, flags)

	assert.Error(t, err)
	assert.Equal(t, expectedError, err)

	mockProvider.AssertCalled(t, "LoadConfig")
	mockProvider.AssertCalled(t, "LoadLinks")
	mockProvider.AssertCalled(t, "LoadSystemPrompt")
	mockProvider.AssertCalled(t, "LoadContext")
	// Removed AppConfig from mock assertion arguments
	mockLLM.AssertCalled(t, "GenerateTicketDetails", mock.AnythingOfType("context.backgroundCtx"), "Test Summary", "System prompt content", "Context content")
	mockLLM.AssertNumberOfCalls(t, "GenerateTicketDetails", 1)
	mockMapper.AssertNotCalled(t, "MapSuggestionToKey", mock.Anything, mock.Anything)
	mockResolver.AssertNotCalled(t, "Resolve", mock.Anything, mock.Anything, mock.Anything)
}

func TestCreateCmdRunE_MapProjectError(t *testing.T) {
	Log = zerolog.Nop()

	mockProvider := new(MockConfigProvider)
	mockLLM := new(MockLLMClient) // Use new mock name
	mockMapper := new(MockProjectMapper)
	mockResolver := new(MockIssueTypeResolver)

	testAppConfig := &config.AppConfig{
		MCPServerURL: "http://mcp.example.com",
		LLM:          config.LLMConfig{Provider: "openai", OpenAI: config.OpenAIConfig{ModelName: "gpt-4"}}, // Updated config structure
	}
	mockProvider.On("LoadConfig").Return(testAppConfig, nil)
	testLinksConfig := &config.LinksConfig{ /* ... */ } // Use a valid config
	mockProvider.On("LoadLinks").Return(testLinksConfig, nil)
	mockProvider.On("LoadSystemPrompt").Return("System prompt content", nil)
	mockProvider.On("LoadContext").Return("Context content", nil)

	llmSuggestion := "Unknown Project"
	// Removed AppConfig from mock call
	mockLLM.On("GenerateTicketDetails", mock.AnythingOfType("context.backgroundCtx"), "Test Summary", "System prompt content", "Context content").Return(llm.LLMResponse{ProjectNameSuggestion: llmSuggestion}, nil)

	expectedError := errors.New("map project error")
	mockMapper.On("MapSuggestionToKey", llmSuggestion, testLinksConfig).Return("", nil, expectedError)

	args := []string{"Test Summary"}
	flags := map[string]string{}
	_, err := executeCreateCmd(mockProvider, mockLLM, nil, mockMapper, mockResolver, args, flags)

	assert.Error(t, err)
	assert.Equal(t, expectedError, err)

	mockProvider.AssertCalled(t, "LoadConfig")
	mockProvider.AssertCalled(t, "LoadLinks")
	mockProvider.AssertCalled(t, "LoadSystemPrompt")
	mockProvider.AssertCalled(t, "LoadContext")
	// Removed AppConfig from mock assertion arguments
	mockLLM.AssertCalled(t, "GenerateTicketDetails", mock.AnythingOfType("context.backgroundCtx"), "Test Summary", "System prompt content", "Context content")
	mockMapper.AssertCalled(t, "MapSuggestionToKey", llmSuggestion, testLinksConfig)
	mockMapper.AssertNumberOfCalls(t, "MapSuggestionToKey", 1)
	mockResolver.AssertNotCalled(t, "Resolve", mock.Anything, mock.Anything, mock.Anything)
}

func TestCreateCmdRunE_MCPCreateError(t *testing.T) {
	Log = zerolog.Nop()

	mockProvider := new(MockConfigProvider)
	mockLLM := new(MockLLMClient) // Use new mock name
	mockMCP := new(MockMCPClient) // Instantiate MCP mock
	mockMapper := new(MockProjectMapper)
	mockResolver := new(MockIssueTypeResolver)

	testAppConfig := &config.AppConfig{
		MCPServerURL: "http://mcp.example.com",
		LLM:          config.LLMConfig{Provider: "openai", OpenAI: config.OpenAIConfig{ModelName: "gpt-4"}}, // Updated config structure
	}
	mockProvider.On("LoadConfig").Return(testAppConfig, nil)
	testLinksConfig := &config.LinksConfig{
		Projects: []config.ProjectLink{
			{Name: "Test Project", Key: "TEST", DefaultIssueType: "Task"},
		},
	}
	mockProvider.On("LoadLinks").Return(testLinksConfig, nil)
	mockProvider.On("LoadSystemPrompt").Return("System prompt content", nil)
	mockProvider.On("LoadContext").Return("Context content", nil)

	expectedLLMResponse := llm.LLMResponse{
		Summary:               "Generated Title",
		Description:           "Generated Description",
		ProjectNameSuggestion: "Test Project",
	}
	// Removed AppConfig from mock call
	mockLLM.On("GenerateTicketDetails", mock.AnythingOfType("context.backgroundCtx"), "Test Summary", "System prompt content", "Context content").Return(expectedLLMResponse, nil)

	matchedLinkPtr := &testLinksConfig.Projects[0]
	mockMapper.On("MapSuggestionToKey", "Test Project", testLinksConfig).Return("TEST", matchedLinkPtr, nil)

	// Corrected Resolve call signature - Expecting project key ("TEST") as 3rd arg
	mockResolver.On("Resolve", "", matchedLinkPtr, "TEST").Return("Task")

	expectedMCPRequest := mcpclient.CreateIssueRequest{
		ProjectKey:  "TEST",
		IssueType:   "Task",
		Summary:     "Generated Title",
		Description: "Generated Description",
	}
	expectedError := errors.New("mcp create error")
	mockMCP.On("CreateIssue", mock.AnythingOfType("context.backgroundCtx"), expectedMCPRequest).Return(nil, expectedError)

	args := []string{"Test Summary"}
	flags := map[string]string{}
	_, err := executeCreateCmd(mockProvider, mockLLM, mockMCP, mockMapper, mockResolver, args, flags)

	assert.Error(t, err)
	assert.Equal(t, expectedError, err)

	mockProvider.AssertExpectations(t)
	mockLLM.AssertExpectations(t)
	mockMapper.AssertExpectations(t)
	mockResolver.AssertExpectations(t)
	mockMCP.AssertExpectations(t)
}

func TestCreateCmdRunE_TypeFlagOverride(t *testing.T) {
	Log = zerolog.Nop()

	mockProvider := new(MockConfigProvider)
	mockLLM := new(MockLLMClient) // Use mock from mocks_test.go
	mockMCP := new(MockMCPClient)
	mockMapper := new(MockProjectMapper)
	mockResolver := new(MockIssueTypeResolver)

	testAppConfig := &config.AppConfig{
		MCPServerURL: "http://mcp.example.com",
		LLM:          config.LLMConfig{Provider: "openai", OpenAI: config.OpenAIConfig{ModelName: "gpt-4"}}, // Updated config structure
	}
	mockProvider.On("LoadConfig").Return(testAppConfig, nil)
	testLinksConfig := &config.LinksConfig{
		Projects: []config.ProjectLink{
			{Name: "Test Project", Key: "TEST", DefaultIssueType: "Task"}, // Default is Task
		},
	}
	mockProvider.On("LoadLinks").Return(testLinksConfig, nil)
	mockProvider.On("LoadSystemPrompt").Return("System prompt content", nil)
	mockProvider.On("LoadContext").Return("Context content", nil)

	expectedLLMResponse := llm.LLMResponse{
		Summary:               "Generated Title",
		Description:           "Generated Description",
		ProjectNameSuggestion: "Test Project",
	}
	// Removed AppConfig from mock call
	mockLLM.On("GenerateTicketDetails", mock.AnythingOfType("context.backgroundCtx"), "Test Summary", "System prompt content", "Context content").Return(expectedLLMResponse, nil)

	matchedLinkPtr := &testLinksConfig.Projects[0]
	mockMapper.On("MapSuggestionToKey", "Test Project", testLinksConfig).Return("TEST", matchedLinkPtr, nil)

	flagType := "Bug" // Override with Bug
	// Expect Resolve to be called with the flag type and return it
	// Corrected Resolve call signature - Expecting project key ("TEST") as 3rd arg
	mockResolver.On("Resolve", flagType, matchedLinkPtr, "TEST").Return(flagType)

	// Expect MCP request to use the overridden type
	expectedMCPRequest := mcpclient.CreateIssueRequest{
		ProjectKey:  "TEST",
		IssueType:   flagType, // Should be "Bug"
		Summary:     "Generated Title",
		Description: "Generated Description",
	}
	mockMCP.On("CreateIssue", mock.AnythingOfType("context.backgroundCtx"), expectedMCPRequest).Return(&mcpclient.CreateIssueResponse{Key: "TEST-456", ID: "10002", Self: "http://jira.example.com/browse/TEST-456"}, nil)

	args := []string{"Test Summary"}
	flags := map[string]string{"type": flagType} // Pass the flag
	_, err := executeCreateCmd(mockProvider, mockLLM, mockMCP, mockMapper, mockResolver, args, flags)

	assert.NoError(t, err)

	mockProvider.AssertExpectations(t)
	mockLLM.AssertExpectations(t)
	mockMapper.AssertExpectations(t)
	mockResolver.AssertExpectations(t)
	mockMCP.AssertExpectations(t)
}

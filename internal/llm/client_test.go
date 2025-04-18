package llm

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	openai "github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewOpenAIClient tests the constructor for OpenAIClient.
func TestNewOpenAIClient(t *testing.T) {
	t.Run("Nil_OpenAI_Client", func(t *testing.T) {
		_, err := NewOpenAIClient(nil, "test-model")
		require.Error(t, err, "Expected error when providing a nil openai.Client")
		assert.ErrorIs(t, err, ErrLLMClientNil, "Expected ErrLLMClientNil")
	})

	t.Run("Empty_ModelName_Defaults", func(t *testing.T) {
		// Need a non-nil client
		dummyClient := openai.NewClient("dummy-key")
		llmClient, err := NewOpenAIClient(dummyClient, "") // Empty model name
		require.NoError(t, err, "Should not error with empty model name")
		require.NotNil(t, llmClient, "Client should be created")
		assert.Equal(t, openai.GPT4o, llmClient.modelName, "Model name should default to openai.GPT4o")
	})

	t.Run("Valid_Input", func(t *testing.T) {
		dummyClient := openai.NewClient("dummy-key")
		model := "custom-model"
		llmClient, err := NewOpenAIClient(dummyClient, model)
		require.NoError(t, err, "Should not error with valid input")
		require.NotNil(t, llmClient, "Client should be created")
		assert.Equal(t, model, llmClient.modelName, "Model name should be set correctly")
		assert.Equal(t, dummyClient, llmClient.client, "Internal openai.Client should be set")
	})
}

// TestOpenAIClient_GenerateTicketDetails tests the primary method of the OpenAIClient.
func TestOpenAIClient_GenerateTicketDetails(t *testing.T) {
	testCases := []struct {
		name             string
		userInput        string
		systemPrompt     string
		contextContent   string
		mockResponse     string // Raw JSON string expected from OpenAI
		mockStatusCode   int
		expectError      bool
		expectedResponse LLMResponse // Expected parsed struct
		expectedErrMsg   string      // Substring to check in error message if expectError is true
	}{
		{
			name:           "Successful_Generation_And_Parse",
			userInput:      "Create a task for refactoring the auth module.",
			systemPrompt:   "You are a Jira bot.",
			contextContent: "Auth module uses legacy code.",
			mockResponse: `{
				"id": "chatcmpl-123",
				"object": "chat.completion",
				"created": 1677652288,
				"model": "gpt-4o",
				"choices": [{
					"index": 0,
					"message": {
						"role": "assistant",
						"content": "{\n  \"summary\": \"Refactor auth module\",\n  \"description\": \"The auth module needs refactoring due to legacy code.\",\n  \"project_name_suggestion\": \"Backend\"\n}"
					},
					"finish_reason": "stop"
				}],
				"usage": {"prompt_tokens": 50, "completion_tokens": 50, "total_tokens": 100}
			}`, // Note: Content is the JSON string LLMResponse expects
			mockStatusCode: http.StatusOK,
			expectError:    false,
			expectedResponse: LLMResponse{
				Summary:               "Refactor auth module",
				Description:           "The auth module needs refactoring due to legacy code.",
				ProjectNameSuggestion: "Backend",
			},
		},
		{
			name:           "API_Error_Response",
			userInput:      "Another prompt",
			systemPrompt:   "You are a Jira bot.",
			contextContent: "",
			mockResponse: `{
				"error": { "message": "Invalid API key.", "type": "invalid_request_error", "code": "invalid_api_key" }
			}`,
			mockStatusCode: http.StatusUnauthorized,
			expectError:    true,
			expectedErrMsg: "failed to create LLM completion", // Updated expected error prefix
		},
		{
			name:           "Empty_Choices_In_Response",
			userInput:      "Prompt leading to empty choices",
			systemPrompt:   "You are a Jira bot.",
			contextContent: "",
			mockResponse: `{
				"id": "chatcmpl-456", "object": "chat.completion", "created": 1677652290, "model": "gpt-4o", "choices": [],
				"usage": {"prompt_tokens": 10, "completion_tokens": 0, "total_tokens": 10}
			}`,
			mockStatusCode: http.StatusOK,
			expectError:    true,
			expectedErrMsg: ErrLLMEmptyResponse.Error(), // Check for specific error
		},
		{
			name:           "Malformed_JSON_In_Response",
			userInput:      "Create task",
			systemPrompt:   "You are a Jira bot.",
			contextContent: "",
			mockResponse: `{
				"id": "chatcmpl-789", "object": "chat.completion", "created": 1677652299, "model": "gpt-4o",
				"choices": [{"index": 0, "message": {"role": "assistant", "content": "{ \"summary\": \"Test\", \"description\": \"Missing quote }"}, "finish_reason": "stop"}]
			}`, // Malformed JSON in content
			mockStatusCode: http.StatusOK,
			expectError:    true,
			expectedErrMsg: "failed to parse LLM response", // Check for wrapper message from GenerateTicketDetails
		},
		{
			name:           "Missing_Required_Field_In_Parsed_JSON",
			userInput:      "Create task",
			systemPrompt:   "You are a Jira bot.",
			contextContent: "",
			mockResponse: `{
				"id": "chatcmpl-abc", "object": "chat.completion", "created": 1677652300, "model": "gpt-4o",
				"choices": [{"index": 0, "message": {"role": "assistant", "content": "{ \"description\": \"Only description\", \"project_name_suggestion\": \"Proj\" }"}, "finish_reason": "stop"}]
			}`, // Missing "summary"
			mockStatusCode: http.StatusOK,
			expectError:    true,
			expectedErrMsg: ErrLLMResponseMissingField.Error(), // Check for specific error from ParseLLMResponse
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/chat/completions" {
					http.Error(w, "Not Found", http.StatusNotFound)
					return
				}
				w.Header().Set("Content-Type", "application/json") // Set content type
				w.WriteHeader(tc.mockStatusCode)
				fmt.Fprintln(w, tc.mockResponse)
			}))
			defer server.Close()

			// Configure OpenAI client to use the mock server
			config := openai.DefaultConfig("dummy-api-key")
			config.BaseURL = server.URL + "/v1"
			openaiSdkClient := openai.NewClientWithConfig(config)

			// Create the llm.Client (OpenAIClient) instance
			llmClient, err := NewOpenAIClient(openaiSdkClient, "test-model")
			require.NoError(t, err, "Failed to create OpenAIClient for test")

			// Call the function under test
			response, err := llmClient.GenerateTicketDetails(context.Background(), tc.userInput, tc.systemPrompt, tc.contextContent)

			// Assertions
			if tc.expectError {
				require.Error(t, err, "Expected an error, but got nil")
				if tc.expectedErrMsg != "" {
					assert.Contains(t, err.Error(), tc.expectedErrMsg, "Error message mismatch")
				}
			} else {
				require.NoError(t, err, "Did not expect an error, but got: %v", err)
				assert.Equal(t, tc.expectedResponse, response, "Expected response mismatch")
			}
		})
	}
}

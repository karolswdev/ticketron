package llm

import (
	"testing"
)

func TestParseLLMResponse(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		expectError bool
		expected    LLMResponse // Only checked if expectError is false
	}{
		{
			name:        "Valid JSON",
			input:       `{"summary": "Test Summary", "description": "Test Desc", "project_name_suggestion": "TESTPROJ"}`,
			expectError: false,
			expected: LLMResponse{
				Summary:               "Test Summary",
				Description:           "Test Desc",
				ProjectNameSuggestion: "TESTPROJ",
			},
		},
		{
			name:        "Invalid JSON - Syntax Error",
			input:       `{"summary": "Test Summary", "description": "Test Desc", "project_name_suggestion": "TESTPROJ",}`, // Extra comma
			expectError: true,
		},
		{
			name:        "Invalid JSON - Missing Key",
			input:       `{"summary": "Test Summary", "description": "Test Desc"}`, // Missing project_name_suggestion
			expectError: true,                                                      // JSON is valid, but validation should fail due to missing required key
			expected: LLMResponse{
				Summary:               "Test Summary",
				Description:           "Test Desc",
				ProjectNameSuggestion: "", // Expect zero value for missing key
			},
		},
		{
			name:        "Empty Input String",
			input:       "",
			expectError: true,
		},
		{
			name:        "Valid JSON with Extra Keys",
			input:       `{"summary": "Test Summary", "description": "Test Desc", "project_name_suggestion": "TESTPROJ", "extra_key": "ignore_me"}`,
			expectError: false,
			expected: LLMResponse{
				Summary:               "Test Summary",
				Description:           "Test Desc",
				ProjectNameSuggestion: "TESTPROJ",
			},
		},
		{
			name:        "JSON is just a string",
			input:       `"this is not a json object"`,
			expectError: true, // Expecting a JSON object, not just a string literal
		},
		{
			name:        "Valid JSON with standard markdown fence",
			input:       "```json\n{\"summary\": \"Fence Summary\", \"description\": \"Fence Desc\", \"project_name_suggestion\": \"FENCE\"}\n```",
			expectError: false,
			expected: LLMResponse{
				Summary:               "Fence Summary",
				Description:           "Fence Desc",
				ProjectNameSuggestion: "FENCE",
			},
		},
		{
			name:        "Valid JSON with fence missing specifier",
			input:       "```\n{\"summary\": \"No Spec Summary\", \"description\": \"No Spec Desc\", \"project_name_suggestion\": \"NOSPEC\"}\n```",
			expectError: false,
			expected: LLMResponse{
				Summary:               "No Spec Summary",
				Description:           "No Spec Desc",
				ProjectNameSuggestion: "NOSPEC",
			},
		},
		{
			name:        "Valid JSON with uppercase JSON fence specifier",
			input:       "```JSON\n{\"summary\": \"Upper Summary\", \"description\": \"Upper Desc\", \"project_name_suggestion\": \"UPPER\"}\n```",
			expectError: false,
			expected: LLMResponse{
				Summary:               "Upper Summary",
				Description:           "Upper Desc",
				ProjectNameSuggestion: "UPPER",
			},
		},
		{
			name:        "Valid JSON with extra backticks in fence", // Regex handles 3+
			input:       "````json\n{\"summary\": \"Extra Ticks\", \"description\": \"Extra Desc\", \"project_name_suggestion\": \"EXTRATICKS\"}\n````",
			expectError: false,
			expected: LLMResponse{
				Summary:               "Extra Ticks",
				Description:           "Extra Desc",
				ProjectNameSuggestion: "EXTRATICKS",
			},
		},
		{
			name:        "Valid JSON with leading/trailing whitespace around fences",
			input:       "  \n\n ```json\n{\"summary\": \"Whitespace Summary\", \"description\": \"Whitespace Desc\", \"project_name_suggestion\": \"WHITESPACE\"}\n``` \n ",
			expectError: false,
			expected: LLMResponse{
				Summary:               "Whitespace Summary",
				Description:           "Whitespace Desc",
				ProjectNameSuggestion: "WHITESPACE",
			},
		},
		{
			name:        "Valid JSON with leading/trailing whitespace inside fences",
			input:       "```json \n \n {\"summary\": \"Inner Space\", \"description\": \"Inner Desc\", \"project_name_suggestion\": \"INNERSPACE\"} \n \n ```",
			expectError: false,
			expected: LLMResponse{
				Summary:               "Inner Space",
				Description:           "Inner Desc",
				ProjectNameSuggestion: "INNERSPACE",
			},
		},
		{
			name:        "Valid JSON missing optional description field",
			input:       "```json\n{\"summary\": \"No Desc\", \"project_name_suggestion\": \"NODESC\"}\n```",
			expectError: false,
			expected: LLMResponse{
				Summary:               "No Desc",
				Description:           "", // Expect zero value
				ProjectNameSuggestion: "NODESC",
			},
		},
		{
			name:        "Valid JSON with text before fence",
			input:       "Here is the JSON:\n```json\n{\"summary\": \"Text Before\", \"description\": \"TB Desc\", \"project_name_suggestion\": \"TEXTBEFORE\"}\n```",
			expectError: false,
			expected: LLMResponse{
				Summary:               "Text Before",
				Description:           "TB Desc",
				ProjectNameSuggestion: "TEXTBEFORE",
			},
		},
		{
			name:        "Valid JSON with text after fence",
			input:       "```json\n{\"summary\": \"Text After\", \"description\": \"TA Desc\", \"project_name_suggestion\": \"TEXTAFTER\"}\n```\nLet me know if this works.",
			expectError: false,
			expected: LLMResponse{
				Summary:               "Text After",
				Description:           "TA Desc",
				ProjectNameSuggestion: "TEXTAFTER",
			},
		},
		{
			name:        "Valid JSON with text before and after fence",
			input:       "Okay, here's the data:\n```json\n{\"summary\": \"Text Both\", \"description\": \"Both Desc\", \"project_name_suggestion\": \"TEXTBOTH\"}\n```\nHope this helps!",
			expectError: false,
			expected: LLMResponse{
				Summary:               "Text Both",
				Description:           "Both Desc",
				ProjectNameSuggestion: "TEXTBOTH",
			},
		},
		{
			name:        "Malformed fence - missing closing backticks",
			input:       "```json\n{\"summary\": \"Bad Fence\", \"project_name_suggestion\": \"BADFENCE\"}",
			expectError: true, // Regex won't match, fallback won't find closing brace
		},
		{
			name:        "No JSON object found",
			input:       "Just some random text without any JSON.",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ParseLLMResponse(tc.input)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected an error, but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Did not expect an error, but got: %v", err)
				}
				if result.Summary != tc.expected.Summary {
					t.Errorf("Expected Summary %q, got %q", tc.expected.Summary, result.Summary)
				}
				if result.Description != tc.expected.Description {
					t.Errorf("Expected Description %q, got %q", tc.expected.Description, result.Description)
				}
				if result.ProjectNameSuggestion != tc.expected.ProjectNameSuggestion {
					t.Errorf("Expected ProjectNameSuggestion %q, got %q", tc.expected.ProjectNameSuggestion, result.ProjectNameSuggestion)
				}
			}
		})
	}
}

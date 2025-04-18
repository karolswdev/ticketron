package cmd

// This file contains mock implementations used across different test files
// within the cmd package, but which need to be accessible from outside
// _test.go files (e.g., for integration tests).

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/karolswdev/ticketron/internal/llm"
)

// --- Mock LLMClient ---

// MockLLMClient is a mock implementation of the llm.Client interface.
// Exported for use in integration tests.
type MockLLMClient struct {
	mock.Mock // Implements llm.Client
}

// GenerateTicketDetails matches llm.Client interface
func (m *MockLLMClient) GenerateTicketDetails(ctx context.Context, userInput, systemPrompt, contextContent string) (llm.LLMResponse, error) {
	args := m.Called(ctx, userInput, systemPrompt, contextContent)
	respArg := args.Get(0)
	var resp llm.LLMResponse
	if respArg != nil {
		resp = respArg.(llm.LLMResponse)
	}
	return resp, args.Error(1) // Return potentially zero struct and error
}

// Add other shared mocks here if needed later.

package llm

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	openai "github.com/sashabaranov/go-openai"
)

// Client defines the interface for interacting with different LLM providers.
type Client interface {
	// GenerateTicketDetails takes user input, system prompt, and context, interacts with the LLM,
	// parses the response, and returns the structured ticket details or an error.
	GenerateTicketDetails(ctx context.Context, userInput, systemPrompt, contextContent string) (LLMResponse, error)
}

// OpenAIClient implements the llm.Client interface for the OpenAI API.
type OpenAIClient struct {
	client    *openai.Client
	modelName string
}

// NewOpenAIClient creates a new OpenAI client wrapper.
// It requires a configured go-openai client and the model name to use.
func NewOpenAIClient(client *openai.Client, modelName string) (*OpenAIClient, error) {
	if client == nil {
		return nil, ErrLLMClientNil // Return nil for *OpenAIClient on error
	}
	if modelName == "" {
		log.Warn().Msg("modelName is empty for OpenAIClient, defaulting to gpt-4o")
		modelName = openai.GPT4o // Default if empty
	}
	return &OpenAIClient{
		client:    client,
		modelName: modelName,
	}, nil
}

// GenerateTicketDetails implements the llm.Client interface for OpenAI.
// It constructs the prompt, calls the OpenAI API, and parses the response.
func (o *OpenAIClient) GenerateTicketDetails(ctx context.Context, userInput, systemPrompt, contextContent string) (LLMResponse, error) {
	// 1. Build the full prompt
	fullPrompt := ConstructPrompt(userInput, systemPrompt, contextContent)
	log.Debug().Str("full_prompt", fullPrompt).Msg("Constructed full prompt for LLM")

	// 2. Call the OpenAI API
	if o.client == nil {
		return LLMResponse{}, ErrLLMClientNil
	}
	if fullPrompt == "" {
		// Should not happen if ConstructPrompt works, but check anyway
		return LLMResponse{}, ErrLLMPromptEmpty
	}

	log.Debug().Str("model", o.modelName).Msg("Preparing OpenAI chat completion request")
	req := openai.ChatCompletionRequest{
		Model: o.modelName,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: fullPrompt,
			},
		},
	}

	log.Debug().Interface("request", req).Msg("Sending request to OpenAI API")
	resp, err := o.client.CreateChatCompletion(ctx, req) // Pass context
	if err != nil {
		log.Error().Err(err).Msg("OpenAI API call failed")
		return LLMResponse{}, fmt.Errorf("%w: %w", ErrLLMCompletion, err)
	}
	log.Debug().Interface("response", resp).Msg("Received response from OpenAI API")

	if len(resp.Choices) == 0 {
		log.Error().Msg("Received an empty response (no choices) from OpenAI")
		return LLMResponse{}, ErrLLMEmptyResponse
	}
	rawResponse := resp.Choices[0].Message.Content
	log.Debug().Str("raw_response", rawResponse).Msg("Extracted raw response content")

	// 3. Parse the response
	parsedResponse, err := ParseLLMResponse(rawResponse)
	if err != nil {
		// Error already logged in ParseLLMResponse
		return LLMResponse{}, fmt.Errorf("failed to parse LLM response: %w", err) // Wrap error from parser
	}

	log.Info().Msg("Successfully generated and parsed ticket details from OpenAI")
	return parsedResponse, nil
}

// Note: The old CallOpenAI function has been removed as its logic is integrated into GenerateTicketDetails.

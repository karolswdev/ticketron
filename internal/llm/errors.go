package llm

import "errors"

// Sentinel errors for LLM client and parsing operations.

// ErrLLMClientNil indicates the LLM client (e.g., OpenAI client) was nil when used.
var ErrLLMClientNil = errors.New("LLM client cannot be nil")

// ErrLLMPromptEmpty indicates the prompt provided to the LLM was empty.
var ErrLLMPromptEmpty = errors.New("prompt cannot be empty")

// ErrLLMCompletion indicates an error occurred during the LLM API call (e.g., network error, API error).
// The underlying error from the LLM SDK should be wrapped.
var ErrLLMCompletion = errors.New("failed to create LLM completion")

// ErrLLMEmptyResponse indicates the LLM returned a response with no usable content (e.g., no choices).
var ErrLLMEmptyResponse = errors.New("received an empty response from LLM")

// ErrLLMResponseParse indicates a general error occurred while parsing the LLM's raw response.
var ErrLLMResponseParse = errors.New("failed to parse LLM response")

// ErrLLMResponseJSONFind indicates the expected JSON object could not be found in the LLM response.
var ErrLLMResponseJSONFind = errors.New("failed to find JSON object in LLM response")

// ErrLLMResponseJSONUnmarshal indicates an error occurred while unmarshaling the JSON from the LLM response.
// The underlying JSON error should be wrapped.
var ErrLLMResponseJSONUnmarshal = errors.New("failed to unmarshal LLM response JSON")

// ErrLLMResponseMissingField indicates a required field was missing from the parsed LLM response JSON.
// The specific missing field should be mentioned in the error message where this is returned.
var ErrLLMResponseMissingField = errors.New("parsed LLM response is missing a required field")

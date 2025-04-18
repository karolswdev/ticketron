package llm

import (
	"encoding/json"
	"fmt"
	"regexp" // Import regexp
	"strings"

	"github.com/rs/zerolog/log"
)

// LLMResponse defines the structure expected for the JSON data returned by the LLM
// after processing a user's request for ticket creation. It includes fields for
// the suggested summary, description, and project alias.
// Note: The 'IssueType' field is currently missing from this struct based on the
// system prompt and expected LLM output format.
type LLMResponse struct {
	Summary               string `json:"summary"`
	Description           string `json:"description"` // Description is optional in validation
	ProjectNameSuggestion string `json:"project_name_suggestion"`
	// IssueType is currently missing based on file content
}

// ParseLLMResponse takes the raw string response from the LLM, attempts to clean it
// by removing potential markdown code fences and trimming whitespace, and then
// unmarshals the resulting JSON string into an LLMResponse struct.
// It performs basic validation to ensure required fields ('summary', 'project_name_suggestion')
// are present in the parsed response.
// It returns the parsed LLMResponse and nil error on success, or an error if
// unmarshalling or validation fails.
// Regex to find JSON possibly enclosed in markdown code fences.
// It looks for ``` (optional json specifier) ... ```
// and captures the content starting with { and ending with }.
// It handles leading/trailing whitespace around the JSON block.
// Using \x60 for backticks to avoid diff parser issues.
var jsonRegex = regexp.MustCompile(`(?s)\x60{3}(?:[jJ][sS][oO][nN])?\s*(\{.*\})\s*\x60{3}`)

func ParseLLMResponse(rawResponse string) (LLMResponse, error) {
	log.Debug().Str("raw_response", rawResponse).Msg("Attempting to parse LLM response")

	var jsonStr string
	match := jsonRegex.FindStringSubmatch(rawResponse)

	if len(match) == 2 {
		// Found JSON within ```json ... ``` or ``` ... ```
		jsonStr = match[1]
		log.Debug().Str("extracted_json", jsonStr).Msg("Extracted JSON using regex from code fences")
	} else {
		// Fallback: Assume the response might be JSON without fences, or with malformed fences.
		// Trim whitespace aggressively.
		trimmed := strings.TrimSpace(rawResponse)
		// Basic check if it looks like a JSON object
		if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") {
			jsonStr = trimmed
			log.Debug().Str("trimmed_json", jsonStr).Msg("Using trimmed response as JSON (no valid fences found)")
		} else {
			log.Error().Str("raw_response", rawResponse).Msg("Could not find JSON object within code fences or as a standalone object")
			return LLMResponse{}, ErrLLMResponseJSONFind // Use sentinel error
		}
	}

	// Trim potential extra whitespace from the extracted/trimmed JSON string itself
	jsonStr = strings.TrimSpace(jsonStr)

	log.Debug().Str("final_json_string", jsonStr).Msg("Final JSON string for unmarshalling")

	var response LLMResponse
	err := json.Unmarshal([]byte(jsonStr), &response)
	if err != nil {
		log.Error().Err(err).Str("json_string", jsonStr).Msg("Failed to unmarshal LLM response JSON")
		return LLMResponse{}, fmt.Errorf("%w: %w", ErrLLMResponseJSONUnmarshal, err) // Use sentinel error
	}
	log.Debug().Interface("parsed_response", response).Msg("Successfully unmarshalled LLM response")

	// Basic validation (remains the same)
	if response.Summary == "" {
		log.Error().Interface("parsed_response", response).Msg("Parsed LLM response is missing 'summary'")
		return response, fmt.Errorf("%w: summary", ErrLLMResponseMissingField) // Use sentinel error
	}
	if response.ProjectNameSuggestion == "" {
		log.Error().Interface("parsed_response", response).Msg("Parsed LLM response is missing 'project_name_suggestion'")
		return response, fmt.Errorf("%w: project_name_suggestion", ErrLLMResponseMissingField) // Use sentinel error
	}

	log.Info().Msg("LLM response parsed and validated successfully")
	return response, nil
}

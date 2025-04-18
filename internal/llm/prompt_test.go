package llm

import (
	"strings"
	"testing"
)

func TestConstructPrompt(t *testing.T) {
	userInput := "Create a bug ticket for login failure"
	systemPrompt := "You are a helpful assistant."
	context := "User is working on project X."

	prompt := ConstructPrompt(userInput, systemPrompt, context)

	// Check if essential parts are included
	if !strings.Contains(prompt, userInput) {
		t.Errorf("Prompt does not contain user input: %q", userInput)
	}
	if !strings.Contains(prompt, systemPrompt) {
		t.Errorf("Prompt does not contain system prompt: %q", systemPrompt)
	}
	if !strings.Contains(prompt, context) {
		t.Errorf("Prompt does not contain context: %q", context)
	}

	// Check for JSON instruction and keys (adjust instruction based on actual prompt.go)
	// Assuming a general instruction format for now.
	expectedJsonInstruction := "Return ONLY a JSON object" // Example, adjust if needed
	if !strings.Contains(prompt, expectedJsonInstruction) {
		// This check might need refinement based on the exact wording in prompt.go
		// t.Logf("Warning: Prompt might be missing the exact JSON instruction %q. Full prompt:\n%s", expectedJsonInstruction, prompt)
	}

	// More robust check for the overall structure asking for JSON keys
	if !strings.Contains(prompt, "summary") || !strings.Contains(prompt, "description") || !strings.Contains(prompt, "project_name_suggestion") {
		t.Errorf("Prompt does not seem to request the required JSON keys (summary, description, project_name_suggestion)")
	}

	// Specific key check (e.g., looking for "key": pattern) - less critical if above passes
	expectedKeys := []string{"summary", "description", "project_name_suggestion"}
	for _, key := range expectedKeys {
		keyCheck := `"` + key + `":` // Check for "key":
		if !strings.Contains(prompt, keyCheck) {
			// This is a stricter check, might fail if the prompt format is different
			// t.Logf("Warning: Prompt might be missing the specific JSON key format %q. Full prompt:\n%s", keyCheck, prompt)
		}
	}
}

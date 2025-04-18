package llm

import (
	"strings"
)

// ConstructPrompt builds the final prompt string to be sent to the LLM.
// It combines the base system instructions (systemPrompt), optional contextual information
// (context, typically from context.md), and the user's specific request (userInput).
// It explicitly instructs the LLM to format its response as a JSON object containing
// "summary", "description", and "project_name_suggestion" fields.
func ConstructPrompt(userInput string, systemPrompt string, context string) string {
	// Use a strings.Builder for efficient string concatenation
	var promptBuilder strings.Builder

	// 1. Add the System Prompt (Instructions for the LLM)
	promptBuilder.WriteString(systemPrompt)
	promptBuilder.WriteString("\n\n") // Add separation

	// 2. Add the Context (Information from context.md)
	if context != "" {
		promptBuilder.WriteString("Relevant Context:\n")
		promptBuilder.WriteString(context)
		promptBuilder.WriteString("\n\n") // Add separation
	}

	// 3. Add the User's Request
	promptBuilder.WriteString("User Request:\n")
	promptBuilder.WriteString(userInput)
	promptBuilder.WriteString("\n\n") // Add separation

	// 4. Add Explicit JSON Output Instructions
	promptBuilder.WriteString("Based on the user request and context, generate a response in the following JSON format ONLY:\n")
	promptBuilder.WriteString("{\n")
	promptBuilder.WriteString("  \"summary\": \"<A concise summary of the ticket/task>\",\n")
	promptBuilder.WriteString("  \"description\": \"<A detailed description of the ticket/task>\",\n")
	promptBuilder.WriteString("  \"project_name_suggestion\": \"<A suggested project name based on the request>\"\n")
	promptBuilder.WriteString("}\n")
	promptBuilder.WriteString("Ensure the output is a single, valid JSON object and nothing else.")

	return promptBuilder.String()
}

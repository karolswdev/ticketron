package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEnsureConfigDir verifies the EnsureConfigDir function.
// TestLoadConfig verifies the LoadConfig function.
// TestLoadLinks verifies the LoadLinks function.
// TestLoadSystemPrompt verifies the LoadSystemPrompt function.
// TestLoadContext verifies the LoadContext function.
// TestCreateDefaultConfigFiles verifies the CreateDefaultConfigFiles function.

func TestEnsureConfigDir(t *testing.T) {
	t.Run("DirectoryDoesNotExist", func(t *testing.T) {
		tempDir := t.TempDir() // Use temp dir for hermetic test

		// Call EnsureConfigDir with the temp directory
		returnedDir, err := EnsureConfigDir(tempDir)
		require.NoError(t, err, "EnsureConfigDir should not return an error when creating the directory")
		require.DirExists(t, tempDir, "Base directory should be created") // Ensure the base dir exists
		require.Equal(t, tempDir, returnedDir, "EnsureConfigDir should return the provided base directory path")
	})

	t.Run("DirectoryAlreadyExists", func(t *testing.T) {
		tempDir := t.TempDir() // Use temp dir for hermetic test

		// Create the directory beforehand
		err := os.MkdirAll(tempDir, 0755) // Create the base temp dir
		require.NoError(t, err, "Failed to pre-create directory for test")

		// Call EnsureConfigDir with the existing temp directory
		returnedDir, err := EnsureConfigDir(tempDir)
		require.NoError(t, err, "EnsureConfigDir should not return an error if the directory already exists")
		require.DirExists(t, tempDir, "Base directory should still exist") // Ensure the base dir still exists
		require.Equal(t, tempDir, returnedDir, "EnsureConfigDir should return the provided base directory path")
	})
}

func TestLoadConfig(t *testing.T) {
	t.Run("ValidConfig", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")
		// Ensure precise YAML indentation (2 spaces for provider/openai, 4 for model_name)
		validYAML := `
# Use mapstructure tags from config.go (mcp_server_url, provider, openai.model_name)
mcp_server_url: "http://localhost:8081"
llm:
  provider: "openai"
  openai:
    model_name: "gpt-4o"
`
		err := os.WriteFile(configPath, []byte(validYAML), 0644)
		require.NoError(t, err, "Failed to write valid config file for test")

		// Load config from the temp directory
		cfg, err := LoadConfig(tempDir)
		require.NoError(t, err, "LoadConfig should not return an error for a valid config")
		require.NotNil(t, cfg, "Config object should not be nil")
		// Assert against values from the temp config file
		assert.Equal(t, "openai", cfg.LLM.Provider, "Should load provider from temp file")
		assert.Equal(t, "gpt-4o", cfg.LLM.OpenAI.ModelName, "Should load OpenAI model name from temp file")
		assert.Equal(t, "http://localhost:8081", cfg.MCPServerURL, "Should load MCP server URL from temp file")
	})

	t.Run("FileNotFound", func(t *testing.T) {
		tempDir := t.TempDir()          // Need temp dir to specify non-existent path
		cfg, err := LoadConfig(tempDir) // Load from empty temp dir
		// LoadConfig should return defaults, not an error, when the file is not found
		require.NoError(t, err, "LoadConfig should not return an error when the config file does not exist")
		require.NotNil(t, cfg, "Config object should not be nil even if file not found")
		assert.Equal(t, "http://localhost:8080", cfg.MCPServerURL, "Should return default MCP URL") // Check default using string literal
		assert.Equal(t, "openai", cfg.LLM.Provider, "Should return default LLM provider")           // Check default provider
		assert.Equal(t, "gpt-4o", cfg.LLM.OpenAI.ModelName, "Should return default OpenAI model")   // Check default model
	})

	t.Run("InvalidYAML", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")
		invalidYAML := `llm: provider: "openai"` // Malformed YAML
		err := os.WriteFile(configPath, []byte(invalidYAML), 0644)
		require.NoError(t, err, "Failed to write invalid config file for test")

		// Load config from the temp directory containing invalid YAML
		_, err = LoadConfig(tempDir)
		// LoadConfig should return an error when the YAML is invalid in the specified baseDir
		require.Error(t, err, "LoadConfig should return an error for invalid YAML")
	})
}

func TestLoadLinks(t *testing.T) {
	t.Run("ValidLinks", func(t *testing.T) {
		tempDir := t.TempDir()
		linksPath := filepath.Join(tempDir, "links.yaml")
		// Corrected YAML indentation
		validYAML := `
# Use list format for projects as defined in LinksConfig struct
projects:
  - name: "Project One"
    key: "PROJ1" # Key is needed for matching, assuming it's still relevant
    default_issue_type: "Bug"
  - name: "Project Two"
    key: "PROJ2" # Key is needed for matching
`
		err := os.WriteFile(linksPath, []byte(validYAML), 0644)
		require.NoError(t, err, "Failed to write valid links file for test")

		// Load links from the temp directory
		linksCfg, err := LoadLinks(tempDir)
		require.NoError(t, err, "LoadLinks should not return an error for valid links")
		require.NotNil(t, linksCfg, "LinksConfig object should not be nil")
		// Assert against values from the temp links file
		require.Len(t, linksCfg.Projects, 2, "Should load 2 projects from temp file")
		assert.Equal(t, "Project One", linksCfg.Projects[0].Name)
		assert.Equal(t, "PROJ1", linksCfg.Projects[0].Key)
		assert.Equal(t, "Bug", linksCfg.Projects[0].DefaultIssueType)
		assert.Equal(t, "Project Two", linksCfg.Projects[1].Name)
		assert.Equal(t, "PROJ2", linksCfg.Projects[1].Key)
		assert.Equal(t, "", linksCfg.Projects[1].DefaultIssueType) // Check empty default
	})

	t.Run("FileNotFound", func(t *testing.T) {
		tempDir := t.TempDir()         // Need temp dir to specify non-existent path
		cfg, err := LoadLinks(tempDir) // Load from empty temp dir
		// LoadLinks should return an empty config, not an error, when the file is not found
		require.NoError(t, err, "LoadLinks should not return an error when the links file does not exist")
		require.NotNil(t, cfg, "LinksConfig object should not be nil even if file not found")
		assert.Empty(t, cfg.Projects, "Projects list should be empty") // Check empty
	})

	t.Run("InvalidYAML", func(t *testing.T) {
		tempDir := t.TempDir()
		linksPath := filepath.Join(tempDir, "links.yaml")
		invalidYAML := `projects: - PROJ1` // Malformed YAML
		err := os.WriteFile(linksPath, []byte(invalidYAML), 0644)
		require.NoError(t, err, "Failed to write invalid links file for test")

		// Load links from the temp directory containing invalid YAML
		_, err = LoadLinks(tempDir)
		// LoadLinks should return an error when the YAML is invalid in the specified baseDir
		require.Error(t, err, "LoadLinks should return an error for invalid YAML")
	})
}

func TestLoadSystemPrompt(t *testing.T) {
	t.Run("ValidPrompt", func(t *testing.T) {
		tempDir := t.TempDir()
		promptPath := filepath.Join(tempDir, "system_prompt.txt")
		promptContent := "This is the system prompt."
		err := os.WriteFile(promptPath, []byte(promptContent), 0644)
		require.NoError(t, err, "Failed to write valid prompt file for test")

		// Load prompt from the temp directory
		prompt, err := LoadSystemPrompt(tempDir)
		require.NoError(t, err, "LoadSystemPrompt should not return an error for a valid prompt file")
		// Assert against the exact content from the temp prompt file
		assert.Equal(t, promptContent, prompt, "Should load the exact prompt content from the temp file")
	})

	t.Run("FileNotFound", func(t *testing.T) {
		tempDir := t.TempDir()                   // Need temp dir to specify non-existent path
		prompt, err := LoadSystemPrompt(tempDir) // Load from empty temp dir
		// LoadSystemPrompt should return an empty string, not an error, when the file is not found
		require.NoError(t, err, "LoadSystemPrompt should not return an error when the prompt file does not exist")
		assert.Empty(t, prompt, "Prompt should be empty when file not found") // Check empty
	})
}

func TestLoadContext(t *testing.T) {
	t.Run("ValidContext", func(t *testing.T) {
		tempDir := t.TempDir()
		contextPath := filepath.Join(tempDir, "context.md")
		contextContent := "# Project Context\n\nDetails about the project."
		err := os.WriteFile(contextPath, []byte(contextContent), 0644)
		require.NoError(t, err, "Failed to write valid context file for test")

		// Load context from the temp directory
		context, err := LoadContext(tempDir)
		require.NoError(t, err, "LoadContext should not return an error for a valid context file")
		// Assert against the exact content from the temp context file
		assert.Equal(t, contextContent, context, "Should load the exact context content from the temp file")
	})

	t.Run("FileNotFound", func(t *testing.T) {
		tempDir := t.TempDir()               // Need temp dir to specify non-existent path
		context, err := LoadContext(tempDir) // Load from empty temp dir
		// LoadContext should return an empty string, not an error, when the file is not found
		require.NoError(t, err, "LoadContext should not return an error when the context file does not exist")
		assert.Empty(t, context, "Context should be empty when file not found") // Check empty
	})
}

func TestCreateDefaultConfigFiles(t *testing.T) {
	t.Run("CreateDefaults", func(t *testing.T) {
		tempDir := t.TempDir() // Use temp dir for hermetic test

		// Create default files in the temp directory
		err := CreateDefaultConfigFiles(tempDir)
		require.NoError(t, err, "CreateDefaultConfigFiles should not return an error")

		// Check if all expected files exist in the temp directory
		require.FileExists(t, filepath.Join(tempDir, "config.yaml"), "Config file should be created")
		require.FileExists(t, filepath.Join(tempDir, "links.yaml"), "Links file should be created")
		require.FileExists(t, filepath.Join(tempDir, "system_prompt.txt"), "System prompt file should be created")
		require.FileExists(t, filepath.Join(tempDir, "context.md"), "Context file should be created")
	})

	// Optional: Add a test case where files already exist (should not overwrite)
	t.Run("FilesAlreadyExist", func(t *testing.T) {
		tempDir := t.TempDir()

		// Pre-create one of the files with specific content
		configPath := filepath.Join(tempDir, "config.yaml") // Use literal filename
		initialContent := "llm:\n  provider: 'dummy'"
		err := os.WriteFile(configPath, []byte(initialContent), 0644)
		require.NoError(t, err)

		// Attempt to create default files again in the same temp directory
		err = CreateDefaultConfigFiles(tempDir)
		require.NoError(t, err, "CreateDefaultConfigFiles should not return an error even if files exist")

		// Verify the pre-existing file was NOT overwritten
		currentContentBytes, err := os.ReadFile(configPath)
		require.NoError(t, err)
		assert.Equal(t, initialContent, string(currentContentBytes), "Existing config file should not be overwritten")

		// Verify other files were still created (or kept if they also existed)
		require.FileExists(t, filepath.Join(tempDir, "links.yaml"), "Links file should exist")
		require.FileExists(t, filepath.Join(tempDir, "system_prompt.txt"), "System prompt file should exist")
		require.FileExists(t, filepath.Join(tempDir, "context.md"), "Context file should exist")
	})
}

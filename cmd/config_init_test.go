package cmd

import (
	"bytes"
	"errors"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestConfigInitCmd_Success(t *testing.T) {
	mockProvider := new(MockConfigProvider)
	var out bytes.Buffer

	// Setup mock expectation for CreateDefaultConfigFiles
	mockProvider.On("CreateDefaultConfigFiles", mock.AnythingOfType("string")).Return(nil)

	// Prepare command
	cmd := &cobra.Command{}
	// Set the output writer for the command if configInitRunE uses it directly
	// cmd.SetOut(&out) // Not needed if we pass the buffer directly to configInitRunE

	// Call configInitRunE directly
	err := configInitRunE(mockProvider, &out, cmd, []string{})

	// Assertions
	assert.NoError(t, err)
	assert.Contains(t, out.String(), "Configuration directory and default files ensured.") // Check for actual success message
	mockProvider.AssertExpectations(t)
	mockProvider.AssertCalled(t, "CreateDefaultConfigFiles", mock.AnythingOfType("string"))
}

func TestConfigInitCmd_ProviderError(t *testing.T) {
	mockProvider := new(MockConfigProvider)
	var out bytes.Buffer

	// Setup mock expectation for CreateDefaultConfigFiles to return an error
	expectedErr := errors.New("failed to create config dir")
	mockProvider.On("CreateDefaultConfigFiles", mock.AnythingOfType("string")).Return(expectedErr)

	// Prepare command
	cmd := &cobra.Command{}
	// cmd.SetOut(&out)

	// Call configInitRunE directly
	err := configInitRunE(mockProvider, &out, cmd, []string{})

	// Assertions
	assert.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)                                   // Check if the specific error is returned/wrapped
	assert.Contains(t, err.Error(), "failed to initialize configuration") // Check for wrapped error message
	assert.Empty(t, out.String())                                         // No success message expected
	mockProvider.AssertExpectations(t)
	mockProvider.AssertCalled(t, "CreateDefaultConfigFiles", mock.AnythingOfType("string"))
}

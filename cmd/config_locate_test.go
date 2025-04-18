package cmd

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigLocateCmd_Success(t *testing.T) {
	mockProvider := new(MockConfigProvider)
	var out bytes.Buffer

	// Setup mock expectation for EnsureConfigDir
	expectedPath := "/home/user/.config/ticketron"
	mockProvider.On("EnsureConfigDir").Return(expectedPath, nil)

	// Prepare command
	// cmd := &cobra.Command{} // Not needed as we call configLocateRunE directly
	// cmd.SetOut(&out) // Output is written directly by configLocateRun

	// Call configLocateRun directly
	err := configLocateRunE(mockProvider, &out) // Correct function name

	// Assertions
	assert.NoError(t, err)
	assert.Contains(t, out.String(), "Configuration directory:")
	assert.Contains(t, out.String(), expectedPath)
	mockProvider.AssertExpectations(t)
	mockProvider.AssertCalled(t, "EnsureConfigDir")
}

func TestConfigLocateCmd_ProviderError(t *testing.T) {
	mockProvider := new(MockConfigProvider)
	var out bytes.Buffer

	// Setup mock expectation for EnsureConfigDir to return an error
	expectedErr := errors.New("cannot create config dir")
	mockProvider.On("EnsureConfigDir").Return("", expectedErr)

	// Prepare command
	// cmd := &cobra.Command{} // Not needed
	// cmd.SetOut(&out)

	// Call configLocateRun directly
	err := configLocateRunE(mockProvider, &out) // Correct function name

	// Assertions
	assert.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)                                 // Check if the specific error is returned/wrapped
	assert.Contains(t, err.Error(), "error ensuring config directory:") // Check for actual wrapped error message
	assert.Empty(t, out.String())                                       // No success message expected
	mockProvider.AssertExpectations(t)
	mockProvider.AssertCalled(t, "EnsureConfigDir")
}

package cmd

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestConfigSetKeyCmd_Success(t *testing.T) {
	mockKeyring := new(MockKeyringClient)
	var out bytes.Buffer // This will be used now

	// Setup mock expectation for Set
	// Use constants defined in config_set_key.go
	service := keyringServiceName
	user := keyringUserName // Correct user name
	apiKey := "test-api-key-123"
	mockKeyring.On("Set", service, user, apiKey).Return(nil)

	// Prepare command
	// Call configSetKeyRun directly, passing the buffer
	err := configSetKeyRun(mockKeyring, &out, apiKey)

	// Assertions
	assert.NoError(t, err)
	// Check the output written to the buffer
	assert.Contains(t, out.String(), "API key stored successfully.")
	mockKeyring.AssertExpectations(t)
	mockKeyring.AssertCalled(t, "Set", service, user, apiKey)
}

func TestConfigSetKeyCmd_KeyringError(t *testing.T) {
	mockKeyring := new(MockKeyringClient)
	var out bytes.Buffer

	// Setup mock expectation for Set to return an error
	service := keyringServiceName
	user := keyringUserName
	apiKey := "test-api-key-123"
	expectedErr := errors.New("failed to access keyring")
	mockKeyring.On("Set", service, user, apiKey).Return(expectedErr)

	// Call configSetKeyRun directly
	err := configSetKeyRun(mockKeyring, &out, apiKey)

	// Assertions
	assert.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)                        // Check if the specific error is returned/wrapped
	assert.Contains(t, err.Error(), "failed to store API key") // Check for wrapped error message
	assert.Empty(t, out.String())                              // No success message expected
	mockKeyring.AssertExpectations(t)
	mockKeyring.AssertCalled(t, "Set", service, user, apiKey)
}

func TestConfigSetKeyCmd_NoArgs(t *testing.T) {
	mockKeyring := new(MockKeyringClient)
	var out bytes.Buffer
	// Call configSetKeyRun directly with an empty string for the key
	// as the cobra arg validation happens before RunE/configSetKeyRun
	err := configSetKeyRun(mockKeyring, &out, "")

	// Assertions
	assert.Error(t, err)
	// Check for the specific error returned by configSetKeyRunE for missing args
	// Check for the specific error returned by configSetKeyRun for empty key
	assert.EqualError(t, err, "API key cannot be empty")
	assert.Empty(t, out.String())
	mockKeyring.AssertNotCalled(t, "Set", mock.Anything, mock.Anything, mock.Anything)
}

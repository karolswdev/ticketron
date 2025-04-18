package mcpclient

import "errors"

// Sentinel errors for MCP client operations.

// ErrMCPServerURLMissing indicates the MCP server URL is not configured.
var ErrMCPServerURLMissing = errors.New("MCP server URL is not configured")

// ErrMCPServerURLParse indicates an error occurred while parsing the MCP server URL.
var ErrMCPServerURLParse = errors.New("failed to parse MCP server URL")

// ErrRequestMarshal indicates an error occurred while marshaling the request body.
var ErrRequestMarshal = errors.New("failed to marshal request body")

// ErrRequestCreate indicates an error occurred while creating the HTTP request.
var ErrRequestCreate = errors.New("failed to create HTTP request")

// ErrRequestExecute indicates an error occurred while executing the HTTP request.
var ErrRequestExecute = errors.New("failed to execute HTTP request")

// ErrResponseDecode indicates an error occurred while decoding the response body.
var ErrResponseDecode = errors.New("failed to decode response body")

// ErrMCPServerError indicates the MCP server returned a non-2xx status code with a specific error message.
// The actual error message from the server should be wrapped.
var ErrMCPServerError = errors.New("MCP server returned an error")

// ErrMCPServerErrorUnparseable indicates the MCP server returned a non-2xx status code,
// but the error response body could not be parsed or was empty.
var ErrMCPServerErrorUnparseable = errors.New("MCP server returned an unparseable error")

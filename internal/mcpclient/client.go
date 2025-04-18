package mcpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/karolswdev/ticketron/internal/config"
)

// Client provides methods for interacting with the Jira MCP (Model Context Protocol) server.
// It handles constructing requests, sending them to the configured MCP server URL,
// and decoding the responses.
type Client struct {
	BaseURL    *url.URL
	HTTPClient *http.Client
}

// New creates and initializes a new MCP Client instance based on the provided AppConfig.
// It parses the MCPServerURL from the config and sets up a default HTTP client
// with a timeout. It returns an error if the URL is missing or invalid.
func New(cfg *config.AppConfig) (*Client, error) {
	if cfg.MCPServerURL == "" {
		return nil, ErrMCPServerURLMissing // Use sentinel error
	}
	baseURL, err := url.Parse(cfg.MCPServerURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrMCPServerURLParse, err) // Use sentinel error
	}

	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: time.Second * 10, // Default timeout
		},
	}, nil
}

// CreateIssue sends a POST request to the MCP server's /create_jira_issue endpoint
// to create a new Jira issue based on the provided CreateIssueRequest data.
// It handles JSON marshalling of the request, sending the HTTP request,
// and decoding the JSON response (either CreateIssueResponse on success or ErrorResponse on failure).
// It returns the CreateIssueResponse or an error if the request or decoding fails,
// or if the server returns a non-201 status code.
func (c *Client) CreateIssue(ctx context.Context, reqBody CreateIssueRequest) (*CreateIssueResponse, error) {
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrRequestMarshal, err) // Use sentinel error
	}

	// Construct the full URL for the endpoint
	endpointURL := c.BaseURL.ResolveReference(&url.URL{Path: "/create_jira_issue"})

	log.Debug().RawJSON("request_body", jsonData).Str("url", endpointURL.String()).Msg("Sending MCP CreateIssue request") // Added Debug log
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpointURL.String(), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrRequestCreate, err) // Use sentinel error
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrRequestExecute, err) // Use sentinel error
	}
	defer resp.Body.Close()

	// Read the body first for logging, then check status code
	respBodyBytes, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		log.Warn().Err(readErr).Msg("Failed to read MCP response body for logging")
		// Continue processing status code, but body might be lost
	} else {
		// Reset body for subsequent decoders
		resp.Body = io.NopCloser(bytes.NewBuffer(respBodyBytes))
		log.Debug().Int("status_code", resp.StatusCode).RawJSON("response_body", respBodyBytes).Msg("Received MCP CreateIssue response") // Added Debug log
	}

	if resp.StatusCode != http.StatusCreated {
		// Attempt to decode the known error structure first
		var errResp ErrorResponse
		// Attempt to decode the known error structure first
		if decodeErr := json.NewDecoder(resp.Body).Decode(&errResp); decodeErr == nil && errResp.Error != "" {
			// Wrap the specific server message with our sentinel error
			return nil, fmt.Errorf("%w: %s (status %d)", ErrMCPServerError, errResp.Error, resp.StatusCode)
		}
		// If decoding fails or the error message is empty, return the unparseable error sentinel
		return nil, fmt.Errorf("%w (status %d)", ErrMCPServerErrorUnparseable, resp.StatusCode)
	}

	var successResp CreateIssueResponse
	if err := json.NewDecoder(resp.Body).Decode(&successResp); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrResponseDecode, err) // Use sentinel error
	}

	return &successResp, nil
}

// SearchIssues sends a POST request to the MCP server's /search_jira_issues endpoint
// to search for Jira issues based on the provided SearchIssuesRequest data (e.g., JQL).
// It handles JSON marshalling of the request, sending the HTTP request,
// and decoding the JSON response (either SearchIssuesResponse on success or ErrorResponse on failure).
// It returns the SearchIssuesResponse or an error if the request or decoding fails,
// or if the server returns a non-200 status code.
func (c *Client) SearchIssues(ctx context.Context, reqBody SearchIssuesRequest) (*SearchIssuesResponse, error) {
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrRequestMarshal, err) // Use sentinel error
	}

	// Construct the full URL for the endpoint
	endpointURL := c.BaseURL.ResolveReference(&url.URL{Path: "/search_jira_issues"})

	log.Debug().RawJSON("request_body", jsonData).Str("url", endpointURL.String()).Msg("Sending MCP SearchIssues request") // Added Debug log
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpointURL.String(), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrRequestCreate, err) // Use sentinel error
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrRequestExecute, err) // Use sentinel error
	}
	defer resp.Body.Close()

	// Read the body first for logging, then check status code
	respBodyBytes, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		log.Warn().Err(readErr).Msg("Failed to read MCP response body for logging")
		// Continue processing status code, but body might be lost
	} else {
		// Reset body for subsequent decoders
		resp.Body = io.NopCloser(bytes.NewBuffer(respBodyBytes))
		log.Debug().Int("status_code", resp.StatusCode).RawJSON("response_body", respBodyBytes).Msg("Received MCP SearchIssues response") // Added Debug log
	}

	if resp.StatusCode != http.StatusOK { // Expecting 200 OK for search
		// Attempt to decode the known error structure first
		var errResp ErrorResponse
		// Attempt to decode the known error structure first
		if decodeErr := json.NewDecoder(resp.Body).Decode(&errResp); decodeErr == nil && errResp.Error != "" {
			// Wrap the specific server message with our sentinel error
			return nil, fmt.Errorf("%w: %s (status %d)", ErrMCPServerError, errResp.Error, resp.StatusCode)
		}
		// If decoding fails or the error message is empty, return the unparseable error sentinel
		return nil, fmt.Errorf("%w (status %d)", ErrMCPServerErrorUnparseable, resp.StatusCode)
	}

	var successResp SearchIssuesResponse
	if err := json.NewDecoder(resp.Body).Decode(&successResp); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrResponseDecode, err) // Use sentinel error
	}

	return &successResp, nil
}

// GetIssue sends a GET request to the MCP server's /jira_issue/{issueKey} endpoint
// to retrieve details for a specific Jira issue identified by its key.
// It handles constructing the URL, sending the HTTP request, and decoding the JSON response
// (either Issue on success or ErrorResponse on failure).
// It returns the Issue details or an error if the request or decoding fails,
// or if the server returns a non-200 status code.
func (c *Client) GetIssue(ctx context.Context, issueKey string) (*Issue, error) {
	// Construct the relative path with the issue key
	relativePath := fmt.Sprintf("/jira_issue/%s", issueKey)

	// Construct the full URL for the endpoint
	endpointURL := c.BaseURL.ResolveReference(&url.URL{Path: relativePath})

	log.Debug().Str("url", endpointURL.String()).Msg("Sending MCP GetIssue request")       // Added Debug log
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpointURL.String(), nil) // No body for GET
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrRequestCreate, err) // Use sentinel error
	}

	req.Header.Set("Accept", "application/json") // Expect JSON response

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrRequestExecute, err) // Use sentinel error
	}
	defer resp.Body.Close()

	// Read the body first for logging, then check status code
	respBodyBytes, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		log.Warn().Err(readErr).Msg("Failed to read MCP response body for logging")
		// Continue processing status code, but body might be lost
	} else {
		// Reset body for subsequent decoders
		resp.Body = io.NopCloser(bytes.NewBuffer(respBodyBytes))
		log.Debug().Int("status_code", resp.StatusCode).RawJSON("response_body", respBodyBytes).Msg("Received MCP GetIssue response") // Added Debug log
	}

	if resp.StatusCode != http.StatusOK { // Expecting 200 OK for get
		// Attempt to decode the known error structure first
		var errResp ErrorResponse
		// Attempt to decode the known error structure first
		if decodeErr := json.NewDecoder(resp.Body).Decode(&errResp); decodeErr == nil && errResp.Error != "" {
			// Wrap the specific server message with our sentinel error
			return nil, fmt.Errorf("%w: %s (status %d)", ErrMCPServerError, errResp.Error, resp.StatusCode)
		}
		// If decoding fails or the error message is empty, return the unparseable error sentinel
		return nil, fmt.Errorf("%w (status %d)", ErrMCPServerErrorUnparseable, resp.StatusCode)
	}

	var issue Issue // Use the Issue struct from types.go
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrResponseDecode, err) // Use sentinel error
	}

	return &issue, nil
}

package mcpclient

import (
	"context"       // Added context import
	"encoding/json" // Added for errors.Is
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/karolswdev/ticketron/internal/config" // Added config import
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a mock server and client
func setupMockServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *Client) {
	server := httptest.NewServer(handler)
	// Create a minimal mock config for the client
	mockCfg := &config.AppConfig{
		MCPServerURL: server.URL,
	}
	client, err := New(mockCfg)
	require.NoError(t, err, "New client should not return an error")
	require.NotNil(t, client, "New client should not be nil")
	return server, client
}

func TestNew(t *testing.T) {
	baseURL := "http://test.example.com"
	mockCfg := &config.AppConfig{MCPServerURL: baseURL}
	client, err := New(mockCfg)
	require.NoError(t, err, "New client should not return an error")
	assert.NotNil(t, client, "Client should not be nil")
	assert.Equal(t, baseURL, client.BaseURL.String(), "BaseURL should match")
	assert.NotNil(t, client.HTTPClient, "HTTPClient should be initialized")
}

func TestCreateIssue(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		expectedReq := CreateIssueRequest{
			ProjectKey:  "PROJ",
			Summary:     "Test Summary",
			Description: "Test Description",
			IssueType:   "Task",
		}
		expectedResp := CreateIssueResponse{
			Key:  "PROJ-123",
			ID:   "10001",
			Self: "http://jira.example.com/rest/api/2/issue/10001",
		}

		handler := func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method, "Expected POST method")
			assert.Equal(t, "/create_jira_issue", r.URL.Path, "Expected correct path")
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"), "Expected correct Content-Type")

			bodyBytes, err := io.ReadAll(r.Body)
			require.NoError(t, err, "Failed to read request body")
			defer r.Body.Close()

			var actualReq CreateIssueRequest
			err = json.Unmarshal(bodyBytes, &actualReq)
			require.NoError(t, err, "Failed to unmarshal request body")
			assert.Equal(t, expectedReq, actualReq, "Request body mismatch")

			w.WriteHeader(http.StatusCreated)
			w.Header().Set("Content-Type", "application/json")
			err = json.NewEncoder(w).Encode(expectedResp)
			require.NoError(t, err, "Failed to encode response")
		}

		server, client := setupMockServer(t, handler)
		defer server.Close()

		ctx := context.Background()
		resp, err := client.CreateIssue(ctx, expectedReq)
		require.NoError(t, err, "CreateIssue should not return an error")
		assert.Equal(t, expectedResp, *resp, "Response body mismatch")
	})

	t.Run("ClientError", func(t *testing.T) {
		expectedReq := CreateIssueRequest{ProjectKey: "BAD"} // Minimal request for error test
		expectedErrorMsg := "Missing projectKey"

		handler := func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/create_jira_issue", r.URL.Path)

			w.WriteHeader(http.StatusBadRequest)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"error": "%s"}`, expectedErrorMsg)
		}

		server, client := setupMockServer(t, handler)
		defer server.Close()

		ctx := context.Background()
		_, err := client.CreateIssue(ctx, expectedReq)
		require.Error(t, err, "CreateIssue should return an error")
		// Check for the correct sentinel error and the specific message
		assert.ErrorIs(t, err, ErrMCPServerError, "Error should be ErrMCPServerError")
		assert.Contains(t, err.Error(), expectedErrorMsg, "Error message should contain server error")
		assert.Contains(t, err.Error(), "(status 400)", "Error message should contain status code")
	})

	t.Run("ServerError", func(t *testing.T) {
		expectedReq := CreateIssueRequest{ProjectKey: "FAIL"}
		expectedErrorMsg := "Internal server problem"

		handler := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"error": "%s"}`, expectedErrorMsg)
		}

		server, client := setupMockServer(t, handler)
		defer server.Close()

		ctx := context.Background()
		_, err := client.CreateIssue(ctx, expectedReq)
		require.Error(t, err, "CreateIssue should return an error")
		// Check for the correct sentinel error and the specific message
		assert.ErrorIs(t, err, ErrMCPServerError, "Error should be ErrMCPServerError")
		assert.Contains(t, err.Error(), expectedErrorMsg, "Error message should contain server error")
		assert.Contains(t, err.Error(), "(status 500)", "Error message should contain status code")
	})

	t.Run("NonJsonError", func(t *testing.T) {
		expectedReq := CreateIssueRequest{ProjectKey: "GATEWAY"}

		handler := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprint(w, "Bad Gateway")
		}

		server, client := setupMockServer(t, handler)
		defer server.Close()

		ctx := context.Background() // Add context
		_, err := client.CreateIssue(ctx, expectedReq)
		require.Error(t, err, "CreateIssue should return an error")
		// Check for the correct sentinel error for unparseable responses
		assert.ErrorIs(t, err, ErrMCPServerErrorUnparseable, "Error should be ErrMCPServerErrorUnparseable")
		assert.Contains(t, err.Error(), "(status 502)", "Error message should contain status code")
	})
}

func TestSearchIssues(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		expectedReq := SearchIssuesRequest{
			JQL:        "project = PROJ",
			MaxResults: 10,
			// Fields removed
		}
		// Use the Issue struct from types.go
		expectedResp := SearchIssuesResponse{
			StartAt:    0,
			MaxResults: 10,
			Total:      2,
			Issues: []Issue{
				{Key: "PROJ-1", ID: "10001", Self: "...", Fields: IssueFields{Summary: "Issue 1", Status: Status{Name: "Open"}}},
				{Key: "PROJ-2", ID: "10002", Self: "...", Fields: IssueFields{Summary: "Issue 2", Status: Status{Name: "Closed"}}},
			},
		}

		handler := func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/search_jira_issues", r.URL.Path)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			bodyBytes, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			defer r.Body.Close()

			var actualReq SearchIssuesRequest
			err = json.Unmarshal(bodyBytes, &actualReq)
			require.NoError(t, err)
			assert.Equal(t, expectedReq, actualReq)

			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			err = json.NewEncoder(w).Encode(expectedResp)
			require.NoError(t, err)
		}

		server, client := setupMockServer(t, handler)
		defer server.Close()

		ctx := context.Background()
		resp, err := client.SearchIssues(ctx, expectedReq)
		require.NoError(t, err)
		assert.Equal(t, expectedResp, *resp)
	})

	t.Run("ClientError", func(t *testing.T) {
		expectedReq := SearchIssuesRequest{JQL: "bad jql"}
		expectedErrorMsg := "Invalid JQL"

		handler := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"error": "%s"}`, expectedErrorMsg)
		}

		server, client := setupMockServer(t, handler)
		defer server.Close()

		ctx := context.Background()
		_, err := client.SearchIssues(ctx, expectedReq)
		require.Error(t, err)
		// Check for the correct sentinel error and the specific message
		assert.ErrorIs(t, err, ErrMCPServerError, "Error should be ErrMCPServerError")
		assert.Contains(t, err.Error(), expectedErrorMsg, "Error message should contain server error")
		assert.Contains(t, err.Error(), "(status 400)", "Error message should contain status code")
	})

	t.Run("ServerError", func(t *testing.T) {
		expectedReq := SearchIssuesRequest{JQL: "fail jql"}
		expectedErrorMsg := "Search failed"

		handler := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"error": "%s"}`, expectedErrorMsg)
		}

		server, client := setupMockServer(t, handler)
		defer server.Close()

		ctx := context.Background()
		_, err := client.SearchIssues(ctx, expectedReq)
		require.Error(t, err)
		// Check for the correct sentinel error and the specific message
		assert.ErrorIs(t, err, ErrMCPServerError, "Error should be ErrMCPServerError")
		assert.Contains(t, err.Error(), expectedErrorMsg, "Error message should contain server error")
		assert.Contains(t, err.Error(), "(status 500)", "Error message should contain status code")
	})
}

func TestGetIssue(t *testing.T) {
	issueKey := "PROJ-456"

	t.Run("Success", func(t *testing.T) {
		// Use the Issue struct from types.go
		expectedResp := Issue{
			Key:  issueKey,
			ID:   "10003",
			Self: "...",
			Fields: IssueFields{
				Summary:     "Detailed Issue",
				Description: "More details here",
				Status:      Status{Name: "In Progress"},
				IssueType:   IssueType{Name: "Task"},
			},
		}

		handler := func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			expectedPath := fmt.Sprintf("/jira_issue/%s", issueKey)
			assert.Equal(t, expectedPath, r.URL.Path)

			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			err := json.NewEncoder(w).Encode(expectedResp)
			require.NoError(t, err)
		}

		server, client := setupMockServer(t, handler)
		defer server.Close()

		ctx := context.Background()
		resp, err := client.GetIssue(ctx, issueKey)
		require.NoError(t, err)
		assert.Equal(t, expectedResp, *resp)
	})

	t.Run("NotFound", func(t *testing.T) {
		expectedErrorMsg := "Issue not found"

		handler := func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			expectedPath := fmt.Sprintf("/jira_issue/%s", issueKey)
			assert.Equal(t, expectedPath, r.URL.Path)

			w.WriteHeader(http.StatusNotFound)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"error": "%s"}`, expectedErrorMsg)
		}

		server, client := setupMockServer(t, handler)
		defer server.Close()

		ctx := context.Background()
		_, err := client.GetIssue(ctx, issueKey)
		require.Error(t, err)
		// Check for the correct sentinel error and the specific message
		assert.ErrorIs(t, err, ErrMCPServerError, "Error should be ErrMCPServerError")
		assert.Contains(t, err.Error(), expectedErrorMsg, "Error message should contain server error")
		assert.Contains(t, err.Error(), "(status 404)", "Error message should contain status code")
	})

	t.Run("ServerError", func(t *testing.T) {
		expectedErrorMsg := "Failed to retrieve issue"

		handler := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"error": "%s"}`, expectedErrorMsg)
		}

		server, client := setupMockServer(t, handler)
		defer server.Close()

		ctx := context.Background()
		_, err := client.GetIssue(ctx, issueKey)
		require.Error(t, err)
		// Check for the correct sentinel error and the specific message
		assert.ErrorIs(t, err, ErrMCPServerError, "Error should be ErrMCPServerError")
		assert.Contains(t, err.Error(), expectedErrorMsg, "Error message should contain server error")
		assert.Contains(t, err.Error(), "(status 500)", "Error message should contain status code")
	})
}

// Removed TestParseErrorResponse as error handling is done within the client methods

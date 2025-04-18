package mcpclient

// CreateIssueRequest defines the JSON structure expected by the MCP server's
// /create_jira_issue endpoint. It contains the necessary details to create a new Jira issue.
type CreateIssueRequest struct {
	ProjectKey  string `json:"projectKey"`
	Summary     string `json:"summary"`
	Description string `json:"description"`
	IssueType   string `json:"issueType"`
}

// SearchIssuesRequest defines the JSON structure expected by the MCP server's
// /search_jira_issues endpoint. It contains the JQL query and optional pagination parameters.
type SearchIssuesRequest struct {
	JQL        string `json:"jql"`
	MaxResults int    `json:"maxResults,omitempty"`
	StartAt    int    `json:"startAt,omitempty"`
}

// CreateIssueResponse defines the JSON structure returned by the MCP server's
// /create_jira_issue endpoint upon successful issue creation. It includes the key, ID, and self URL of the new issue.
type CreateIssueResponse struct {
	Key  string `json:"key"`
	ID   string `json:"id"`
	Self string `json:"self"`
}

// SearchIssuesResponse defines the JSON structure returned by the MCP server's
// /search_jira_issues endpoint. It includes pagination details and a slice of matching Issues.
type SearchIssuesResponse struct {
	StartAt    int     `json:"startAt"`
	MaxResults int     `json:"maxResults"`
	Total      int     `json:"total"`
	Issues     []Issue `json:"issues"`
}

// Issue represents a single Jira issue, typically returned as part of a search result
// or when retrieving a specific issue via the /jira_issue/{issueKey} endpoint.
type Issue struct {
	Key    string      `json:"key" yaml:"key"`
	ID     string      `json:"id" yaml:"id"`
	Self   string      `json:"self" yaml:"self"`
	Fields IssueFields `json:"fields" yaml:"fields"`
}

// IssueFields holds the core fields associated with a Jira Issue, such as summary,
// status, issue type, and description.
type IssueFields struct {
	Summary     string    `json:"summary" yaml:"summary"`
	Status      Status    `json:"status" yaml:"status"`
	IssueType   IssueType `json:"issuetype" yaml:"issuetype"`
	Description string    `json:"description,omitempty" yaml:"description,omitempty"` // Added optional description
}

// Status represents the status field of a Jira Issue, containing its name.
type Status struct {
	Name string `json:"name" yaml:"name"`
}

// IssueType represents the issue type field of a Jira Issue, containing its name.
type IssueType struct {
	Name string `json:"name" yaml:"name"`
}

// ErrorResponse defines the standard JSON structure used by the MCP server to return
// error messages when a request fails.
type ErrorResponse struct {
	Error string `json:"error"`
}

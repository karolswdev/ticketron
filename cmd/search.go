package cmd

import (
	"encoding/json" // Added for JSON output
	"errors"
	"fmt"
	"io" // Added for io.Writer
	"reflect"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3" // Added for YAML output

	"github.com/karolswdev/ticketron/internal/config" // Added for config errors
	"github.com/karolswdev/ticketron/internal/mcpclient"
)

// searchRunE holds the logic for the search command, accepting dependencies.
func searchRunE(cfgProvider ConfigProvider, mcpClient MCPClient, out io.Writer, cmd *cobra.Command, args []string) error {
	// Get flag values
	jqlFlag, _ := cmd.Flags().GetString("jql")
	maxResults, _ := cmd.Flags().GetInt("max-results")
	outputFormat, _ := cmd.Flags().GetString("output")
	outputFieldsStr, _ := cmd.Flags().GetString("output-fields") // Get raw flag string

	// Determine JQL query
	var jqlQuery string
	switch {
	case jqlFlag != "":
		jqlQuery = jqlFlag
	case len(args) > 0:
		jqlQuery = strings.Join(args, " ")
	default:
		err := errors.New("no JQL query provided")
		log.Error().Err(err).Msg("JQL query missing")
		fmt.Fprintln(cmd.ErrOrStderr(), "Error: No JQL query provided.")
		fmt.Fprintln(cmd.ErrOrStderr(), "Please provide the query as arguments or use the --jql flag.")
		return err
	}

	// Prepare request
	request := mcpclient.SearchIssuesRequest{
		JQL:        jqlQuery,
		MaxResults: maxResults,
	}

	// Call MCP server
	ctx := cmd.Context()
	resp, err := mcpClient.SearchIssues(ctx, request)
	if err != nil {
		log.Error().Err(err).Msg("Failed to search issues via MCP")
		// User feedback based on error type using switch
		switch {
		case errors.Is(err, mcpclient.ErrRequestExecute):
			fmt.Fprintf(cmd.ErrOrStderr(), "Error connecting to the MCP server: %v\n", err)
			fmt.Fprintln(cmd.ErrOrStderr(), "Please ensure the MCP server is running and the URL is correct.")
		case errors.Is(err, mcpclient.ErrMCPServerError):
			fmt.Fprintf(cmd.ErrOrStderr(), "MCP server returned an error during search: %v\n", err)
		case errors.Is(err, mcpclient.ErrMCPServerErrorUnparseable):
			fmt.Fprintf(cmd.ErrOrStderr(), "MCP server returned an error with an unparseable response body during search: %v\n", err)
		case errors.Is(err, mcpclient.ErrResponseDecode):
			fmt.Fprintf(cmd.ErrOrStderr(), "Failed to decode the search results from the MCP server: %v\n", err)
		default:
			fmt.Fprintf(cmd.ErrOrStderr(), "An unexpected error occurred while searching issues via MCP: %v\n", err)
		}
		return err
	}

	// Parse fields only if the flag string is not empty
	var fields []string
	if outputFieldsStr != "" {
		rawFields := strings.Split(outputFieldsStr, ",")
		validFields := make([]string, 0, len(rawFields))
		for _, field := range rawFields {
			trimmedField := strings.TrimSpace(field)
			if trimmedField != "" {
				validFields = append(validFields, trimmedField)
			}
		}
		// Only assign if parsing resulted in non-empty fields
		if len(validFields) > 0 {
			fields = validFields
		}
	}

	switch outputFormat {
	case "json":
		var outputData interface{}
		// Use parsed fields if available (implies flag was set and valid)
		if len(fields) > 0 {
			filteredIssues := make([]map[string]interface{}, 0, len(resp.Issues))
			for _, issue := range resp.Issues {
				filteredIssues = append(filteredIssues, extractFields(issue, fields))
			}
			outputData = filteredIssues
		} else {
			// Default: output full response object
			outputData = resp
		}
		jsonData, err := json.MarshalIndent(outputData, "", "  ")
		if err != nil {
			log.Error().Err(err).Msg("Failed to marshal search results to JSON")
			fmt.Fprintf(cmd.ErrOrStderr(), "Error formatting search results as JSON: %v\n", err)
			return err
		}
		fmt.Fprintln(out, string(jsonData))

	case "yaml":
		var outputData interface{}
		// Use parsed fields if available (implies flag was set and valid)
		if len(fields) > 0 {
			filteredIssues := make([]map[string]interface{}, 0, len(resp.Issues))
			for _, issue := range resp.Issues {
				filteredIssues = append(filteredIssues, extractFields(issue, fields))
			}
			outputData = filteredIssues // Assign filtered data
		} else {
			// Default: output only the issues list
			outputData = resp.Issues
		}
		yamlData, err := yaml.Marshal(outputData)
		if err != nil {
			log.Error().Err(err).Msg("Failed to marshal search results to YAML")
			fmt.Fprintf(cmd.ErrOrStderr(), "Error formatting search results as YAML: %v\n", err)
			return err
		}
		fmt.Fprintln(out, string(yamlData))

	case "tsv":
		if len(resp.Issues) == 0 {
			log.Info().Msg("No issues found matching the query.")
			fmt.Fprintln(out, "No issues found.")
			return nil
		}
		var tsvFields []string
		// Revert to original default field names (lowercase JSON style)
		defaultTsvFields := []string{"key", "fields.summary", "fields.status.name", "fields.issuetype.name"}

		// Determine fields based *only* on whether the flag string was provided.
		if outputFieldsStr != "" {
			// Flag string was provided. Use parsed 'fields' if valid, otherwise default.
			if len(fields) > 0 {
				tsvFields = fields
			} else {
				// Flag string was provided but invalid (e.g., "-f ,,,"), use default
				log.Warn().Str("flag_value", outputFieldsStr).Msg("Invalid value for --output-fields, using default TSV fields.")
				tsvFields = defaultTsvFields
			}
		} else {
			// Flag string was empty, use default
			tsvFields = defaultTsvFields
		}

		fmt.Fprintln(out, strings.Join(tsvFields, "\t")) // Print header

		for _, issue := range resp.Issues {
			values := make([]string, 0, len(tsvFields)) // Re-initialize slice for each issue
			for _, fieldPath := range tsvFields {
				value, found := getValueByPath(issue, fieldPath)
				var formattedValue string
				if found && value != nil {
					formattedValue = fmt.Sprintf("%v", value)
					// Restore sanitization
					formattedValue = strings.ReplaceAll(formattedValue, "\t", " ")
					formattedValue = strings.ReplaceAll(formattedValue, "\n", " ")
					formattedValue = strings.ReplaceAll(formattedValue, "\r", " ")
				} else {
					formattedValue = ""
				}
				values = append(values, formattedValue)
			}
			fmt.Fprintln(out, strings.Join(values, "\t"))
		}

	default: // Includes "text" or any unspecified format
		if len(resp.Issues) == 0 {
			log.Info().Msg("No issues found matching the query.")
			fmt.Fprintln(out, "No issues found.")
		} else {
			log.Info().Int("count", len(resp.Issues)).Msg("Found issues")
			fmt.Fprintf(out, "Found %d issues:\n", len(resp.Issues))
			for _, issue := range resp.Issues {
				status := issue.Fields.Status.Name
				summary := issue.Fields.Summary
				fmt.Fprintf(out, "- %s - %s - %s\n", issue.Key, status, summary)
			}
		}
	}

	return nil
}

// getValueByPath initiates the recursive traversal to find a value by path.
func getValueByPath(data interface{}, path string) (interface{}, bool) {
	parts := strings.Split(path, ".")
	val, found := getValueRecursive(reflect.ValueOf(data), parts)
	if !found || !val.IsValid() || !val.CanInterface() {
		return nil, false
	}
	return val.Interface(), true
}

// getValueRecursive performs the actual recursive traversal.
func getValueRecursive(current reflect.Value, pathParts []string) (reflect.Value, bool) {
	if len(pathParts) == 0 {
		return current, true
	}
	part := pathParts[0]
	remainingParts := pathParts[1:]

	// Dereference pointers and interfaces
	for current.Kind() == reflect.Ptr || current.Kind() == reflect.Interface {
		if current.IsNil() {
			return reflect.Value{}, false // Cannot traverse nil
		}
		current = current.Elem()
		if !current.IsValid() {
			return reflect.Value{}, false // Invalid element after dereferencing
		}
	}

	switch current.Kind() {
	case reflect.Struct:
		var nextVal reflect.Value
		found := false

		// 1. Try direct field name match (case-insensitive)
		fieldVal := current.FieldByNameFunc(func(name string) bool {
			return strings.EqualFold(name, part)
		})
		if fieldVal.IsValid() {
			if fieldVal.CanInterface() {
				structField, _ := current.Type().FieldByNameFunc(func(name string) bool { return strings.EqualFold(name, part) })
				jsonTag := structField.Tag.Get("json")
				if jsonTag != "-" {
					nextVal = fieldVal
					found = true
				}
			}
		}

		// 2. If not found by name, try JSON tag match
		if !found {
			fmt.Printf("[DEBUG JSON Tag Lookup] Part: '%s', Current Type: %s\n", part, current.Type().String()) // Log entry to JSON tag loop
			for i := 0; i < current.NumField(); i++ {
				structField := current.Type().Field(i)
				jsonTag := structField.Tag.Get("json")
				tagName := strings.Split(jsonTag, ",")[0]
				fmt.Printf("[DEBUG JSON Tag Lookup] Checking Field: '%s', TagName: '%s'\n", structField.Name, tagName) // Log field being checked

				if tagName == part && jsonTag != "-" {
					fieldValByIndex := current.Field(i)
					fmt.Printf("[DEBUG JSON Tag Lookup] Match found! Field: '%s', IsValid: %v, CanInterface: %v\n", structField.Name, fieldValByIndex.IsValid(), fieldValByIndex.CanInterface()) // Log match details
					if fieldValByIndex.IsValid() && fieldValByIndex.CanInterface() {
						nextVal = fieldValByIndex
						found = true
						break
					}
				}
			}
		}

		if !found {
			fmt.Printf("[DEBUG getValueRecursive Struct] Field '%s' not found by name or JSON tag on type %s.\n", part, current.Type().String()) // Log failure
			return reflect.Value{}, false
		}
		return getValueRecursive(nextVal, remainingParts)

	case reflect.Map:
		var mapValue reflect.Value
		keyKind := current.Type().Key().Kind()
		if keyKind == reflect.String {
			mapValue = current.MapIndex(reflect.ValueOf(part))
			if !mapValue.IsValid() {
				iter := current.MapRange()
				for iter.Next() {
					k := iter.Key()
					if k.Kind() == reflect.String && strings.EqualFold(k.String(), part) {
						mapValue = iter.Value()
						break
					}
				}
			}
		}

		if !mapValue.IsValid() {
			return reflect.Value{}, false
		}
		return getValueRecursive(mapValue, remainingParts)

	default:
		return reflect.Value{}, false
	}
}

// extractFields uses getValueByPath to build the result map for a single issue.
func extractFields(issue mcpclient.Issue, fields []string) map[string]interface{} {
	result := make(map[string]interface{})
	for _, fieldPath := range fields {
		if value, found := getValueByPath(issue, fieldPath); found {
			result[fieldPath] = value
		} else {
			result[fieldPath] = nil // Field not found or inaccessible, include as null
		}
	}
	return result
}

// searchCmd represents the search command
var searchCmd = &cobra.Command{
	Use:   "search [JQL Query]",
	Short: "Search for JIRA issues using JQL",
	Long: `Searches for JIRA issues using a JQL query via the MCP server.
You can provide the JQL query directly as arguments or use the --jql flag.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgProvider := &DefaultConfigProvider{}
		cfg, err := cfgProvider.LoadConfig()
		if err != nil {
			log.Error().Err(err).Msg("Failed to load configuration for search command setup")
			if errors.Is(err, config.ErrConfigRead) || errors.Is(err, config.ErrConfigParse) {
				fmt.Fprintln(cmd.ErrOrStderr(), "Error reading or parsing config.yaml. Please check its format and permissions.")
			} else if errors.Is(err, config.ErrConfigDirCreate) || errors.Is(err, config.ErrConfigDirStat) || errors.Is(err, config.ErrConfigDirNotDir) {
				fmt.Fprintln(cmd.ErrOrStderr(), "Error accessing configuration directory. Please check permissions.")
			} else {
				fmt.Fprintf(cmd.ErrOrStderr(), "An unexpected error occurred loading config.yaml: %v\n", err)
			}
			fmt.Fprintln(cmd.ErrOrStderr(), "You might need to run 'tix config init'.")
			return err
		}

		mcpClient, err := newDefaultMCPClient(cfg)
		if err != nil {
			log.Error().Err(err).Msg("Failed to create MCP client for search command setup")
			if errors.Is(err, mcpclient.ErrMCPServerURLMissing) {
				fmt.Fprintln(cmd.ErrOrStderr(), "Error: MCP Server URL is not configured.")
				fmt.Fprintln(cmd.ErrOrStderr(), "Please set 'mcp_server_url' in ~/.ticketron/config.yaml or use the TICKETRON_MCP_SERVER_URL environment variable.")
			} else if errors.Is(err, mcpclient.ErrMCPServerURLParse) {
				fmt.Fprintf(cmd.ErrOrStderr(), "Error parsing MCP Server URL: %v\n", err)
				fmt.Fprintln(cmd.ErrOrStderr(), "Please check the URL format in your configuration.")
			} else {
				fmt.Fprintf(cmd.ErrOrStderr(), "Failed to initialize MCP client: %v\n", err)
			}
			return err
		}

		out := cmd.OutOrStdout()
		return searchRunE(cfgProvider, mcpClient, out, cmd, args)
	},
}

func init() {
	searchCmd.Flags().String("jql", "", "JQL query string")
	searchCmd.Flags().Int("max-results", 20, "Maximum number of results to return")
	searchCmd.Flags().StringP("output-fields", "f", "", "Comma-separated fields to include in JSON/YAML/TSV output (e.g., key,fields.summary,fields.status.name)") // Updated help text

	rootCmd.AddCommand(searchCmd)
}

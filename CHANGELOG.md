# Changelog

## [Unreleased]

### Changed
- Updated `CONTRIBUTING.md` to recommend using `Makefile` targets (`make fmt`, `make lint`, `make test`) in the contribution workflow.

### Fixed
- Corrected `Makefile` build target to use `./main.go` instead of `./cmd/tix`.


## [0.1.0] - 2025-04-17

### Added
- Initial project structure with Go modules (`go.mod`, `go.sum`).
- Basic `main.go` entrypoint using Cobra CLI framework (`github.com/spf13/cobra`).
- Placeholder `tix create` command (`cmd/create.go`) registered with root command.
- Placeholder flags (`--type`, `--summary`, `--project`, `--description`) to `create` command (`cmd/create.go`).
- `internal/config` package and `EnsureConfigDir` function to locate/create the config directory (`ticketron/internal/config/config.go`).
- `config.LoadLinks()` function in `internal/config` to load project key aliases from `~/.ticketron/links.yaml`.
- `config.LoadSystemPrompt()` function in `internal/config` to load the system prompt from `~/.ticketron/system_prompt.txt`.
- `config.LoadContext()` function in `internal/config` to load context from `~/.ticketron/context.md`.
- `config.CreateDefaultConfigFiles()` function in `internal/config` to create default config files (`config.yaml`, `links.yaml`, `system_prompt.txt`, `context.md`).
- Placeholder `config` command group (`ticketron/cmd/config.go`).
- Placeholder `config show` subcommand (`ticketron/cmd/config_show.go`).
- Placeholder `config locate` subcommand (`ticketron/cmd/config_locate.go`).
- Placeholder `config init` subcommand (`ticketron/cmd/config_init.go`).
- `internal/llm` directory for LLM interaction logic.
- `github.com/sashabaranov/go-openai` dependency.
- `internal/llm.ConstructPrompt` function to build the LLM prompt from user input, system prompt, and context, specifically requesting JSON output (`summary`, `description`, `project_name_suggestion`). (Phase 5, Task 3)
- `internal/llm.CallOpenAI` function to interact with the OpenAI API using the `go-openai` SDK.
- `internal/llm/parser.go` with `LLMResponse` struct and `ParseLLMResponse` function to parse JSON responses from the LLM. (Phase 5, Task 5)
- Defined Go structs (`CreateIssueResponse`, `SearchIssuesResponse`, `Issue`, `IssueFields`, `Status`, `IssueType`, `ErrorResponse`) for MCP API responses in `internal/mcpclient/types.go`.
- Implemented MCP client (`internal/mcpclient/client.go`) with `New()` constructor and `CreateIssue()` method for sending `POST /create_jira_issue` requests.
- Implemented `SearchIssues()` method in MCP client (`internal/mcpclient/client.go`) for sending `POST /search_jira_issues` requests.
- Implemented `GetIssue()` method in MCP client (`internal/mcpclient/client.go`) for sending `GET /jira_issue/{issueKey}` requests.
- Defined interfaces (`ConfigProvider`, `LLMProcessor`, `MCPClient`, `ProjectMapper`, `IssueTypeResolver`) in `cmd/interfaces.go` for `create` command dependencies.
- Secure API key handling using OS keychain (`github.com/zalando/go-keyring`) with fallback to `TICKETRON_LLM_API_KEY` environment variable (`internal/config/config.go`).
- New command `tix config set-key <api-key>` to store the API key in the keychain (`cmd/config_set_key.go`).
- Implemented `tix config locate` command to display configuration directory and file paths (`cmd/config_locate.go`).
- Implemented `tix config show` command to display current configuration (excluding API key) from `config.LoadConfig()` (`cmd/config_show.go`).
- Added `LICENSE` file (MIT License). (Phase 1, Task 3)
- Added `CONTRIBUTING.md` file with basic contribution guidelines. (Phase 1, Task 3)
- Added `golangci-lint` configuration (`.golangci.yml`) and integrated linting into the CI workflow (`.github/workflows/ci.yml`). (Phase 1, Task 4)
- Added structured logging using `zerolog` (`github.com/rs/zerolog`). (Phase 7, Task 3)
- Added persistent `--log-level` flag to `rootCmd` to control verbosity (debug, info, warn, etc.).
- Implemented `tix search [JQL]` command (`cmd/search.go`) to search JIRA issues via MCP server. (Phase 7, Task 4)
- Added `--jql` and `--max-results` flags to `search` command.
- Added `docs/usage.md` and `docs/architecture.md` with CLI usage examples and architecture overview. (Phase 7, Task 8)
- Added Go Report Card, GoDoc, and CI badges to `README.md`. (Phase 7, Task 7)
- Added `tix --version` flag to display the application version. (Phase 2.1, Task 2)
- Added comprehensive GoDoc comments to all exported symbols in `internal/`, `cmd/`, and `main.go`. (Phase 2.1, Task 3)
- Added code coverage report generation (`go test -coverprofile`) and upload to Codecov via GitHub Actions (`.github/workflows/ci.yml`). Added Codecov badge to `README.md`. (Phase 2.2, Task 1)
- Added vulnerability scanning using `govulncheck` (`.github/workflows/ci.yml`). (Phase 2.2, Task 2)
- Added `--interactive` (`-i`) flag to `tix create` command to prompt for user confirmation before creating the JIRA issue. (Phase 2.3, Task 1)
- Added persistent `--output` (`-o`) flag to support JSON output format for `tix search` and `tix create` commands. (Phase 2.3, Task 2)
- Added `tix completion [bash|zsh|fish|powershell]` command to generate shell completion scripts. (Phase 3.2, Task 3)
- Added `tix context show|edit|add` commands for managing the persistent LLM context file (`~/.ticketron/context.md`). (Phase 3.3, Task 2)

### Changed
- Implemented `config.LoadConfig()` function in `internal/config` to load configuration from `~/.ticketron/config.yaml` using Viper, with support for defaults and environment variable overrides (`TICKETRON_*`).
- Integrated configuration loading (`LoadConfig`, `LoadLinks`, `LoadSystemPrompt`, `LoadContext`) into the `create` command (`cmd/create.go`), including error handling. (Phase 6, Task 1)
- Integrated LLM interaction (prompt, API call, parse) into the `create` command (`cmd/create.go`), using loaded config and handling API key presence and errors. (Phase 6, Task 2)
- Implemented project name to key mapping in `create` command (`cmd/create.go`) using `links.yaml` data, with case-insensitive matching and error handling for mismatches. (Phase 6, Task 3)
- Implemented logic in `create` command to determine final issue type based on `--type` flag, `links.yaml` default, or hardcoded "Task". (Phase 6, Task 4)
- Integrated MCP client call (`CreateIssue`) into the `create` command (`cmd/create.go`). (Phase 6, Task 6)
- Replaced `fmt.Print*` and `fmt.Errorf` calls in `cmd/create.go` with `Log.Info()`, `Log.Debug()`, and `Log.Error().Err(err)` for progress, details, and error reporting. (Phase 7, Task 1)
- Improved error handling and user feedback in `create` command (`cmd/create.go`), providing more specific messages for config loading, API key issues, project mapping, LLM failures, and MCP interactions. Added basic progress messages. (Phase 7, Task 1)
- Added handling for MCP server URL configuration and client creation errors in `cmd/create.go`.
- Added handling for `CreateIssue` API call errors and success responses (displaying key/URL) in `cmd/create.go`.
- Added `context` import to `cmd/create.go`.
- Updated `README.md` with elevator pitch, features, installation, configuration, usage examples, and development instructions. (Phase 7, Task 6)
- Refactored `config.LoadLinks` to return `LinksConfig` struct instead of `map[string]string`, enabling access to `DefaultIssueType`. Updated `cmd/create.go` to use the new struct for project mapping and issue type determination logic. (Phase 7, Refactor Task)
- Refactored `internal/config` package for improved testability, introducing `baseDir` parameter to functions like `EnsureConfigDir`, `LoadConfig`, etc. (Phase 2.1, Task 1)
- Updated `internal/config/config_test.go` to use `t.TempDir()` for creating temporary test directories. (Phase 2.1, Task 1)
- Refined logging levels in `cmd/create.go`, changing routine `Info` logs to `Debug`. (Phase 2.1, Task 4)
- Enhanced `README.md`: Updated prerequisites (Go 1.24+, make), added Quick Start section, updated Development section with Makefile info, clarified API key handling, and improved overall clarity for newcomers. (Phase 2, Task 2.1)
- Enhanced `docs/usage.md`: Added YAML example with field selection (`-f`) for `search` command and clarified `-f` flag description. (Phase 2, Task 2.2)
- Updated `docs/architecture.md`: Corrected configuration filenames (`links.yaml` instead of `projects.yaml`) and clarified secure API key handling (keychain/env vars instead of direct config storage). Updated diagrams to reflect keyring usage. (Phase 2, Task 2.3)
- Refined `golangci-lint` configuration: enabled `gocritic`, `gocyclo` (max-complexity: 20), `misspell`, `bodyclose`, `unconvert`. Fixed resulting simple lint issues. Updated deprecated `run.skip-files` to `issues.exclude-files`. (Phase 2.2, Task 3)
- Improved robustness of LLM response parser (`internal/llm/parser.go`) to handle various markdown code fence formats and whitespace variations using regex. Added corresponding test cases (`internal/llm/parser_test.go`). (Phase 2.2, Task 4)
- Updated `README.md` and `docs/usage.md` to document `--version`, `--interactive`, and `--output` flags. (Phase 2.3, Task 3)
- Refactored `cmd/create.go`'s `createCmdRunner.Run` function to improve readability by extracting helper functions. (Phase 3.2, Task 1)
- Updated `README.md` and `docs/usage.md` with instructions on how to use and install shell completions. (Phase 3.2, Task 3)
- Refactored LLM interaction to support multiple providers (interface, OpenAI impl, config updates, DI updates). (Phase 3.3, Task 1)
- Standardized error handling across `internal/` packages using sentinel errors (`config`, `llm`, `mcpclient`).
- Improved user-facing error messages in `cmd/create` and `cmd/search` to provide more context and guidance on failure.
- Updated unit tests to align with new error handling patterns.
- Updated `LLMProcessor` interface and implementations to accept `*config.AppConfig`.
- Updated `internal/llm/client.go`'s `CallOpenAI` function to accept the model name from configuration.
- Updated `internal/config/config.go`'s `EnsureConfigDir` function to prioritize `TICKETRON_CONFIG_DIR`.
- Updated `internal/config/config_test.go` to fix YAML syntax errors and align assertions.
- Updated integration tests (`test/integration/`) to fix command arguments, struct field names, mock response formats, and config file generation.
- Corrected JSON output format in `cmd/search.go` to marshal the entire response object.
- Refactored `cmd/config_show.go`, `cmd/create.go`, `cmd/search.go`, `cmd/config_set_key.go`, `cmd/config_init.go`, `cmd/config_locate.go` to use dependency injection for improved testability.
- Moved shared provider implementations to `cmd/providers.go`.
- Updated `tix create` command to retrieve the API key using the new secure method (`cmd/create.go`).
- Updated `tix config init` command to use the correct function for creating default config files (`cmd/config_init.go`).
- Updated `README.md` and `docs/usage.md` to reflect the new secure API key handling (keychain command and environment variable).

### Fixed
- Basic validation for LLM response in `internal/llm/parser`.
- Improved error wrapping using `fmt.Errorf` with `%w` in `cmd/search` and `internal/llm`.

### Removed
- `llm.api_key` field from `config.yaml` and `internal/config.AppConfig`.
- Deleted `cmd/create_test.go` (to be rewritten).

---
All notable changes to this project will be documented in this file.

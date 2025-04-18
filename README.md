# Ticketron (tix)

A smart CLI helper for JIRA, powered by LLMs.

[![Go Report Card](https://goreportcard.com/badge/github.com/karolswdev/ticketron)](https://goreportcard.com/report/github.com/karolswdev/ticketron) [![GoDoc](https://godoc.org/github.com/karolswdev/ticketron?status.svg)](https://godoc.org/github.com/karolswdev/ticketron) [![CI](https://github.com/karolswdev/ticketron/actions/workflows/ci.yml/badge.svg)](https://github.com/karolswdev/ticketron/actions/workflows/ci.yml) [![codecov](https://codecov.io/gh/karolswdev/ticketron/branch/main/graph/badge.svg)](https://codecov.io/gh/karolswdev/ticketron)

## Elevator Pitch

Ticketron (`tix`) simplifies interacting with JIRA directly from your command line. Leveraging the power of Large Language Models (LLMs) via the `jira-mcp-server` and OpenAI, it allows you to create and search for JIRA tickets using natural language or concise commands. The primary goal is to streamline your workflow, making ticket management faster and more intuitive by automatically adding relevant context.

## Features

*   Create JIRA issues using natural language (`tix create ...`)
*   Search JIRA issues using JQL (`tix search ...`)
*   Contextual ticket generation via LLM (using OpenAI)
*   Configuration management (`tix config init|show|locate|set-key`)
*   Secure API key storage (OS Keychain or Env Var)
*   LLM Context management (`tix context show|edit|add`)
*   Shell completion support (Bash, Zsh, Fish, PowerShell)
*   Structured output formats (JSON, YAML, TSV)
*   Structured logging for easier debugging

## Installation

### Prerequisites

*   **Go:** Version 1.24.0 or later.
*   **jira-mcp-server:** Access to a running instance of the [jira-mcp-server](https://github.com/karolswdev/ticketron/tree/main/jira-mcp/jira-mcp-server) (required for JIRA communication).
*   **OpenAI API Key:** Required for the LLM features.
*   **Make:** Required for development tasks (building, testing, linting).

### Install Command

```bash
go install github.com/karolswdev/ticketron/cmd/tix@latest
```
Ensure your `$GOPATH/bin` or `$HOME/go/bin` is in your `PATH`.


### Shell Completion

`tix` supports generating shell completion scripts for Bash, Zsh, Fish, and PowerShell.

To generate the script for your shell, use the `completion` command:

```bash
# For Bash
tix completion bash

# For Zsh
tix completion zsh

# For Fish
tix completion fish

# For PowerShell
tix completion powershell
```

Follow the instructions printed by the command to load the completions into your current session or configure them to load automatically.

Example for Bash (add to `~/.bashrc` or `~/.bash_profile`):

```bash
source <(tix completion bash)
```

Example for Zsh (add to `~/.zshrc`):

```bash
# Ensure compinit is loaded
autoload -U compinit; compinit

# Load completions
source <(tix completion zsh)
```

Refer to the output of `tix completion [your-shell]` for the most accurate setup instructions for your environment.

## Quick Start

1.  **Install `tix`:**
    ```bash
    go install github.com/karolswdev/ticketron/cmd/tix@latest
    ```
    (Ensure `$GOPATH/bin` or `$HOME/go/bin` is in your `PATH`)

2.  **Initialize Configuration:**
    ```bash
    tix config init
    ```
    (Creates `~/.ticketron/` with default files. Edit `~/.ticketron/config.yaml` to set your `mcp_server.url`.)

3.  **Set OpenAI API Key:**
    ```bash
    # Recommended: Use OS Keychain
    tix config set-key <your-openai-api-key>

    # Alternative: Use Environment Variable
    # export TICKETRON_LLM_API_KEY="<your-openai-api-key>"
    ```

4.  **Create your first ticket:**
    ```bash
    # Example: Create a bug report (uses LLM if description is provided)
    tix create bug "Login button unresponsive" --description "When clicking the login button on the main page, nothing happens. Check console logs."
    ```

## Configuration

Ticketron requires configuration files stored in `~/.ticketron/`. You can initialize a default configuration directory and template files using:

```bash
tix config init
```

This command creates the `~/.ticketron/` directory if it doesn't exist and populates it with default template files (`config.yaml`, `links.yaml`, `system_prompt.txt`, `context.md`). It **does not** handle the API key setup.

### OpenAI API Key

The OpenAI API key is required for LLM features and must be configured separately for security reasons. You have two options:

1.  **Recommended: Use the OS Keychain:**
    ```bash
    tix config set-key <your-openai-api-key>
    ```
    This command securely stores your key in the operating system's keychain. `ticketron` will automatically retrieve it from there.

2.  **Alternative: Environment Variable:**
    Set the `TICKETRON_LLM_API_KEY` environment variable:
    ```bash
    export TICKETRON_LLM_API_KEY="<your-openai-api-key>"
    ```
    The application will use this variable if the key is not found in the keychain.

### Configuration Files

Key configuration files found in `~/.ticketron/`:

*   **`config.yaml`**: Main configuration.
    *   `mcp_server.url`: URL of your running `jira-mcp-server`.
    *   Other settings like default project, issue type, logging level.
    *   **Note:** The `openai.api_key` is **no longer stored here**.
*   **`links.yaml`**: Maps short project aliases to full JIRA project keys and default issue types.
    *   Example:
        ```yaml
        links:
          - name: WEB
            key: WEBPROJECT
            default_issue_type: Story
          - name: API
            key: BACKENDAPI
            default_issue_type: Task
          - name: DS
            key: DATASCIENCE
            default_issue_type: Bug
        ```
*   **`system_prompt.txt`**: Contains the system prompt used to instruct the LLM during ticket creation. Modify this to tailor the LLM's behavior.
*   **`context.md`**: Provides persistent context to the LLM about your projects, team, or common ticket patterns.

Use `tix config show` to view the current configuration (excluding the API key) and `tix config locate` to find the configuration directory.

## Basic Usage
Use `tix --version` to display the application version.


### Create Tickets

```bash
# Create a bug with a simple title
tix create bug "Login fails on checkout page"

# Create a story for a specific project (using alias from links.yaml)
tix create --type Story --project WEB "Implement new user profile page design"

# Create a task with more detail (will be processed by LLM)
tix create task "Refactor auth module to use JWT" --description "The current session management is outdated. We need to switch to JWT for better security and scalability. Update login, logout, and token refresh endpoints."

# Create a ticket interactively (prompts for confirmation before sending)
tix create -i bug "UI glitch on settings page"

# Create a ticket and output the result as JSON
tix create task "Setup CI pipeline" -o json
```

### Search Tickets

```bash
# Search using natural language (processed by LLM to generate JQL)
tix search "my open bugs in the WEB project"

# Search using explicit JQL
tix search --jql "project = WEB AND status = 'In Progress' ORDER BY created DESC"

# Search for unresolved tickets assigned to you
tix search --jql "assignee = currentUser() AND resolution = Unresolved"

# Search for tickets and output results as JSON
tix search "my open tasks" -o json
# Search and output specific fields as JSON
tix search "status = Done" -o json -f key,fields.summary,fields.status.name

# Search and output results as YAML
tix search "my open tasks" -o yaml

# Search and output results as TSV (Tab-Separated Values)
tix search "project = API" -o tsv

# Search and output specific fields as TSV
tix search "project = API" -o tsv -f key,fields.issuetype.name,fields.summary

```

### Manage Configuration

```bash
# Initialize configuration files
tix config init

# Show current configuration values
tix config show

# Show the location of the configuration directory
tix config locate

# Securely set the OpenAI API key in the OS keychain
tix config set-key <your-openai-api-key>
```



### Manage LLM Context

The `context.md` file provides persistent context to the LLM. You can manage this file using the `tix context` commands:

```bash
# Show the current content of the context file
tix context show

# Open the context file in your default editor ($EDITOR)
tix context edit

# Add a new line of text to the context file
tix context add "Remember to always include the component name in bug reports."
```


## Development

This project uses a `Makefile` to streamline common development tasks. Ensure you have `make` installed.

*   **Build:** Compile the `tix` binary.
    ```bash
    make build
    ```
*   **Test:** Run unit and integration tests.
    ```bash
    make test
    ```
*   **Lint:** Run the Go linter (`golangci-lint`).
    ```bash
    make lint
    ```
*   **Format:** Format Go source code using `gofmt`.
    ```bash
    make fmt
    ```
*   **All Checks:** Run format, lint, and test targets.
    ```bash
    make check
    ```

Refer to the `Makefile` for other available targets.

## Contributing

Please note that this project is released with a [Contributor Code of Conduct](./CODE_OF_CONDUCT.md). By participating in this project you agree to abide by its terms.

See [CONTRIBUTING.md](./CONTRIBUTING.md) for details on how to contribute.

## License

This project is licensed under the MIT License. See the LICENSE file for details.
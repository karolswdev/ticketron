# docs/usage.md

This document provides detailed usage instructions for the `ticketron` CLI tool (`tix`).

## Global Flags

*   `--log-level <level>`: Sets the logging level for the command execution. Available levels: `debug`, `info`, `warn`, `error`. Default is `info`.
    ```bash
    tix --log-level debug create "Fix the login button alignment"
    ```
*   `--version`: Displays the application version.
    ```bash
    tix --version
    ```



## Shell Completion

`tix` can generate completion scripts for common shells, allowing you to use tab-completion for commands and flags.

**Generating the Script:**

Use the `completion` command followed by your shell type:

```bash
tix completion [bash|zsh|fish|powershell]
```

**Loading Completions:**

The command output will include instructions specific to your shell. Here are common examples:

*   **Bash:**
    *   Load for current session:
        ```bash
        source <(tix completion bash)
        ```
    *   Load permanently (add to `~/.bashrc` or `~/.bash_profile`):
        ```bash
        # Add this line to your profile
        source <(tix completion bash)
        ```
        Alternatively, save the output to the appropriate system directory (e.g., `/etc/bash_completion.d/` or `/usr/local/etc/bash_completion.d/`).

*   **Zsh:**
    *   Ensure `compinit` is enabled (add to `~/.zshrc` if needed):
        ```zsh
        autoload -U compinit; compinit
        ```
    *   Load for current session:
        ```zsh
        source <(tix completion zsh)
        ```
    *   Load permanently (add to `~/.zshrc`):
        ```zsh
        # Add this line to your .zshrc
        source <(tix completion zsh)
        ```
        Alternatively, save the output to a file in your `$fpath` (e.g., `tix completion zsh > "${fpath[1]}/_tix"`).

*   **Fish:**
    *   Load for current session:
        ```fish
        tix completion fish | source
        ```
    *   Load permanently:
        ```fish
        tix completion fish > ~/.config/fish/completions/tix.fish
        ```

*   **PowerShell:**
    *   Load for current session:
        ```powershell
        tix completion powershell | Out-String | Invoke-Expression
        ```
    *   Load permanently: Save the output to a `.ps1` file and source it from your PowerShell profile.
        ```powershell
        tix completion powershell > $HOME\Documents\PowerShell\tix_completion.ps1
        # Add '. $HOME\Documents\PowerShell\tix_completion.ps1' to your profile
        ```

Always refer to the output of `tix completion [your-shell]` for the most precise instructions.



## Configuration Overview

`ticketron` relies on configuration files typically stored in `~/.ticketron/`. You can initialize this directory and default template files using `tix config init`.

### OpenAI API Key

The OpenAI API key is essential for features involving the LLM (like `tix create` without explicit details). It is **not** stored directly in `config.yaml` for security. Provide it using one of these methods:

1.  **Keychain (Recommended):** Use `tix config set-key <your-key>` to store it securely in your OS keychain.
2.  **Environment Variable:** Set the `TICKETRON_LLM_API_KEY` environment variable.

The tool prioritizes the keychain, falling back to the environment variable if the key isn't found in the keychain.

### Key Files in `~/.ticketron/`

*   **`config.yaml`**: Contains settings like the `mcp_server.url`, default project/issue type fallbacks, and logging preferences.
*   **`links.yaml`**: Maps convenient project aliases (e.g., `WEB`) to full JIRA project keys (e.g., `WEBPROJECT`) and specifies default issue types per project.
*   **`system_prompt.txt`**: The template used to instruct the LLM. Customize this to guide ticket generation.
*   **`context.md`**: Provides persistent background context to the LLM (e.g., team standards, project details).

---
## `tix create`

Creates a new JIRA issue based on natural language input. The tool uses an LLM to parse the input and determine the appropriate summary, description, project, and issue type.

**Basic Usage:**

```bash
# Let the LLM infer project and type
tix create "The user profile page is throwing a 500 error after the latest deployment."

# Specify the issue type
tix create --type Bug "Cannot add attachments to comments on issue PROJ-123."

# Specify the project key
tix create --project WEB "Implement a dark mode toggle in the user settings."

# Specify both project and type
tix create --project API --type Task "Refactor the authentication middleware to use JWT."

# More complex request
tix create --project CORE --type Story "As a user, I want to be able to reset my password via email so that I can regain access to my account if I forget my password."

# Create interactively (prompts for confirmation)
tix create -i "Fix typo on landing page"

# Create and output result as JSON
tix create --project API --type Task "Add rate limiting" -o json
```

**Flags:**

*   `--type <type>`: Specify the JIRA issue type (e.g., Bug, Story, Task).
*   `--project <key|alias>`: Specify the JIRA project key or an alias defined in `links.yaml`.
*   `--description <text>`: Provide a detailed description for the issue. If omitted, the LLM might generate one based on the summary.
*   `-i`, `--interactive`: Prompt for confirmation before creating the issue.
*   `-o`, `--output <format>`: Specify the output format. Currently supports `json`.

**Notes:**

*   If `--project` or `--type` are not provided, `ticketron` attempts to infer them from your input, `links.yaml`, and `config.yaml`.
*   The LLM generates a summary and description based on your input if not fully specified.

## `tix search`

Searches for JIRA issues using JIRA Query Language (JQL).

**Basic Usage:**

```bash
# Find all open bugs assigned to the current user in the 'PROJ' project
tix search --jql 'project = PROJ AND status = Open AND assignee = currentUser() AND type = Bug'

# Find the 5 most recently updated issues in any project
tix search --jql 'ORDER BY updated DESC' --max-results 5

# Find issues created in the last 7 days containing "login" in the summary
tix search --jql 'summary ~ "login" AND created >= -7d'

# Find high-priority tasks in the 'API' project
tix search --jql 'project = API AND priority = High AND type = Task'

# Search and output results as JSON
tix search --jql "assignee = currentUser() AND status = 'In Review'" -o json

# Search and output specific fields as JSON
tix search "status = Done" -o json -f key,fields.summary,fields.status.name

# Search and output results as YAML
tix search "my open tasks" -o yaml

# Search and output results as TSV (Tab-Separated Values)
tix search "project = API" -o tsv

# Search and output specific fields as TSV
tix search "project = API" -o tsv -f key,fields.issuetype.name,fields.summary

# Search and output specific fields as YAML
tix search "project = WEB AND status = 'Code Review'" -o yaml -f key,fields.summary,fields.assignee.displayName
```

**Flags:**

*   `--jql <query>`: (Required) The JIRA Query Language string.
*   `--max-results <number>`: The maximum number of issues to return. Defaults to 50.
*   `-o`, `--output <format>`: Specify the output format. Supports `text` (default), `json`, `yaml`, `tsv`.
*   `-f`, `--output-fields <fields>`: Comma-separated list of fields to include when using structured output formats (`json`, `yaml`, `tsv`). Use JIRA field dot notation (e.g., `key,fields.summary,fields.status.name`). If omitted for `tsv`, default fields are used; for `json`/`yaml`, the full issue structure is returned by default.
## `tix config`

Manages the `ticketron` configuration.

**Subcommands:**

*   `tix config show`: Displays the currently loaded configuration values (from `~/.ticketron/config.yaml`).
    ```bash
    tix config show
    ```
*   `tix config locate`: Prints the path to the configuration files (`config.yaml` and `projects.yaml`).
    ```bash
    tix config locate
    ```
*   `tix config init`: Creates the `~/.ticketron/` directory and default template configuration files (`config.yaml`, `links.yaml`, `system_prompt.txt`, `context.md`) if they don't exist. **Note:** This command *no longer* handles API key setup; use `tix config set-key` for that.
    ```bash
    tix config init
    ```


*   `tix config set-key <api-key>`: Securely stores your OpenAI API key in the OS keychain. This is the recommended way to provide the key.
    ```bash
    tix config set-key sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
    ```



## `tix context`

Manages the persistent LLM context file (`~/.ticketron/context.md`). This file provides background information to the LLM for tasks like ticket creation.

**Subcommands:**

*   `tix context show`: Displays the current content of the `context.md` file.
    ```bash
    tix context show
    ```
*   `tix context edit`: Opens the `context.md` file in your default text editor (determined by the `$EDITOR` environment variable, falling back to `vim` or `notepad`).
    ```bash
    tix context edit
    ```
*   `tix context add <entry>`: Appends a new line of text to the `context.md` file.
    ```bash
    tix context add "All backend services are written in Go."
    tix context add "Frontend team prefers tasks over stories for UI bugs."
    ```


---
# Secure API Key Handling Plan for Ticketron

*(Generated: 2025-04-17)*

## 1. Context

This document outlines the plan for addressing Task 5 of Phase 7 (`TODO.md`) for the `ticketron` project: exploring and implementing secure handling for sensitive configuration, specifically the OpenAI API key.

Currently, the API key can be stored in plain text within `~/.ticketron/config.yaml` (`llm.api_key`), although an environment variable override (`TICKETRON_LLM_API_KEY`) is also supported via Viper. Storing secrets directly in configuration files is insecure.

## 2. Analysis of Options

Several methods for handling the API key were analyzed:

*   **Environment Variables (`TICKETRON_LLM_API_KEY`)**:
    *   *Pros:* Easy implementation (already partially supported), common practice, good cross-platform compatibility.
    *   *Cons:* Moderate security (can be inspected/leaked if not managed carefully).
    *   *Verdict:* Good baseline/fallback.
*   **OS Keychain/Keyring (via `go-keyring`)**:
    *   *Pros:* High security (uses native OS secure storage), good user experience (set once).
    *   *Cons:* Moderate implementation complexity (requires library, setup command).
    *   *Verdict:* Best balance of security and usability for a local CLI tool.
*   **Dedicated Secrets Management Tools (Vault, etc.)**:
    *   *Pros:* Very high security, centralized management.
    *   *Cons:* High implementation complexity, complex UX for local tool.
    *   *Verdict:* Overkill for `ticketron`.
*   **Separate, Permission-Restricted File**:
    *   *Pros:* Easy implementation, better than main config file.
    *   *Cons:* Low-moderate security (still plain text on disk).
    *   *Verdict:* Less secure/convenient than keychain or env vars.
*   **Command-Line Flags (`--api-key=...`)**:
    *   *Pros:* Trivial implementation.
    *   *Cons:* Very poor security (leaks via history/process list).
    *   *Verdict:* Unsuitable.

## 3. Recommendation

A tiered approach is recommended:

1.  **Primary Method: OS Keychain/Keyring**
    *   Use `github.com/zalando/go-keyring` to store and retrieve the API key.
2.  **Fallback Method: Environment Variable**
    *   Support `TICKETRON_LLM_API_KEY`.
3.  **Remove:** Ability to store the key directly in `config.yaml`.

*Rationale:* This approach prioritizes the most secure practical method (keychain) while providing a standard, flexible fallback (environment variable) suitable for various user environments. It explicitly removes the insecure plain text file storage option.

## 4. High-Level Implementation Plan

1.  **Add Dependency:** `go get github.com/zalando/go-keyring`
2.  **Modify `internal/config/config.go`:**
    *   Remove `APIKey` field from `LLMConfig` struct.
    *   Create a new function `GetAPIKey() (string, error)` that:
        *   Attempts to retrieve the key using `keyring.Get("ticketron", "openai_api_key")`.
        *   If keychain fails/key not found, attempts to retrieve using `os.Getenv("TICKETRON_LLM_API_KEY")`.
        *   Returns an error if neither method yields a key, instructing the user on how to set it.
    *   Remove the `llm.api_key` entry from the `defaultConfigYAML` constant.
3.  **Create `cmd/config_set_key.go`:**
    *   Implement a new command `tix config set-key`.
    *   Prompt the user securely for their API key.
    *   Store the key using `keyring.Set("ticketron", "openai_api_key", key)`.
    *   Provide confirmation feedback to the user.
4.  **Update `cmd/create.go` and `cmd/search.go`:**
    *   Replace direct access to `cfg.LLM.APIKey` with a call to `config.GetAPIKey()`.
    *   Implement proper error handling if `GetAPIKey()` returns an error.
    *   Use the retrieved key to initialize the OpenAI client.
5.  **Update Documentation (`README.md`, `docs/usage.md`):**
    *   Clearly document `tix config set-key` as the primary, recommended method for setting the API key.
    *   Document the `TICKETRON_LLM_API_KEY` environment variable as an alternative/fallback method.
    *   Remove any mention of setting `llm.api_key` within the `config.yaml` file.
6.  **Update `cmd/config_init.go` / `CreateDefaultConfigFiles`:**
    *   Ensure the `defaultConfigYAML` constant in `internal/config/config.go` no longer includes the `llm.api_key` field.
    *   Consider adding an informational message during the `init` process suggesting the user run `tix config set-key` after initialization.
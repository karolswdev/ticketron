package config

import "errors"

// Sentinel errors for configuration loading and processing.

// ErrConfigNotFound indicates the main config file (config.yaml) was not found.
var ErrConfigNotFound = errors.New("configuration file not found")

// ErrConfigRead indicates an error occurred while reading the config file.
var ErrConfigRead = errors.New("failed to read configuration file")

// ErrConfigParse indicates an error occurred while parsing the config file.
var ErrConfigParse = errors.New("failed to parse configuration file")

// ErrLinksNotFound indicates the links file (links.yaml) was not found.
var ErrLinksNotFound = errors.New("links file not found")

// ErrLinksRead indicates an error occurred while reading the links file.
var ErrLinksRead = errors.New("failed to read links file")

// ErrLinksParse indicates an error occurred while parsing the links file.
var ErrLinksParse = errors.New("failed to parse links file")

// ErrSystemPromptNotFound indicates the system prompt file (system_prompt.txt) was not found.
var ErrSystemPromptNotFound = errors.New("system prompt file not found")

// ErrSystemPromptRead indicates an error occurred while reading the system prompt file.
var ErrSystemPromptRead = errors.New("failed to read system prompt file")

// ErrContextNotFound indicates the context file (context.md) was not found.
var ErrContextNotFound = errors.New("context file not found")

// ErrContextRead indicates an error occurred while reading the context file.
var ErrContextRead = errors.New("failed to read context file")

// ErrConfigDirCreate indicates an error occurred while creating the config directory.
var ErrConfigDirCreate = errors.New("failed to create config directory")

// ErrConfigDirStat indicates an error occurred while checking the config directory.
var ErrConfigDirStat = errors.New("failed to check config directory")

// ErrConfigDirNotDir indicates the config path exists but is not a directory.
var ErrConfigDirNotDir = errors.New("config path exists but is not a directory")

// ErrDefaultFileWrite indicates an error occurred while writing a default config file.
var ErrDefaultFileWrite = errors.New("failed to write default config file")

// ErrDefaultFileStat indicates an error occurred while checking a default config file.
var ErrDefaultFileStat = errors.New("failed to check default config file")

// ErrProjectMappingFailed indicates that a suggested project name could not be mapped to a key.
var ErrProjectMappingFailed = errors.New("could not map project name suggestion to a known project key")

// ErrKeyringSet indicates an error occurred while setting a key in the OS keyring.
var ErrKeyringSet = errors.New("failed to set key in OS keyring")

// ErrKeyringGet indicates an error occurred while getting a key from the OS keyring (excluding 'not found').
var ErrKeyringGet = errors.New("failed to get key from OS keyring")

// ErrAPIKeyNotFound is defined in config.go for now due to usage scope, but logically belongs here.
// Consider moving it if refactoring occurs.

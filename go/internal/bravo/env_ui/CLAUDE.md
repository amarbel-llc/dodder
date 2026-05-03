# env_ui

CLI environment and user interface abstraction layer.

## Key Types

- `Env`: Main interface providing access to stdin/stdout/stderr and UI operations
- `env`: Implementation managing file descriptors and CLI state
- `Options`: Configuration for UI behavior (stderr output, TTY state, prefix)

## Features

- Standardized access to stdin, stdout, stderr, and UI output
- User confirmation prompts via `Confirm()`
- Error retry dialogs via `Retry()`
- Color output configuration for different streams
- String format writer creation with truncation and color options
- Integration with CLI config and debug context
- Verbose and quiet mode support

# command

CLI command framework with request handling, flag parsing, and shell completion.

## Key Types

- `Cmd`: Base command interface with Run method
- `Request`: Command request with context and environment
- `Utility`: Command utility grouping related commands
- `Description`: Short and long command descriptions
- `Completion`: Shell completion data

## Features

- Flag set management with custom parsing
- Command registration via utilities map
- Request-based execution pattern
- Shell completion generation support

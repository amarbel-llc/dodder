# config_cli

Common CLI configuration flags for commands.

## Key Types

- `Config` - standard CLI flags struct

## Fields

- `Debug` - debug.Options for debug mode
- `Verbose/Quiet` - output verbosity flags
- `Todo` - TODO mode flag
- `dryRun` - dry-run mode (private, accessed via IsDryRun/SetDryRun)

## Features

- Implements CommandComponentWriter for flag definitions
- Default() constructor for standard configuration
- Flag definitions via SetFlagDefinitions()

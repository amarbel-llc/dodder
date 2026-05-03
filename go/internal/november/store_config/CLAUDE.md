# store_config

Mutable store configuration managing types, tags, repos, and runtime settings.

## Key Types

- `Store`: Read-only configuration access interface
- `StoreMutable`: Mutable store with add/flush operations
- `Config`: Full configuration including tags, types, repos, print options

## Features

- Type and tag registration from transacted objects
- Mutable configuration with change tracking
- File extension mappings for types
- Print options and defaults management
- Recompilation tracking for config changes

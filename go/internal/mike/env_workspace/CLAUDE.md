# env_workspace

Workspace environment managing working directory, configuration, and filesystem store.

## Key Types

- `Env`: Main workspace environment interface
- `Config`: Workspace configuration combining defaults and file extensions
- `Store`: Workspace store with supplies and storage implementation

## Features

- Detects and loads `.dodder-workspace` configuration files
- Supports temporary workspaces when no config exists
- Manages workspace defaults (type, tags) from config hierarchy
- Creates and initializes filesystem store for working copies
- Finds workspace config by walking up directory tree

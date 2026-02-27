# env_repo

Repository environment managing configuration, directory layout, blob stores, and locking.

## Key Types

- `Env`: Main repository environment with config and directory layouts
- `Options`: Repository initialization options (base path, permit flags)
- `BlobStoreEnv`: Blob store environment for content storage

## Features

- Loads and manages genesis configuration (private/public)
- Creates directory layout for blobs and repository data
- Manages file locking via locksmith pattern
- Handles XDG base directory paths
- Provides blob store access and inventory list storage
- Supports cache reset operations

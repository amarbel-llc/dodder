# repo

Repository interface definitions for local and remote repository operations.

## Key Types

- `Repo`: Base repository interface for object store, blob store, and queries
- `LocalRepo`: Extended interface with locking and workspace access
- `Importer`: Import operation handler
- `ImporterOptions`: Configuration for import operations

## Features

- Unified interface for local and remote repositories
- Query group construction and execution
- Inventory list management
- Remote pull operations
- Object history retrieval

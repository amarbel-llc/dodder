# typed_blob_store

Type-safe blob storage abstraction with versioned codec support for different content types.

## Key Types

- `Stores`: Collection of specialized typed stores (inventory lists, repos, types, tags)
- `RepoStore`: Typed storage for repository configuration blobs
- `Tag`: Multi-version tag blob store (TOML v0/v1, Lua v1/v2)
- `Config`: Multi-version config store (TOML v0/v1)

## Features

- Generic typed blob store with compile-time type safety
- Version-specific decoders (TOML and Lua formats)
- Automatic format selection based on type string
- Lua VM pooling for executable tag definitions
- Buffered read/write with automatic pool management

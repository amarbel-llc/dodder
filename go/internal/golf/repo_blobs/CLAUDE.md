# repo_blobs

Repository blob location and access configuration.

## Key Types

- `Blob`: Base interface for repository blob access
- `BlobMutable`: Writable blob interface with public key management
- `BlobXDG`: XDG-compliant directory blob storage
- `BlobOverridePath`: Blob with custom path override
- `BlobUri`: URI-based blob access

## Versions

- `TomlXDGV0`: XDG directory-based configuration
- `TomlLocalOverridePathV0`: Custom path override configuration
- `TomlUriV0`: URI-based configuration

## Features

- Remote vs local blob detection via `IsRemote()`
- Public key management for repository identification
- Supported connection types: native, Unix socket, stdio (local/SSH), URL

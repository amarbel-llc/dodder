# blob_store_configs

Configuration types and interfaces for blob storage backends.

## Key Types

- `Config`: Base interface for blob store configurations
- `ConfigMutable`: Writable configuration interface
- `ConfigLocalHashBucketed`: Configuration for hash-bucketed local storage
- `ConfigSFTPUri`, `ConfigSFTPConfigExplicit`: SFTP remote storage configurations
- `TypedConfig`, `TypedMutableConfig`: Type-safe config wrappers

## Versions

- `TomlV0`, `TomlV1`, `TomlV2`: Versioned TOML configuration formats
- `TomlSFTPV0`, `TomlSFTPViaSSHConfigV0`: SFTP-specific configurations
- `TomlPointerV0`, `TomlUriV0`: Pointer and URI-based configurations

## Features

- Default hash type: BLAKE2b-256
- Hash bucketing with configurable depth (default: 2-char buckets)
- Compression and encryption support via interfaces
- Internal file locking support

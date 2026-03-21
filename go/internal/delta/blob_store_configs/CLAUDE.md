# blob_store_configs

Configuration types and interfaces for blob storage backends.

## Key Types

- `Config`: Base interface for blob store configurations
- `ConfigMutable`: Writable configuration interface
- `ConfigLocalHashBucketed`: Configuration for hash-bucketed local storage
- `ConfigSFTPUri`, `ConfigSFTPConfigExplicit`: SFTP remote storage
  configurations
- `TypedConfig`, `TypedMutableConfig`: Type-safe config wrappers

## Versions

- `TomlLocalHashBucketedV0`, `TomlLocalHashBucketedV1`,
  `TomlLocalHashBucketedV2`: Versioned TOML configuration for local
  hash-bucketed stores
- `TomlSFTPV0`, `TomlSFTPViaSSHConfigV0`: SFTP-specific configurations
- `TomlPointerV0`, `TomlUriV0`: Pointer and URI-based configurations

## Features

- Default hash type: BLAKE2b-256
- Hash bucketing with configurable depth (default: 2-char buckets)
- Compression and encryption support via interfaces
- Internal file locking support

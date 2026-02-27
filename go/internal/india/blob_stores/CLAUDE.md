# blob_stores

Factory and management layer for content-addressable blob storage backends.

## Key Types

- `BlobStoreInitialized`: Combines blob store config with initialized BlobStore interface
- `BlobStoreMap`: Map of blob store ID strings to initialized stores
- `CopyResult`: Result of blob copy operation with state tracking

## Key Functions

- `MakeBlobStores`: Creates all blob stores from directory layout and config
- `MakeBlobStore`: Factory for individual blob stores (local, SFTP, pointer)
- `CopyBlobIfNecessary`: Smart blob copying with existence checking
- `MakeRemoteBlobStore`: Creates remote blob store from config

## Features

- Supports local hash-bucketed and remote SFTP blob stores
- Pointer-based blob store references for indirection
- Multi-store management with XDG override support
- Copy verification and state tracking

# sku

Core SKU (Stock Keeping Unit) types representing versioned objects with pool management.

## Key Types

- `Transacted`: Primary versioned object with ObjectId, Metadata, and external state
- `CheckedOut`: Object checked out to working copy with internal/external pair
- `FSItem`: Filesystem item representing checked out files
- `Proto`: Prototype object for creation
- `Conflicted`: Merge conflict representation

## Key Interfaces

- `TransactedGetter`: Access to Transacted pointer
- `ExternalLike`: External object with state and repo info
- `Config`: SKU configuration interface

## Features

- Pool-based memory management with GetTransactedPool()
- Resetter pattern for efficient object reuse
- Heap implementations for cursor and TAI-ordered access
- Collection types (List, Index, WorkingList)
- Store and query interfaces

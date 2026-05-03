# object_probe_index

Page-based binary index for fast object location lookups by digest.

## Purpose

Memory-mapped page index for locating objects by SHA digest with concurrent access.

## Key Types

- `Index`: 256-page index (1 digit bucketing) with hash type support
- `Loc`: Location record pointing to object position in storage

## Features

- Single-digit bucketing (256 pages) for fast digest-based lookups
- Support for duplicate or unique digest enforcement
- ReadOne for single location, ReadMany for duplicate digest scenarios
- Parallel page flushing with wait groups
- Hash format conversion for cross-format lookups

# zettel_id_index

Index for managing unique Zettel IDs with persistence and collision detection.

## Key Types

- `Index`: Interface for creating, tracking, and peeking Zettel IDs

## Key Functions

- `MakeIndex`: Factory; currently always returns the v1 implementation
- `CreateZettelId`: Generates new unique Zettel ID
- `AddZettelId`: Registers existing Zettel ID to prevent collisions
- `PeekZettelIds`: Previews next N available Zettel IDs

## Features

- Two implementations:
  - **v1 (bitset-based, active):** persists via
    `encoding.BinaryMarshaler` on a `collections.Bitset`.
  - **v0 (map-based, dormant):** persists via a raw big-endian uint32
    stream (`uint32 count` + `count` × `uint32` keys via
    `binary.Write`). Gated behind a hardcoded `if true { v1 } else
    { v0 }` switch in `main.go`'s factory; kept for the migration
    path and for reading legacy on-disk indexes.
- Thread-safe ID generation with mutex protection
- Collision detection for existing IDs
- Configurable predictable vs random ID selection

## Persistence format

Neither implementation uses gob. v0 writes a hand-rolled uint32 stream;
v1 writes the bitset's `MarshalBinary` output. Both store at
`directoryLayout.FileCacheObjectId()` (resolves to
`DirDataIndex("object_id")` under `.dodder/local/share/index/`).
The read path includes a sanity check that detects stale on-disk
formats and rebuilds the pool from scratch rather than risking a
giant allocation (see `v0/main.go` comment referencing #68).

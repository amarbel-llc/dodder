# zettel_id_index

Index for managing unique Zettel IDs with persistence and collision detection.

## Key Types

- `Index`: Interface for creating, tracking, and peeking Zettel IDs

## Key Functions

- `MakeIndex`: Factory creating v0 or v1 index implementation
- `CreateZettelId`: Generates new unique Zettel ID
- `AddZettelId`: Registers existing Zettel ID to prevent collisions
- `PeekZettelIds`: Previews next N available Zettel IDs

## Features

- Two implementations: v0 (map-based) and v1 (bitset-based)
- Persistent storage via gob encoding
- Thread-safe ID generation with mutex protection
- Collision detection for existing IDs
- Configurable predictable vs random ID selection

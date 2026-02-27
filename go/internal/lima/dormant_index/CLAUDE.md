# dormant_index

Tag-based index for marking objects as dormant (inactive/archived).

## Purpose

Tracks tags that mark objects as dormant, persisted to disk for query filtering.

## Key Types

- `Index`: Tag collection with change tracking and persistence

## Features

- Add/remove dormant tags with change logging
- Check if SKU objects contain dormant tags
- Binary serialization with uint16 length prefixes
- Flush only when changes exist (skip dry runs)
- Load from disk with EOF tolerance

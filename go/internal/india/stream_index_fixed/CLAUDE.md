# stream_index_fixed

Fixed-length binary stream index for O(1) random access to SKU objects.

## Key Types

- `Index`: Top-level index managing pages, probes, and overflow
- `binaryEncoder`: Encodes SKU objects into fixed-width entries
- `binaryDecoder`: Decodes fixed-width entries back to SKU objects
- `pageWriter`: Flushes page changes with append-only or full-rewrite modes
- `overflowWriter`: Sidecar file for entries exceeding inline capacity

## Features

- Fixed-width entries (EntryWidth bytes) for O(1) offset arithmetic
- Inline/overflow split: small entries stored inline, large entries spill to sidecar
- Sigil-based filtering with in-place sigil updates via WriterAt
- Page-partitioned storage with per-page write locks
- Probe index for random access by object ID

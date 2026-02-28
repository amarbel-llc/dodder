# stream_index

Binary stream index for fast object serialization and indexing.

## Key Types

- `binaryEncoder`: Encodes SKU objects to binary format
- `binaryDecoder`: Decodes binary format to SKU objects
- `binaryField`: Field-level binary encoding with key bytes

## Features

- Compact binary format with content length headers
- Field-based encoding using key_bytes constants
- Supports all object metadata (blob, type, tags, TAI, signatures)
- Sigil-based filtering and updates
- WriterAt support for in-place sigil updates

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

## Adding a New Metadata Field to the Binary Codec

New metadata fields on `delta/objects/metadata` are **not** automatically
serialized. Without explicit codec support, the field will be populated during
commit (e.g., by reference discovery) but lost on the next read from the store.

Four files must be updated:

1. **`internal/_/key_bytes/main.go`** — Add a new `Binary` constant (pick an
   unused ASCII byte). Run `go generate ./internal/_/key_bytes/` to regenerate
   `binary_string.go`.

2. **`binary_field.go`** — Add the new key to `binaryFieldOrder`. Position
   matters: fields are written/read in this order.

3. **`binary_encoder.go`** — Add a `case` in `writeFieldKey` for the new key.
   Follow the pattern of the nearest existing field type (e.g., `References` for
   collection fields, `Type` for single-value fields with `markl.Lock`).

4. **`binary_decoder.go`** — Add a matching `case` in `readFieldKey`. The
   decoder must mirror the encoder's wire format exactly.

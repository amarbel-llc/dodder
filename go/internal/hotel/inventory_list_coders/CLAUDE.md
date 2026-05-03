# inventory_list_coders

Codecs for encoding/decoding inventory lists in various formats.

## Key Types

- `coder`: Base coder with pre/post encoding hooks
- `Closet`: Coder registry for format selection

## Formats

- Doddish: Native text format
- JSON V0: JSON serialization format

## Features

- Buffered encoding/decoding with hooks
- Before-encoding and after-decoding callbacks
- Format-agnostic interface for inventory list serialization

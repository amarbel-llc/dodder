# markl

Markl identifier system with binary and text encoding for content-addressable storage.

## Key Types

- `Id`: Markl identifier with text/binary marshaling support
- `Slice`: Collection of markl IDs with streaming read support
- Pool management via `GetId()` and `PutId()`

## Features

- Binary coding for compact storage
- Text format support for human readability
- Thread-safe ID pooling
- Lock mechanism for concurrent access
- Streaming slice reader from newline-delimited text

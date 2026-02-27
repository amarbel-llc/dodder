# markl_io

I/O utilities for markl format with hash computation.

## Key Types

- `readCloser`: Reader that computes hash while reading
- `writeCloser`: Writer that computes hash while writing

## Features

- Tee reading/writing with hash computation
- Implements `interfaces.BlobReader`

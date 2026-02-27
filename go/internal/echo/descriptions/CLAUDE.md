# descriptions

Text description handling for zettel and object metadata with multi-line support.

## Key Types

- `Description`: Wrapper for text descriptions with set/unset tracking
- Supports automatic space-joining when setting multiple values
- Implements text/binary marshaling for persistence

## Features

- Multi-line description parsing from doddish format
- Automatic whitespace normalization
- Newline stripping for display (`StringWithoutNewlines`)
- Integration with doddish scanner for structured text parsing
- CLI formatting helpers for display output

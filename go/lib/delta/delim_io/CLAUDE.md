# delim_io

Delimited reader for parsing delimiter-separated data streams.

## Key Types

- `Reader`: Buffered reader that splits on delimiter bytes
- `reader`: Pool-managed implementation with segment counting

## Features

- Delimiter-based string/bytes reading
- Key-value pair parsing with separators
- EOF tracking and segment counting
- Copy with prefix on delimiter operations
- Pool-based reader recycling

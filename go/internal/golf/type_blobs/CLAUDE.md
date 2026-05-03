# type_blobs

Blob representations for type object metadata with file handling configuration.

## Key Types

- `Blob`: Interface for type metadata (extension, MIME type, formatters)
- `TomlV0`, `TomlV1`: TOML-formatted type configurations
- `UTIGroup`: Uniform Type Identifier grouping for formatters

## Features

- File extension and MIME type configuration
- Binary file detection
- Vim syntax type hints
- Formatter configuration with output format specs
- Lua hooks for type-specific scripting

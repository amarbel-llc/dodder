# object_metadata_fmt_triple_hyphen

Triple-hyphen metadata format parser and formatter for object serialization.

## Key Types

- `Formatter`: Interface for formatting metadata to writer
- `Parser`: Interface for parsing metadata from reader
- `FormatterFamily`: Collection of formatters for different output modes
- `Format`: Complete format with formatter family and parser
- `FormatterContext`: Context for formatting with options and encoder context
- `ParserContext`: Context for parsing with decoder context

## Features

- Multiple formatting modes: BlobPath, InlineBlob, MetadataOnly, BlobOnly
- Triple-hyphen delimited metadata format (---\n...metadata...\n---)
- Streaming metadata parsing and formatting
- Blob writer integration for handling embedded content
- Support for text and binary object representations

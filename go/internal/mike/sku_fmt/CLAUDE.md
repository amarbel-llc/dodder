# sku_fmt

Formatters and printers for SKU (transacted object) serialization and output.

## Key Types

- `JSON`: JSON representation with bidirectional conversion (to/from `sku.Transacted`)
- `JSONMCP`: MCP protocol extension with URI and related URIs
- `PrinterComplete`: Concurrent printer for streaming object output
- `formatterTypFormatterUTIGroups`: Formatter for type-based UTI groups

## Features

- JSON serialization with blob content, metadata, and signatures
- dodder:// URI scheme for object references
- Concurrent buffered printing with channel-based processing
- Type formatter resolution and UTI group extraction
- Metadata-only and full blob formatting modes

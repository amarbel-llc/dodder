# sku_fmt

Formatters and printers for SKU (transacted object) serialization and output.

## Key Types

- `PrinterComplete`: Concurrent printer for streaming object output
- `formatterTypFormatterUTIGroups`: Formatter for type-based UTI groups

## Features

- Concurrent buffered printing with channel-based processing
- Type formatter resolution and UTI group extraction
- Metadata-only and full blob formatting modes

JSON serialization lives in `internal/golf/sku_json_fmt` (`Transacted`/`MCP`),
not here — this package's own `JSON`/`JSONMCP` types were dead code (zero
call sites, a strict subset of `sku_json_fmt.Transacted`/`MCP`) and were
removed.

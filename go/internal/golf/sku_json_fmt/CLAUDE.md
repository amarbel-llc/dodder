# sku_json_fmt

JSON serialization format for SKU transacted objects.

## Purpose

Bidirectional JSON conversion for SKU objects with optional blob content inclusion.

## Key Types

- `Transacted`: JSON representation with all SKU metadata fields
- `Lock`: Type lock information for version control
- `JsonWithUrl`: `Transacted` + embedded bookmark `TomlBookmark`

## Re-exported from `foxtrot/sku_bookmark_blobs`

The bookmark blob struct lives in `internal/foxtrot/sku_bookmark_blobs` (tommy
v0.4.3 codegen-isolation split). `TomlBookmark`, `TomlBookmarkDocument`, and
`DecodeTomlBookmark` are re-exported here so external callers keep using
`sku_json_fmt.*`.

## Features

- Convert SKU objects to/from JSON with full metadata
- Optional blob string embedding for complete object serialization
- Supports both TAI and RFC3339 date formats
- Handles repo public key and signature fields
- Tag set conversion with automatic expansion

# sku_json_fmt

JSON serialization format for SKU transacted objects.

## Purpose

Bidirectional JSON conversion for SKU objects with optional blob content inclusion.

## Key Types

- `Transacted`: JSON representation with all SKU metadata fields
- `Lock`: Type lock information for version control

## Features

- Convert SKU objects to/from JSON with full metadata
- Optional blob string embedding for complete object serialization
- Supports both TAI and RFC3339 date formats
- Handles repo public key and signature fields
- Tag set conversion with automatic expansion

# blob_library

Generic blob storage and retrieval with type-safe pooling.

## Purpose

Provides pooled, typed access to blob content with automatic format conversion.

## Key Types

- `Library[BLOB, BLOB_PTR]`: Generic blob store with pooling and format encoding/decoding

## Features

- Type-safe blob retrieval with automatic deserialization
- Pool-based memory management for blob objects
- SHA verification during blob reads
- Generic interface supports any blob type with custom formatters

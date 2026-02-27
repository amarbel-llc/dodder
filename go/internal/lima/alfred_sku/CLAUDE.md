# alfred_sku

Alfred workflow output formatter for Dodder SKU objects.

## Purpose

Converts Dodder objects (Zettels, Tags) into Alfred JSON format for macOS Alfred Workflow integration.

## Key Types

- `Writer`: Main formatter that converts SKU objects to Alfred items with searchable metadata

## Features

- Formats Zettels with title, subtitle, tags, and searchable matches
- Supports tag expansion and type-based filtering
- Handles errors with formatted Alfred items
- Provides abbreviated IDs and copy-paste support

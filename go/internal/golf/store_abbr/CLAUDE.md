# store_abbr

ID abbreviation index for compressing and expanding object and blob IDs.

## Purpose

Maintains tridex-based index for abbreviating Zettel IDs and Markl IDs (blob digests).

## Key Types

- `indexAbbr`: Cached abbreviation index with lazy loading and flush-on-change
- `indexZettelId`: Specialized abbreviator for Zettel head/tail components
- `indexNotZettelId`: Generic abbreviator for Markl IDs

## Features

- Abbreviate Zettel IDs using head/tail tridex indexes
- Abbreviate blob digests (Markl IDs) using tridex
- Expand abbreviated IDs back to full form
- Tracks seen IDs per genre (Zettel, Tag, Type, Repo)
- Gob-encoded persistence with lazy read and change tracking

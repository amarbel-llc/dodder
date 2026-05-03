# tridex

Thread-safe trie with abbreviation and expansion support for ID compression.

## Key Types

- `Tridex`: Thread-safe trie with add/remove and abbreviate/expand
- `node`: Internal trie node with children map

## Features

- String abbreviation to shortest unique prefix
- Expansion from abbreviation back to full string
- Contains/ContainsExactly queries
- Mutable clone support
- Gob serialization

# queries

Query system for matching and filtering SKU objects with genre optimization.

## Key Types

- `Query`: Main query container with optimized per-genre queries
- `expSigilAndGenre`: Expression combining sigil filters and genre constraints
- `DormantCounter`: Tracks matched dormant/archived objects

## Features

- Genre-optimized query storage and matching
- Internal and external object ID support
- Sigil-based filtering (latest, history, hidden, external)
- Default query composition
- Match-on-empty behavior configuration

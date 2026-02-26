# object_id_provider

Bidirectional mapping between zettel IDs and integer coordinates for indexing.

## Key Types

- `Provider`: Dual provider with Yin/Yang ID lists loaded from repository
- `provider`: Internal slice-based ID lookup and reverse mapping

## Files

- `Yin`: Primary zettel ID list
- `Yang`: Secondary zettel ID list

## Features

- `MakeZettelIdFromCoordinates`: Convert coordinate to zettel ID
- `ZettelId`: Reverse lookup from ID to coordinate
- Thread-safe with mutex locking
- Used by stream index for efficient zettel relationship tracking

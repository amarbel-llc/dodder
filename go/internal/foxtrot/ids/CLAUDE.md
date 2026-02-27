# ids

Core object identifier system with genre-aware parsing and validation.

## Key Types

- `ObjectId`: Universal identifier with genre, left/right parts, and virtual flag
- `Tag`: Tag identifier (e.g., `project`, `-hidden`, `%virtual`)
- `Type`: Type identifier with `!` prefix (e.g., `!zettel`)
- `Id`: Interface for all identifier types with seq conversion

## Supported Formats

- Tags: `tag-name`, `-dependent-tag`, `%virtual-tag`
- Types: `!type-name`
- Repos: `/repo-name`
- Zettels: `prefix/id-suffix`
- Blobs: `@digest`, `purpose@digest`
- Inventory Lists: `sec.asec` (TAI timestamp format)

## Key Functions

- `MakeObjectId`: Parse string to ObjectId
- `SetWithGenre`: Set ID with genre validation
- `ValidateSeqAndGetGenre`: Determine genre from doddish sequence
- `Equals`, `Contains`: Identifier comparison

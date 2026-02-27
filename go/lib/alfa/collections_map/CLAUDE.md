# collections_map

Type-safe generic map wrapper implementing Collection interface.

## Key Types

- `Map[KEY, VALUE]` - generic map wrapper implementing interfaces.Collection

## Key Methods

- `All()` - returns Seq[KEY] iterator over keys
- `AllPairs()` - returns Seq2[KEY, VALUE] iterator over key-value pairs
- `Get/Set()` - access map entries
- `Reset()` - clears map
- `ResetWith/ResetWithSeq()` - replaces map contents

## Features

- Iterator support via All() and AllPairs()
- Collection interface implementation for consistent API

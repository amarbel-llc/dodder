# collections_value

Generic value-based collection types with string keying.

## Key Types

- `Set[T]`: Immutable set with string-keyed elements
- `MutableSet[T]`: Mutable set with add/delete/reset operations

## Features

- Generic value storage (not pointers)
- Factory functions for set creation from sequences/slices
- Stringer-based default keying
- Gob serialization support

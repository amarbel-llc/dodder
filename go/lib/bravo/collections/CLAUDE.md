# collections

Core collection types including bitsets and tridex sets.

## Key Types

- `Bitset`: Thread-safe bit array with on/off counting
- `TridexSet`: Set backed by tridex for prefix operations

## Features

- Growable bitset with binary serialization
- Set operations (add, delete, contains)
- Iteration over on/off bits
- Thread-safe operations via mutex

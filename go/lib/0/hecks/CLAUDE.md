# hecks

Optimized hexadecimal decoding with custom reverse lookup table.

## Functions

- `Decode(dst, src)`: Fast hex decoding using pre-computed reverse hex table

## Implementation Details

- Uses `reverseHexTable` constant for O(1) character-to-value lookup
- Appends decoded bytes directly to dst slice
- Returns number of bytes decoded and any errors
- Validates input for proper hex characters and even length

Alternative to encoding/hex.Decode with different performance characteristics.

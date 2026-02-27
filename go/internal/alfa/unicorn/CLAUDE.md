# unicorn

Unicode/rune utility functions.

## Key Functions

- `IsSpace` - alias for unicode.IsSpace
- `Not()` - inverts a rune predicate function
- `CountRune()` - counts consecutive runes at start of byte slice
- `CutNCharacters()` - splits byte slice at N rune boundary

## Features

- UTF-8 aware byte operations
- Rune counting and slicing
- Case utilities in case.go
- Handles multi-byte Unicode correctly

# cmp

Generic comparison functions and types for ordering operations.

## Key Types

- `Result` - comparison result interface (Less/Equal/Greater constants)
- `Func[ELEMENT]` - generic comparison function type
- `Lesser[ELEMENT]` - wraps Func to provide Less() method
- `Equaler[ELEMENT]` - wraps Func to provide Equals() method

## Key Functions

- `MakeFuncFromEqualerAndLessor3EqualFirst()` - creates comparison func checking equality first
- `MakeFuncFromEqualerAndLessor3LessFirst()` - creates comparison func checking less-than first

## Features

- Type-safe generic comparisons
- Converts between comparison functions and Equaler/Lessor interfaces
- Binary search support in search.go

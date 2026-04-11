# object_change_type

Bitfield type for tracking object change categories.

## Type

- `Type`: Byte-based bitfield for change classification

## Constants

- `TypeUnknown`: Default/uninitialized state
- `TypeLatest`: Object represents latest version
- `TypeHistorical`: Object is from history

## Methods

- `Add(...Type)`: Bitwise OR to add change types
- `Del(Type)`: Bitwise clear to remove change type
- `Contains(Type)`: Check if type is present
- `ContainsAny(...Type)`: Check if any of the types are present

Uses go:generate stringer for string representation.

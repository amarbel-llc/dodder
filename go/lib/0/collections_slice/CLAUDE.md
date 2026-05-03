# collections_slice

Type-safe generic slice wrapper with rich collection operations.

## Key Types

- `Slice[ELEMENT]` - generic slice implementing interfaces.Collection

## Key Functions

- `Make/MakeFromSeq/MakeFromSlice()` - constructors
- `MakeWithLen/MakeWithCap()` - pre-allocated constructors
- `Collect()` - creates slice from Seq iterator

## Key Methods

- `All()` - returns Seq iterator
- `First/Last/Any()` - element access
- `Append/Merge/Insert()` - modification
- `Reset/ResetWith()` - clearing and replacement
- `Clone()` - deep copy
- `Shift/ShiftInPlace()` - offset operations

## Features

- Sort support in sort.go
- Collection interface implementation

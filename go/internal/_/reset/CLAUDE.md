# reset

Generic reset utilities for clearing and reusing data structures.

## Types

- `FuncReset[T]`: Function type for resetting element to zero state
- `FuncResetWith[T]`: Function type for copying src element to dst element
- `resetter[T]`: Struct wrapping reset functions

## Functions

- `MakeResetter[T](reset, resetWith)`: Create resetter with custom reset logic
- `Map[K, V](map)`: Clear map or create if nil, reusing allocation
- `Slice[T](slice)`: Truncate slice to length 0, reusing capacity

## Methods

- `Reset(element)`: Reset element to zero state
- `ResetWith(dst, src)`: Copy src to dst

Used for efficient object reuse without allocations, particularly with object pools.

# equality

Generic equality comparison functions for ordered types.

## Functions

- `MapsOrdered[K, V]()`: Deep equality for maps with ordered key/value types
- `SliceOrdered[V]()`: Deep equality for slices with ordered element types

Uses Go generics with constraints.Ordered to provide type-safe equality checks.

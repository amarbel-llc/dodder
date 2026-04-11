# coordinates

Coordinate system for Zettel IDs using triangular number mapping.

## Key Types

- `ZettelIdCoordinate`: Two-dimensional coordinate (Left, Right) representing a Zettel ID
- `Int`: uint32 alias for coordinate values
- `Float`: float32 alias for calculations

## Key Functions

- `Extrema(n)`: Calculate coordinate range boundaries for a given level
- `SetCoordinates()`, `SetInt()`: Set coordinates from various inputs
- `Id()`: Convert coordinates back to linear Zettel ID

Maps linear Zettel IDs to 2D coordinates using triangular number sequences.

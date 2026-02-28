# objects

Core metadata and object model for Dodder's content-addressable storage.

## Key Types

- `metadata`: Primary implementation of object metadata
- `Metadata`, `MetadataMutable`: Interfaces for reading and writing metadata
- `Tag`, `TagSet`, `TagSetMutable`: Tag management types
- `Type`, `TypeMutable`: Object type handling
- `ContainedObjects`: Collection of nested objects
- `Index`: Binary indexing for fast object access

## Features

- Description, tags, type, and timestamps (TAI)
- Content-addressable digest management (blob, object, signatures)
- Tag operations: add, remove, subtract, fast operations
- Type and tag locking mechanisms
- Index management for tag paths
- Builder pattern for object construction
- Resetter pattern for object pooling

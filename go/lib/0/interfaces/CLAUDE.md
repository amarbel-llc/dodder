# interfaces

Core interface definitions used throughout the Dodder codebase.

## Key Interface Categories

- **Object I/O**: `ObjectIOFactory`, `ObjectReaderFactory`, `ObjectWriterFactory`, `BlobReader`, `BlobWriter`
- **Error Handling**: `ErrorOneUnwrapper`, `ErrorManyUnwrapper`, `ErrorHiddenWrapper`
- **IDs and Keys**: `ObjectId`, `MarklIdGetter`, keyers for various object types
- **Storage**: `BlobStore`, workspace, directory, lock interfaces
- **Collections**: Generic collection interfaces, iterators, pools
- **Commands**: CLI command interfaces and context state
- **Values**: Value interfaces for configuration and objects

## Generic Types

- `Ptr[T]`: Generic pointer constraint for type parameters

This package provides the contract layer that enables loose coupling between modules.
See individual files for specific interface groups.

# store_version

Store format versioning for data compatibility.

## Key Types

- `Version`: Store version number (wrapper around Int)
- `Getter`: Interface for version retrieval

## Key Versions

- `V10`-`V14`: Defined version constants
- `VCurrent`: Current active version (V12)
- `VNext`: Next planned version (V13)

## Features

- Version comparison (Less, Greater, Equal)
- String parsing with validation
- Future version detection and error handling

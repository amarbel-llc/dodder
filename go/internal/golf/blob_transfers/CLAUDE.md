# blob_transfers

Blob import and transfer operations between blob stores.

## Purpose

Handles copying blobs between different blob stores with progress tracking and validation.

## Key Types

- `BlobImporter`: Manages blob import operations with copy results tracking
- `Counts`: Statistics for succeeded, ignored, and failed transfers

## Features

- Conditional blob copying (only if missing at destination)
- Multi-store import support
- Progress reporting with time-based UI updates
- Copy result delegation for tracking missing blobs
- Object blob-closure copy (`ImportObjectBlobClosure`: an object's own blob plus
  its field-level blob references), used by the transform pipeline's
  self-containment copy (dodder#392)

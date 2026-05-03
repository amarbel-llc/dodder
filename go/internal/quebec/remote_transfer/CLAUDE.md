# remote_transfer

Handles importing SKU objects from remote sources into the local repository with blob transfer, conflict detection, and inventory list expansion.

## Key Types

- `importer`: Main import orchestrator with blob transfer, merge conflict handling, and commit logic
- `committer`: Handles committing imported objects to the store
- `deduper`: Manages deduplication during import operations

## Features

- Import: Imports individual transacted SKU objects with blob transfer and merge conflict detection
- ImportSeq: Batch imports sequences of objects with error collection and missing blob tracking
- ImportBlobIfNecessary: Transfers blob data from remote store to local store
- importInventoryList: Recursively imports inventory lists and their contained objects
- importLeaf: Imports individual objects with finalization, signing, and merge conflict resolution

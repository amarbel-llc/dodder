# store_fs

Filesystem store for managing checked out objects in the working directory.

## Key Types

- `Store`: Main filesystem store managing checked out objects
- `FileEncoder`: Encodes objects to filesystem representation
- `FSItem`: Filesystem item with object/blob/conflict file descriptors
- `CheckoutOptions`: Options for checkout operations
- `DeleteCheckout`: Operation for removing checked out files

## Features

- Reads and writes objects to working directory
- Tracks probably/definitely checked out items via dirInfo
- Supports checkout, merge, and diff operations
- Manages file deletions with user/internal separation
- Converts between FSItem and external object representations
- Pattern-based file discovery for queries

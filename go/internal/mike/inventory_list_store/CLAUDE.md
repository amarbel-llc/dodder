# inventory_list_store

Manages creation, storage, and retrieval of inventory lists that track collections of SKU objects.

## Key Types

- `Store`: Main inventory list store implementing `sku.InventoryListStore`
- `blobStoreV1`: Internal blob store for reading/writing inventory list objects

## Features

- Create inventory lists from working lists with automatic finalization
- Write inventory list blobs containing collections of SKUs
- Read and iterate over all stored inventory lists
- Support sorted iteration and content retrieval by blob digest
- Log-based persistence with atomic operations

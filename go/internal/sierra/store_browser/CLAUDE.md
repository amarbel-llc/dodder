# store_browser

External store adapter for browser items (tabs/bookmarks) from the chrest browser extension, mapping them to toml-bookmark SKUs.

## Key Types

- `Store`: Main external store implementation for browser items
- `Item`: Wrapper around browser_items.Item with SKU conversion methods
- `cache`: Tab cache with timestamp tracking for invalidation

## Features

- Initialize: Connects to chrest browser proxy and loads all browser items
- CheckoutOne: Converts transacted toml-bookmark objects to browser tab checked-out items
- QueryCheckedOut: Queries browser items matching URL/tab ID against indexed objects
- SaveBlob: Serializes browser items as TOML bookmark blobs
- Flush: Persists cached browser state changes
- GetObjectIdsForString: Resolves browser item IDs to external object IDs

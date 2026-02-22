# Design: Blob Store Selection for `madder write`

## Problem

`madder write` always writes to the default blob store (first alphabetically).
The inline arg-based switching at line 106-114 exists but is broken: it hardcodes
an SFTP-specific debug log that crashes for non-SFTP stores.

## Solution

### 1. `--blob-store` flag

Comma-separated blob store IDs:

```
madder write --blob-store ~.default,~.backup file.txt
madder write --blob-store ~.remote file1.txt file2.txt
```

- Without the flag: uses default (current behavior)
- Single value: targets one store
- Multiple values: wraps stores in existing `Multi` blob store for simultaneous writes

### 2. Fix inline arg-based switching

When an arg doesn't exist as a file and parses as a `blob_store_id`, switch the
active store for subsequent files. Remove the broken SFTP-specific debug log.

Flag sets initial store(s); inline args override for subsequent files.

### 3. Implementation

Single file change: `lima/commands_madder/write.go`

- Add `BlobStoreIds string` field to `Write` struct with `--blob-store` flag
- In `Run()`: parse comma-separated IDs, look up each via `envBlobStore.GetBlobStore()`,
  wrap multiples in `Multi`
- Fix inline switching: remove SFTP assumption, use generic debug log
- Remove TODO comment

No changes to `Multi`, `BlobStoreEnv`, or `blob_store_id` — infrastructure exists.

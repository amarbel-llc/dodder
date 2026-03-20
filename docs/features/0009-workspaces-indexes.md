-----

## status: exploring
date: 2026-03-20
promotion-criteria:

# External Object Index

## Problem Statement

Queries that include the `.` sigil (`SigilExternal`) trigger a full filesystem
walk on every invocation. `store_fs.dirInfo.processRootDir()` walks the working
directory, stats every file, reconstructs `FSItem` objects from path conventions,
and looks each up in the stream index — all of which is discarded after the
query completes. Nothing is cached between queries. For a workspace with 500
checked-out files, `dodder show .z` does 500+ stat calls, 500+ filename parses,
and 500+ index lookups every time.

The `SigilExternal` bit (`1 << 2`, mapped to `.`) is defined in `ids.Sigil` and
wired into query parsing and evaluation, but no `.`-sigiled entry is ever written
to the stream index. The external state exists only at runtime, built fresh by
`store_fs` on each query.

Git solved this problem with the DIRC (Directory Cache) index format. `.git/index`
stores stat data (mtime, size, inode) alongside blob SHAs for each tracked file.
`git status` reads the index, compares stat data against the filesystem, and only
re-hashes files where stat data changed. Most comparisons are a single stat
syscall against cached metadata — no content hashing needed.

A second problem compounds the first. The data carrier for external state is the
`string_format_writer.Field` type — a struct designed for CLI box-format rendering
(`Key`, `Value`, `ColorType`, `Separator`, `DisableValueQuotes`, `NoTruncate`,
`NeedsNewline`). `store_fs.WriteFSItemToExternal` writes fields with keys
`"object"`, `"blob"`, `"conflict"`, `"lockfile"` and values being relative file
paths. `ReadFSItemFromExternal` reconstructs `FSItem` objects by switching on
these string keys. This coupling means the on-disk checkout representation is
encoded through a type that was never intended to carry structured data, making
it fragile and impossible to serialize to the binary index.

Pluggable checkout stores (FDR-0007) will make this worse. A CalDAV-backed
workspace needs to cache VTODO UIDs, ETags, and sync tokens per object. These
have nothing to do with filesystem paths or display fields. Without a proper
external state type, every new checkout store would need its own ad-hoc runtime
cache and its own filesystem walk equivalent — there is no shared infrastructure
for persisting and validating external state.

## Design

### External State on the Object

Replace the `Field`-based external state representation with a dedicated type on
the object’s metadata. This type knows how to serialize to the binary stream
index and how to validate itself against the actual external store.

```go
// ExternalState holds checkout-store-specific cache data for an object.
// Polymorphic by store type — the StoreType byte selects the encoding.
type ExternalState struct {
    StoreType  StoreType       // fs, caldav, webdav, etc.
    Tracked    external_state.State
    Entries    []ExternalEntry // one per FD (object, blob, lockfile)
}

// ExternalEntry is one file/resource tracked by the external state.
type ExternalEntry struct {
    Role    EntryRole  // object, blob, conflict, lockfile
    Path    string     // relative path (fs) or external UID (caldav)
    ModTime int64      // nanoseconds since epoch (fs) or 0 (caldav)
    Size    int64      // bytes (fs) or 0 (caldav)
    Digest  markl.Id   // content digest at last sync
    ETag    string     // (caldav only) ETag at last sync
}

type StoreType byte

const (
    StoreTypeFS     StoreType = 'f'
    StoreTypeCalDAV StoreType = 'c'
    StoreTypeWebDAV StoreType = 'w'
)

type EntryRole byte

const (
    EntryRoleObject   EntryRole = 'o'
    EntryRoleBlob     EntryRole = 'b'
    EntryRoleConflict EntryRole = 'c'
    EntryRoleLockfile EntryRole = 'l'
)
```

### Where ExternalState Lives

Today `metadata.index` (the `objects.index` struct) has:

- `Dormant` — whether the object is hidden
- `ImplicitTags` — computed tag set
- `TagPaths` — tag hierarchy
- `Comments` — hyphence comments
- `Fields` — the shoehorned `string_format_writer.Field` slice

`Fields` is the only member used for external state. The proposal: add
`ExternalState` as a peer of `Fields` on the index struct, and stop writing
external path data to `Fields`.

```go
type index struct {
    Dormant        values.Bool
    ImplicitTags   TagSetMutable
    TagPaths       tag_paths.Tags
    Comments       collections_slice.Slice[string]
    Fields         collections_slice.Slice[Field] // retained for non-external uses
    ExternalState  ExternalState                  // new
    keyValues
}
```

`FSItem` reconstruction then reads from `ExternalState.Entries` instead of
switching on `Field.Key` strings. The `ReadFSItemFromExternal` /
`WriteFSItemToExternal` methods on `store_fs.Store` become thin wrappers around
`ExternalState` rather than Field parsers.

### Persisting to the Stream Index

Add a new binary key:

```go
// in key_bytes
ExternalState = Binary('e')
```

Add `key_bytes.ExternalState` to `binaryFieldOrder`. The encoder writes:

1. `StoreType` (1 byte)
1. `Tracked` state (1 byte)
1. Entry count (uint8)
1. For each entry:
- `Role` (1 byte)
- Path/UID: length-prefixed string (uint16 + bytes)
- ModTime (int64, 8 bytes, big-endian; 0 if not applicable)
- Size (int64, 8 bytes, big-endian; 0 if not applicable)
- Digest: `markl.Id` binary encoding (existing format)
- ETag: length-prefixed string (uint16 + bytes; empty if not applicable)

The decoder reconstructs `ExternalState` from these bytes. Entries without
external state (objects never checked out) have no `ExternalState` field in the
binary index — same as how objects without tags have no `Tag` field.

### The Sigil Bit

`.`-sigiled entries are written to the stream index alongside `:` and `+`
entries. The `Sigil` byte on the binary index entry carries the `SigilExternal`
bit. Queries for `.z` filter by the External bit, reading only entries with
external state — no filesystem walk needed.

An object can carry multiple sigil bits simultaneously. A checked-out object
has both `SigilLatest` (it’s in the current inventory) and `SigilExternal`
(it has a checkout entry). The page reader’s query matching already supports
bitwise sigil filtering.

### Cache Validation

When a `.`-sigiled entry is read from the index, the checkout store validates
its cache before returning it to the query. For `store_fs`:

1. Stat the path(s) in `ExternalState.Entries`
1. Compare mtime and size against cached values
1. If match: entry is clean — return as-is
1. If mismatch: entry is stale — re-read the file, recompute digest, update the
   cache entry
1. If file missing: entry is deleted externally — mark for user decision

For `store_caldav` (FDR-0007):

1. Compare cached ETag against server (lightweight HEAD request or sync-token
   check)
1. If match: clean
1. If mismatch: stale — fetch updated resource
1. If 404: deleted externally

Validation is lazy per-entry, not eager for the whole index. A query that reads
only one object validates only that entry. A full `.z` query validates all
entries but can parallelize the stat/HEAD calls.

#### Racy-Stat Detection

If a file’s mtime matches the cached mtime but is within the same second as the
last index write, the file may have been modified after the stat was cached. Git
calls this “racy git.” The fix: if `mtime == cached_mtime` AND
`mtime >= index_write_time`, re-hash the file content and compare against the
cached digest. This adds one hash per racy file per query, which is rare in
practice.

### Untracked File Discovery

DIRC only caches known checked-out files. New files created outside dodder (a
`.zettel` file dropped into the working directory, a VTODO created in tasks.org)
won’t be in the cache. Discovering them requires a walk (for fs) or a Discover
call (for CalDAV, FDR-0007).

Two modes:

- **`dodder show .z`** — reads only the cache, no walk. Fast. Shows tracked
  external objects. This is the common case.
- **`dodder status`** — reads the cache, then walks the working directory to
  find untracked files. Slower but comprehensive. Updates the cache with any
  newly discovered files.

The distinction maps to `external_state.State`: `Tracked` entries are in the
cache and have been validated. `Untracked` entries were found by a walk but not
yet checked in. `Recognized` entries were found by a walk and matched to an
existing internal object by content or path convention but haven’t been
explicitly checked out.

### FSItem Migration Path

Today `FSItem` is the primary type for external object tracking:

```go
type FSItem struct {
    ExternalObjectId ids.ExternalObjectId
    Object   fd.FD
    Blob     fd.FD
    Conflict fd.FD
    Lockfile fd.FD
    FDs      interfaces.SetMutable[*fd.FD]
}
```

Each `fd.FD` carries `path`, `modTime`, `digest`, and `state`. The
`ExternalState` type proposed above captures the same data (path, mtime, size,
digest per role) but in a serializable, store-agnostic form.

Migration approach:

1. **Add `ExternalState` to the index struct** alongside `Fields`. Both coexist.
1. **Modify `WriteFSItemToExternal`** to write to `ExternalState` instead of
   `Fields`. Keep writing to `Fields` as well for backward compatibility with
   existing consumers (box format output, organize, etc.).
1. **Modify `ReadFSItemFromExternal`** to read from `ExternalState` when present,
   falling back to `Fields` when not.
1. **Persist `ExternalState` to the binary index.** On index read, `.`-sigiled
   entries populate `ExternalState` directly.
1. **Stop writing external paths to `Fields`.** Once all consumers read from
   `ExternalState`, `Fields` no longer carries external state. It returns to its
   original purpose (display-only data) or is removed if no other consumers
   remain.

`FSItem` itself may eventually be replaced by `ExternalState` + accessors, but
that’s a larger refactor. The immediate goal is to get the cache data into the
index so queries are fast.

### Interaction with FDR-0007 (Pluggable Checkout Stores)

FDR-0007 proposes `.dodder/sync-state/` as a filesystem directory with
per-object `.hyphence` files, per-object `.etag` files, and a `manifest.json`.
With the external object index, most of this data moves into the stream index:

- **VTODO UID mapping** → `ExternalState.Entries[0].Path` (the “path” for CalDAV
  is the external UID)
- **ETag** → `ExternalState.Entries[0].ETag`
- **Sync TAI** → the `.`-sigiled entry’s `Tai` field in the binary index
- **Per-object hyphence base** → stored as a blob reference on the entry
  (`- sync-base < @blake2b256-...`), or in a dedicated `SyncBase` field on
  `ExternalState`

The only sync state that remains filesystem-resident is the sync-token (a
server-wide cursor, not per-object) which could live in the workspace config.

This simplification means FDR-0007 doesn’t need to invent a parallel storage
layer — it uses the same index infrastructure as `store_fs`, just with different
`StoreType` and `ExternalEntry` fields.

### Interaction with FDR-0005 (Workspace-as-Repo)

`check-workspace dirty` currently compares inventory list TAI/digest against the
sync baseline. With the external object index, it can also detect external
dirtiness by iterating `.`-sigiled entries and validating their cache:

- **Dirty relative to parent** — existing, based on `SyncTai`/`SyncDigest` in
  workspace config
- **Dirty relative to external store** — new, based on cache validation of
  `.`-sigiled entries (stale stat data for fs, changed ETags for CalDAV)

This gives `check-workspace dirty` an optional `--external` flag (or always
reports both dimensions) without requiring a full checkin.

### Interaction with FDR-0006 (Two-Stage Commit)

Checkin updates both `:` entries (the committed object) and `.` entries (the new
cache state). With two-stage commit, the plan phase can pre-validate all `.`
entries (stat all files / check all ETags) and classify them before acquiring
`LockSmith`. The commit phase updates both entry types atomically.

## Examples

Before (current behavior):

```
$ time dodder show .z    # walks 500 files, 500 stat calls, 500 index lookups
[one/uno @blake2b256-... !md "first zettel" project-alpha]
[one/dos @blake2b256-... !md "second zettel" project-alpha]
...
real    0.8s
```

After (DIRC):

```
$ time dodder show .z    # reads 2 entries from index, 2 stat calls
[one/uno @blake2b256-... !md "first zettel" project-alpha]
[one/dos @blake2b256-... !md "second zettel" project-alpha]
...
real    0.05s
```

Status with untracked detection:

```
$ dodder status
tracked (clean):    48 objects
tracked (modified):  2 objects
  one/uno: blob mtime changed
  one/dos: object file missing
untracked:           3 files
  notes/draft.md
  attachments/photo.jpg
  inbox/new-task.zettel
```

CalDAV workspace status:

```
$ dodder status
tracked (clean):    45 tasks
tracked (modified):  3 tasks (ETag changed)
  ceroplastes/midtown: status changed externally
  papilio/uptown: description changed externally
  bombyx/downtown: new VALARM added externally
untracked:           1 task (created in tasks.org)
```

## Open Questions

- **Should `.`-sigiled entries be separate index entries or annotations on
  existing `:` entries?** Separate entries double the index size for checked-out
  objects. Annotations (an `ExternalState` field on the `:` entry) are more
  compact but blur the sigil semantics — a `:` entry with external state isn’t
  the same as a `.` entry. The current binary encoder writes the sigil as a
  bitmask, so a single entry can carry both `:` and `.` bits. This suggests
  annotations on existing entries, not separate entries.
- **Should `Fields` be removed entirely once ExternalState is in place?** Fields
  are used by the box format writer for CLI output (the `[one/uno @sha !md ...]`
  format). If external paths are no longer stored in Fields, the box writer needs
  to read from ExternalState instead. Alternatively, Fields could be populated
  from ExternalState at render time rather than at data load time, which is
  cleaner.
- **How large can ExternalState entries get?** For `store_fs`, each entry is
  roughly: 1 (role) + 2 + ~50 (path) + 8 (mtime) + 8 (size) + ~34 (digest) +
  2 + 0 (no ETag) = ~105 bytes per FD. A checked-out object with both object
  and blob files is ~210 bytes of external state. For 1000 checked-out objects,
  that’s ~200KB additional index data — small relative to the existing index
  size.
- **Cache invalidation on `checkout`/`organize`/`edit`.** These commands modify
  files that are tracked in the cache. The cache must be updated after the file
  write completes. This is straightforward for `checkout` (which already writes
  FSItem data) but needs care for `organize` and `edit` (which modify files
  in-place and then checkin).

## Limitations

- Cache validation is per-entry, not atomic. Between validating entry A and
  entry B, the filesystem may have changed. This is inherent to the DIRC model
  and acceptable for the same reasons it’s acceptable in git.
- Racy-stat detection adds one hash per racy file per query. In practice this
  is rare (requires a modification within the same second as the index write).
- Untracked file discovery still requires a filesystem walk. The cache only
  accelerates queries for known checked-out objects.
- The binary index format change requires a reindex for existing repos. This
  can be triggered automatically on first read of an old-format index, same as
  existing format migrations.

## More Information

- [FDR-0007: Pluggable Checkout Stores](0007-pluggable-checkout-stores.md) —
  sync state and cache validation for non-filesystem stores
- [FDR-0005: Workspace-as-Repo](0005-workspace-as-repo.md) — workspace
  isolation and divergence detection
- [FDR-0006: Two-Stage Commit](0006-two-stage-commit.md) — plan-based batch
  commit for checkin
- [Git index format (DIRC)](https://git-scm.com/docs/index-format) — the
  inspiration for stat-based cache validation
- Key files:
  - `_/key_bytes/main.go` — binary index field keys
  - `india/stream_index/binary_encoder.go` — binary index encoder
  - `india/stream_index/binary_field.go` — `binaryFieldOrder`
  - `mike/store_fs/main.go` — `ReadFSItemFromExternal`, `WriteFSItemToExternal`
  - `mike/store_fs/dir_info.go` — filesystem walking (`processRootDir`)
  - `golf/sku/fs_item.go` — `FSItem` type
  - `delta/objects/index.go` — `index` struct with `Fields`
  - `alfa/string_format_writer/fields_writer.go` — `Field` type
  - `_/external_state/main.go` — `Tracked`/`Untracked`/`Recognized` enum
  - `bravo/ids/sigil.go` — `SigilExternal` (`.`)

# Zettel ID Log Design

## Problem

Yin/Yang provider files are mutable flat files outside the store concept. They
require manual editing and a `dodder reindex` when the ID pool is exhausted.
This is inconsistent with dodder's content-addressed, append-only model.

## Design

Replace flat Yin/Yang files with content-addressed delta blobs tracked by a
signed, append-only zettel ID log.

### Version Scheme

- **v0 (implicit):** Current flat Yin/Yang files at `DirObjectId()`. No log, no
  blobs, no signatures. All existing repos use this format today.
- **v1:** Zettel ID log with signed box-format entries referencing
  content-addressed delta blobs.

### Data Model

**Zettel ID log** -- append-only file at `FileZettelIdLog()`. Each entry is a
box-format record encoded via `triple_hyphen_io`. Entry fields:

- Side (Yin or Yang)
- TAI timestamp
- MarklId (SHA digest of the delta blob)
- Word count

**Delta blobs** -- newline-delimited word lists stored in the repo's blob
store. Each blob contains only genuinely new words from that invocation.

**Uniqueness invariant** -- enforced at write time across both sides. Before
writing, load both Yin and Yang providers from the cache. Any candidate word
already present in either side is rejected.

**Provider reconstruction** -- at startup, replay the log in order,
concatenating Yin entries and Yang entries separately. Flat Yin/Yang files
under `DirObjectId()` are a cache rebuilt from the log on reindex. If the log
does not exist (v0 repos), fall back to the flat files.

### Horizontal Versioning

Follows the standard dodder horizontal versioning pattern:

- Type string: `!zettel_id_log-v1`
- `TypeZettelIdLogVCurrent = TypeZettelIdLogV1`
- Architecture A: `CoderToTypedBlob` with `CoderTypeMapWithoutType`
- Future versions add new structs with `Upgrade()` on prior versions

**Interface:**

```go
type Entry interface {
    GetSide() Side
    GetTai() ids.Tai
    GetMarklId() markl.Id
    GetWordCount() int
}
```

### Package Placement

| Package | Path |
|---|---|
| `zettel_id_log` (new) | `go/internal/charlie/zettel_id_log/` |
| `zettel_id_provider` (rename) | `go/internal/charlie/zettel_id_provider/` |
| Type registration | `go/internal/bravo/ids/types_builtin.go` |
| Directory layout | `go/internal/bravo/directory_layout/{main,v3}.go` |
| Genesis | `go/internal/golf/env_repo/genesis.go` |
| Commands | `go/internal/victor/commands_dodder/` |
| `MakeLineSeqFromReader` (new) | `go/lib/charlie/ohio/` |

### Commands

#### `dodder add-zettel-ids-yin` / `dodder add-zettel-ids-yang`

Two commands, one per side. Both accept raw text on stdin.

Pipeline:

1. Read stdin
2. Run `unicorn.ExtractUniqueComponents` on input lines
3. Load both Yin and Yang providers (from cache)
4. Filter candidates: reject any word in either provider
5. If no new words remain, print a message and exit
6. Write the filtered word list as a blob
7. **Acquire repo lock**
8. Append a signed box-format v1 log entry
9. Rebuild the flat file cache for the target side
10. Reset and rebuild the zettel ID availability index

Output: count of new words added and new total pool size
(`len(Yin) * len(Yang)`).

#### `dodder migrate-zettel-ids`

One-time migration from v0 flat files to v1 log. Requires the repo lock.

1. Read existing flat Yin and Yang files from `DirObjectId()`
2. Write each as a blob to the repo's blob store
3. Append two signed v1 log entries (one for Yin, one for Yang)
4. Rebuild flat file caches from the log
5. Rebuild the zettel ID availability index

After migration, the log is the sole source of truth.

### Genesis Changes

`dodder init` with `-yin`/`-yang` flags now accepts raw text (not
pre-processed word lists):

1. Run `ExtractUniqueComponents` on each input
2. Enforce cross-side uniqueness
3. Write each word list as a blob
4. Append two signed v1 log entries
5. Write flat file caches for immediate provider use
6. Reset the zettel ID availability index

### Rollback Strategy

**Dual-architecture:** Log-based loading falls back to flat files when no log
exists. All pre-migration repos continue working unchanged.

**Rollback procedure:** Delete the `zettel_id_log` file; provider reverts to
flat file loading automatically.

**Promotion criteria:** All repos migrated and `just test` passes with
log-only path for one release cycle.

### Changes Summary

**New:**

- Zettel ID log entry interface + v1 struct + coder + type string
- Box-format log reader/writer for the zettel ID log
- `ohio.MakeLineSeqFromReader` iterator
- `dodder add-zettel-ids-yin` command
- `dodder add-zettel-ids-yang` command
- `dodder migrate-zettel-ids` command

**Modified:**

- `genesis.go` -- write blobs + signed log entries instead of `CopyFileLines`
- Provider loading (`zettel_id_provider`) -- replay log if present, fall back
  to flat files
- `ids/types_builtin.go` -- register `TypeZettelIdLogV1`
- Directory layout -- add `FileZettelIdLog()` path
- `complete.bats` -- add new subcommands to completion test

**Renamed:**

- `object_id_provider` -> `zettel_id_provider`

**Unchanged:**

- Coordinate system, zettel ID index, allocation modes, exhaustion handling
- Existing repos continue working until `migrate-zettel-ids` is run

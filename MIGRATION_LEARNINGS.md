# ObjectId3 Migration Learnings

This document captures all learnings from two attempted migration sessions
(objectId2 -> objectId3). Use it to plan a cleaner migration after refreshing
upstream.

## What Was Attempted

The goal: flip `type ObjectId = objectId2` to `type ObjectId = objectId3` in
`go/internal/echo/ids/main.go:51`.

### What objectId2 looks like

```go
type objectId2 struct {
    virtual     bool
    genre       genres.Genre
    middle      byte          // '/', '!', '-', '%', '@', '.'
    left, right catgut.String // e.g. left="one", middle='/', right="dos"
    repoId      catgut.String
}
```

5-field struct using `catgut.String`. Max 255 chars. Encodes identity through
left/middle/right decomposition.

### What objectId3 looks like

```go
type objectId3 struct {
    Genre genres.Genre
    Seq   doddish.Seq // token sequence, e.g. [ident("one"), op('/'), ident("dos")]
}
```

2-field struct using `doddish.Seq` tokens. Simpler, more flexible.

---

## Hidden Dependencies That Must Be Addressed BEFORE Flipping

These are the blocking issues discovered during the migration attempt. They
should ideally be refactored in preparatory PRs on master before the actual
flip.

### 1. GOB Serialization (CRITICAL)

**Problem:** `sku.Transacted` (which contains `ObjectId`) is gob-serialized in
multiple places. Gob encodes struct field layout. Switching ObjectId from
objectId2 to objectId3 changes the binary layout, making existing gob files
unreadable.

**Where gob is used:**
- `romeo/store_config/persist.go:171` - `gob.NewDecoder` loads `config-mutable`
- `romeo/store_config/persist.go:249` - `gob.NewEncoder` saves `config-mutable`
- `juliett/sku/main.go:12` - `gob.Register(Transacted{})`
- `juliett/sku/collections.go:20-21` - `gob.Register` for TransactedSet
- `juliett/sku/keyers.go:17` - `gob.Register` for keyers
- `papa/store_fs/main.go:28` - duplicate `gob.Register(sku.Transacted{})`

**config-mutable is especially tricky:** This gob file is rebuilt during
`loadMutableConfig` and is NOT version-gated. If the struct layout changes, the
file simply fails to decode. The fixture only has `config-seed` (text), and
`config-mutable` is rebuilt at runtime - but it's cached, so stale files from
a previous version will cause failures.

**Recommended approach:** Either:
1. Add a version header to config-mutable and rebuild on version mismatch
2. Switch config-mutable to a non-gob format before the migration
3. Delete config-mutable as part of the store version upgrade path

### 2. Stream Index Binary Format (CRITICAL)

**objectId2 WriteTo format:**
```
genre(1 byte) | [total_len, left_len](2 bytes) | left_bytes | middle(1 byte) | right_bytes
```

**objectId3 needs a WriteTo/ReadFrom:** Currently objectId3 only has
MarshalBinary/UnmarshalBinary (which delegate to Seq's binary marshaler). It
needs WriteTo/ReadFrom that write `genre_byte + seq_binary` for the stream
index `writeFieldWriterTo` call.

**Stream index pages are an INDEX, not source of truth.** The inventory lists
(text format) are the source of truth. So changing the binary format just
requires a version bump and reindex - the pages get rebuilt from inventory lists.

**Key discovery from debugging:** During BATS tests, the stream index binary
decoder was NEVER called during `init-workspace` or `organize`. Objects were
being served from somewhere other than the stream index pages. Investigation
suggests objects come from inventory list parsing during store initialization,
not from pre-existing binary pages.

### 3. Store Version Bump (CRITICAL)

**VCurrent is still V12** in `charlie/store_version/main.go:22`. The plan said
to bump to V13 but it was never done. This means:
- BATS test fixtures are copied from `migration/v12/`
- No reindex is triggered on version change
- Old objectId2 binary data in stream index pages is NOT rebuilt

**The version bump MUST happen** to ensure the stream index is rebuilt and old
binary pages are discarded. Without it, the stream index pages contain objectId2
binary data that objectId3 can't read.

**After bumping:** Run `just test-bats-generate` to regenerate fixtures for the
new version.

### 4. SetLeft/SetRight API (HIGH)

`objectId2` has `SetLeft(string)` and `SetRight(string)` which decompose the
ID into its constituent parts. objectId3 doesn't have these.

**Only external caller:** `sierra/store_browser/item.go:40`:
```go
func (item *Item) GetObjectId() *ids.ObjectId {
    var oid ids.ObjectId
    errors.PanicIfError(oid.SetLeft(item.GetKey()))
    return &oid
}
```

**Fix:** There's already a TODO to replace this with ExternalObjectId. Do it
before the migration. Alternatively, add a `Set()` call with the full ID string
instead of `SetLeft`.

### 5. SetObjectIdOrBlob Type Assertion (HIGH)

`ids/main.go:219-231` has a fast path that type-asserts `*objectId2` and copies
fields directly:
```go
if other, ok := other.(*objectId2); ok {
    id.genre = other.genre
    other.left.CopyTo(&id.left)
    // ...
}
```

**Fix:** Replace with objectId3 equivalent using `ResetWith` or `SetWithSeq`.
The generic string-based fallback (lines 234+) works for any Id type, so the
fast path just needs updating.

### 6. StringSansRepo (MEDIUM)

objectId2 has a `StringSansRepo()` that returns the ID without the repo prefix.
objectId3 has no repo concept in its Seq, so `StringSansRepo()` should just
return `String()`.

**Callers:**
- `echo/ids/id_stringer.go:14` - `StringerSansRepo` wrapper
- `kilo/box_format/transacted.go:169` - formats ObjectId for box output

**Fix:** Add `StringSansRepo()` to objectId3 that returns `id.Seq.String()`.

### 7. TodoSetFromObjectId Bridge Methods (MEDIUM)

- `echo/ids/tag.go:144-146` - `TodoSetFromObjectId`
- `echo/ids/type.go:114-116` - `TodoSetFromObjectId`

**Callers:**
- `november/queries/build_state.go:539`
- `romeo/store_config/main.go:197`

**Fix:** Replace with `tag.Set(objectId.String())` directly. These are already
marked TODO for removal.

---

## Methods objectId3 Needs Before Flipping

These are methods that objectId2 has that external code calls on ObjectId:

| Method | Purpose | Status |
|--------|---------|--------|
| `SetBlob(string) error` | Set genre to Blob | DONE |
| `StringSansRepo() string` | String without repo | DONE |
| `MarshalText() / UnmarshalText()` | Text serialization | DONE |
| `Clone() *objectId3` | Pool-managed clone | DONE |
| `WriteTo(io.Writer) (int64, error)` | Binary stream write | DONE (genre + seq_len + seq_data) |
| `ReadFrom(io.Reader) (int64, error)` | Binary stream read | DONE (with Seq reset) |
| `SetWithGenre(string, GenreGetter) error` | Set with genre hint | DONE (genre dispatch) |

---

## Bugs Found and Fixed During Previous Attempt

All 5 bugs have been resolved. Bugs 1 and 3 were fixed by implementing the
missing methods with the correct behavior. Bugs 2, 4, and 5 were already
addressed in the existing code.

### Bug 1: Stream Index Corruption (ReadFrom Seq reset) — FIXED
`ReadFrom` resets `id.Seq` at the start before reading new data.
Committed in `7d03bd7e0`.

### Bug 2: Empty UnmarshalBinary — ALREADY FIXED
`UnmarshalBinary` already handles `ErrEmptySeq` (object_id3.go lines 255-258).

### Bug 3: SetWithGenre Genre Dispatch — FIXED
`SetWithGenre` dispatches to `SetType` for types, `SetBlob` for blobs, and
falls back to generic `Set` for other genres. Committed in `7d03bd7e0`.

### Bug 4: ValidateSeqAndGetGenre sec.asec Pattern — ALREADY FIXED
`ValidateSeqAndGetGenre` already matches `ident.ident` with numeric parts as
`genres.InventoryList` (main.go lines 369-385).

### Bug 5: IsEmpty Semantics for "/" Placeholder — NO FIX NEEDED
Current `IsEmpty()` behavior (`id.Seq.Len() == 0`) is correct. The "/"
placeholder has a non-empty Seq `[op('/')]` with objectId3, which is the
desired behavior.

---

## BATS Test Failure Categories (37 failures at last count)

### Category 1: `unsupported seq "add.md"` (6 tests)
File extensions treated as seq when filenames have spaces. The doddish scanner
parses `add.md` as an unsupported seq pattern instead of treating it as a
filename/path.

### Category 2: `unsupported seq "2149773475."` or `"one/"` (5 tests)
Truncated numeric/path seqs rejected during config loading. When inventory list
ObjectIds (TAI timestamps) are parsed, the trailing `.` causes issues.

### Category 3: `seq isn't a zettel id` (7 tests)
Abbreviation index (`zettel_id_index`) rejects non-zettel seqs.

### Category 4: Numeric ID Instead of Alias in Output (13 tests)
Output shows `[2149773475. !toml-type-v1]` instead of `[!md !toml-type-v1]`.
The type definition's ObjectId shows as a TAI timestamp instead of the actual
type ID like `!md`. This was the most pervasive and hardest to debug.

**Root cause investigation:** The organize command's data path goes through
`QueryTransactedAsSkuType` -> `executeInternalQuerySkuType` ->
`FuncPrimitiveQuery` -> `streamIndex.ReadPrimitiveQuery`. But debug logging
showed the stream index binary decoder was NEVER called. Objects must be
coming from somewhere else - likely from inventory list text parsing during
store initialization.

**Possible root causes:**
1. The store initialization reads inventory lists to populate the stream index.
   During this process, `inventory_list_store/main.go:185` sets
   `object.ObjectId.SetWithSeq(tai.ToSeq())` for the inventory LIST object
   itself. If pool management is incorrect, this TAI could leak into content
   objects.
2. The box format text parser within inventory list coders may be calling
   `Set()` on objectId3 with a string that gets misidentified by
   `ValidateSeqAndGetGenre`.
3. Config loading via gob may produce corrupted ObjectIds that propagate.

### Category 5: `no coders available for type: "!md"` (1 test)
Type resolution failure - the coder system can't find a handler for `!md`.

### Category 6: Various Edge Cases (5 tests)
Empty type panic, type mismatch, missing output.

---

## Key Architectural Insights

### Data Flow for Queries

```
organize command
  -> store.QueryTransactedAsSkuType(query)
    -> executor.ExecuteTransactedAsSkuType()
      -> if isDotOperatorActive() && WorkspaceStore != nil:
           executeExternalQueryCheckedOut()    // workspace files
         else:
           executeInternalQuerySkuType()       // stream index
             -> FuncPrimitiveQuery()
               -> streamIndex.ReadPrimitiveQuery()
                 -> for each page: makeStreamPageReader()
                   -> makeSeqObjectFromReader()
                     -> decoder.readFormatAndMatchSigil()  // binary decode
```

### Data Sources
- **Source of truth:** Inventory lists (text format, in `.dodder/local/share/inventory_lists_log`)
- **Index (rebuilt):** Stream index binary pages (in `.dodder/local/share/objects_index/Page-N`)
- **Cache (rebuilt):** Config mutable (gob, in `.dodder/local/share/config-mutable`)

### Store Initialization Order
1. Initialize inventory list store
2. Make working list
3. Create zettel ID index
4. Create stream index via `stream_index.MakeIndex` (lazy - doesn't read pages)
5. Stream index pages are read lazily on first query

### When Reindex Happens
- Explicit `dodder reindex` command
- Store version change triggers `SetNeedsFlushHistory` during `Unlock()`
- `Unlock()` calls `FlushInventoryList()` which only adds the inventory list
  object itself (not contents) to the stream index

---

## Recommended Migration Order

### Phase 0: Preparatory PRs (on master, before the branch)
1. ~~**Refactor store_browser:** Replace `item.GetObjectId()` / `SetLeft()`~~
   DONE — replaced with `i.String()`, deleted `GetObjectId()` method
2. ~~**Remove TodoSetFromObjectId:** Replace callers with
   `tag.Set(oid.String())`~~ DONE — deleted methods from tag.go and type.go
3. **Add version-aware config-mutable:** Either add a version header or switch
   to a rebuildable format (DEFERRED — needs separate plan)
4. ~~**Add StringSansRepo to objectId3:** Trivial method returning `String()`~~
   DONE

### Phase 1: Add Missing Methods to objectId3 — COMPLETE
- ~~SetBlob~~ DONE
- ~~WriteTo/ReadFrom (genre byte + seq_len + seq binary)~~ DONE
- ~~SetWithGenre with proper genre dispatch~~ DONE
- ~~MarshalText/UnmarshalText~~ DONE
- ~~Clone~~ DONE

### Phase 2: Bump Store Version
- Change VCurrent from V12 to V13
- Regenerate BATS fixtures with `just test-bats-generate`
- Verify migration tests cover the version bump path

### Phase 3: Flip the Alias
- Change `type ObjectId = objectId2` to `type ObjectId = objectId3`
- Change `GetObjectIdPool()` to return `getObjectIdPool3()`
- Update `SetObjectIdOrBlob` type assertion
- Fix compilation errors

### Phase 4: Test and Debug
- Run `go test ./...` first (unit tests)
- Run `just test-bats` sequentially (`--jobs 1`) to avoid test interference
- Debug failures category by category

### Phase 5: Clean Up
- Delete objectId2.go
- Delete poolObjectId2 / getObjectIdPool2()
- Consider removing SeqId alias (redundant with ObjectId)
- Remove temporary debug logging

---

## Testing Tips

- Always run BATS tests with `--jobs 1` to eliminate parallel test interference
- BATS tests need `DODDER_BIN` set and the debug binary on PATH
- Use `nix develop .#go --command bash -c "just build"` to build
- Regenerate fixtures after any store version or binary format change
- The `info store-version` command determines which fixture directory is used
- Debug logging with `fmt.Fprintf(os.Stderr, ...)` works; `ui.Log().Print()`
  requires debug level flags

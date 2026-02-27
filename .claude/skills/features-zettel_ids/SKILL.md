---
name: features-zettel_ids
description: This skill should be used when working with dodder ZettelIds (two-part identifiers like ceroplastes/midtown), the Zettel ID index, Yin/Yang provider word lists, coordinate-to-ID mapping, ID allocation and exhaustion, or predictable vs random ID selection. Also applies when investigating how "dodder init" populates the ID pool, writing code that calls CreateZettelId / AddZettelId / PeekZettelIds, debugging "zettel ids exhausted" errors, or writing tests that create new zettels or query available IDs.
---

# ZettelId System

Dodder assigns each zettel a unique two-part string identifier (e.g., `ceroplastes/midtown`). The system pre-computes a finite pool of available IDs at repo creation time and tracks which have been consumed.

## ZettelId Format

A `ZettelId` is a struct with `left` and `right` string fields, rendered as `left/right`. Both parts must be non-empty identifiers. Parsed via the doddish tokenizer (identifier `/` identifier).

**Source:** `go/internal/echo/ids/zettel_id.go` -- `ZettelId` struct, `Set()`, `String()`

## Provider Files (Yin/Yang)

Each repo stores two newline-delimited word lists created at `dodder init` time:

- **Yin** -- left-side identifiers, at `DirObjectId()/Yin`
- **Yang** -- right-side identifiers, at `DirObjectId()/Yang`

`DirObjectId()` resolves to `layout.MakeDirData("object_ids")` within the `.dodder` repo data directory (not `.madder`). The files are copied from `-yin` and `-yang` flag paths during genesis.

Total ID space = `len(Yin) * len(Yang)` combinations. The provider lists are immutable after creation.

**Source:** `go/internal/foxtrot/object_id_provider/factory.go` -- `Provider` struct with `yin`/`yang` fields, `New()` reads from `DirObjectId()`

**Provider type:** `go/internal/foxtrot/object_id_provider/main.go` -- `provider` is a `[]string`. Forward lookup via `MakeZettelIdFromCoordinates(i)` returns `provider[i]`. Reverse lookup via `ZettelId(v)` is a linear scan.

## Coordinate System

A triangular-number mapping converts each 2D `(Left, Right)` pair to a single 1D integer and back. This allows the index to store a flat set of integers.

- **2D to 1D:** `n = Left + Right + 1; ext = Extrema(n); return ext.Left + Left`
- **1D to 2D:** `n = round(sqrt(id * 2)); ext = Extrema(n); Left = id - ext.Left; Right = ext.Right - id`
- **Extrema(n):** `Left = (n-1)*n/2 + 1`, `Right = n*(n+1)/2`

**Source:** `go/internal/_/coordinates/kennung.go` -- `ZettelIdCoordinate{Left, Right uint32}`, `Id()`, `SetInt()`

## ZettelId Index

The `Index` interface (`go/internal/india/zettel_id_index/main.go:16-22`) tracks which IDs are available:

```go
type Index interface {
    errors.Flusher
    CreateZettelId() (*ids.ZettelId, error)
    interfaces.ResetableWithError
    AddZettelId(ids.Id) error
    PeekZettelIds(int) ([]*ids.ZettelId, error)
}
```

### v0 Implementation (active)

Uses `map[int]bool` where each key is a 1D coordinate integer. Presence in the map = available.

- **`Reset()`** -- Rebuilds the full pool: iterates all `(l, r)` pairs from `0..len(Yin)-1` x `0..len(Yang)-1`, converts each to a 1D coordinate, inserts into `AvailableIds`.
- **`AddZettelId(id)`** -- Marks an ID as consumed: reverse-looks up the left/right strings in the providers to get coordinates, converts to 1D, deletes from `AvailableIds`.
- **`CreateZettelId()`** -- Allocates a new ID: picks from the available pool (random or predictable), deletes it, maps the coordinate through Yin/Yang providers to produce the string ID.
- **`PeekZettelIds(n)`** -- Preview up to `n` available IDs without consuming them.

**Persistence:** GOB-encoded `encodedIds{AvailableIds}` struct at `FileCacheObjectId()` (resolves to `DirDataIndex("object_id")`). Lazy-loaded on first access, flushed on `Store.Flush()`.

**Thread safety:** `sync.Mutex` protects all map operations.

**Source:** `go/internal/india/zettel_id_index/v0/main.go`

### v1 Implementation (disabled)

Uses `collections.Bitset` instead of a map. Currently gated behind `if false` in the factory (`go/internal/india/zettel_id_index/main.go:30`).

**Source:** `go/internal/india/zettel_id_index/v1/main.go`

## Allocation Modes

Controlled by `configCli.UsePredictableZettelIds()`:

- **Random (default):** `rand.Intn(len(AvailableIds) - 1)`, then iterates to the Nth entry. Provides entropy for user-facing IDs.
- **Predictable:** Always picks the smallest available coordinate. Used in testing for deterministic output.

## Exhaustion

When `len(AvailableIds) == 0`, `CreateZettelId()` returns `ErrZettelIdsExhausted`.

**Source:** `go/internal/foxtrot/object_id_provider/errors.go:33-60`

## Store Integration

The `Store` (`go/internal/tango/store/`) holds a `zettelIdIndex` field. When a zettel is written, `AddZettelId()` is called to mark it consumed. `CreateZettelId()` is called when minting new zettels. The index is flushed as part of `Store.Flush()`.

## Key Source Locations

| Concern | Package |
|---------|---------|
| ZettelId type & parsing | `go/internal/echo/ids/zettel_id.go` |
| Coordinate mapping | `go/internal/_/coordinates/kennung.go` |
| Yin/Yang provider files | `go/internal/foxtrot/object_id_provider/` |
| Index interface | `go/internal/india/zettel_id_index/main.go` |
| v0 index (map-based, active) | `go/internal/india/zettel_id_index/v0/main.go` |
| v1 index (bitset, disabled) | `go/internal/india/zettel_id_index/v1/main.go` |
| Genesis (Yin/Yang copy) | `go/internal/juliett/env_repo/genesis.go` -- `CopyFileLines` calls |
| Directory layout paths | `go/internal/echo/directory_layout/v3.go` -- `DirObjectId()`, `FileCacheObjectId()` |
| Exhaustion error | `go/internal/foxtrot/object_id_provider/errors.go` |
| Store integration | `go/internal/tango/store/` |

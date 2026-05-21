---
name: features-zettel_ids
description: Use when working on the ZettelId internals --- the Yin/Yang provider, the triangular-number coordinate mapping, the Index interface and its v0/v1 implementations, raw-binary persistence, calls to `CreateZettelId` / `AddZettelId` / `PeekZettelIds`, ZettelId genesis from `dodder init`, or `ErrZettelIdsExhausted` debugging.
---

# ZettelId Internals

User-facing behavior (CLI format, exhaustion at the command level) lives in
the plugin copy of this skill. This file covers the implementation: data
structures, source locations, and Index semantics.

## ZettelId Type

A `ZettelId` is a struct with `left` and `right` string fields, rendered as
`left/right`. Both parts must be non-empty identifiers. Parsed via the
doddish tokenizer (identifier `/` identifier).

**Source:** `go/internal/bravo/ids/zettel_id.go` --- `ZettelId` struct,
`Set()`, `String()`.

## Provider Files (Yin/Yang)

Each repo stores two newline-delimited word lists created at `dodder init`
time:

- **Yin** --- left-side identifiers, at `DirObjectId()/Yin`
- **Yang** --- right-side identifiers, at `DirObjectId()/Yang`

`DirObjectId()` resolves to `layout.MakeDirData("object_ids")` within the
`.dodder` repo data directory (not `.madder`). The files are copied from
`-yin` and `-yang` flag paths during genesis.

Total ID space = `len(Yin) * len(Yang)` combinations. The provider lists
are immutable after creation.

**Source:** `go/internal/echo/zettel_id_provider/` --- `Provider` struct
with `yin`/`yang` fields. Reverse lookup of a string -> coordinate is a
linear scan.

## Coordinate System

A triangular-number mapping converts each 2D `(Left, Right)` pair to a
single 1D integer and back. This allows the index to store a flat set of
integers.

- **2D to 1D:** `n = Left + Right + 1; ext = Extrema(n); return ext.Left + Left`
- **1D to 2D:** `n = round(sqrt(id * 2)); ext = Extrema(n); Left = id - ext.Left; Right = ext.Right - id`
- **Extrema(n):** `Left = (n-1)*n/2 + 1`, `Right = n*(n+1)/2`

**Source:** `go/internal/0/coordinates/kennung.go` ---
`ZettelIdCoordinate{Left, Right uint32}`, `Id()`, `SetInt()`.

## ZettelId Index

The `Index` interface (`go/internal/foxtrot/zettel_id_index/main.go`)
tracks which IDs are available:

```go
type Index interface {
    errors.Flusher
    CreateZettelId() (*ids.ZettelId, error)
    interfaces.ResetableWithError
    AddZettelId(ids.Id) error
    PeekZettelIds(int) ([]*ids.ZettelId, error)
}
```

The factory `MakeIndex` is gated behind a hardcoded `if true { v1 } else
{ v0 }` switch --- v1 is the currently active implementation.

### v1 Implementation (active)

Uses `collections.Bitset` indexed by 1D coordinate integer. Persists via
`encoding.BinaryMarshaler` on the bitset --- not gob, not the v0 uint32
stream.

**Source:** `go/internal/foxtrot/zettel_id_index/v1/main.go`.

### v0 Implementation (dormant)

Uses `map[int]bool` where each key is a 1D coordinate integer. Presence in
the map = available.

- **`Reset()`** --- Rebuilds the full pool: iterates all `(l, r)` pairs
  from `0..len(Yin)-1` x `0..len(Yang)-1`, converts each to a 1D
  coordinate, inserts into `AvailableIds`.
- **`AddZettelId(id)`** --- Marks an ID as consumed: reverse-looks up the
  left/right strings in the providers to get coordinates, converts to 1D,
  deletes from `AvailableIds`.
- **`CreateZettelId()`** --- Allocates a new ID: picks from the available
  pool (random or predictable), deletes it, maps the coordinate through
  Yin/Yang providers to produce the string ID.
- **`PeekZettelIds(n)`** --- Preview up to `n` available IDs without
  consuming them.

**Persistence:** raw big-endian uint32 stream at `FileCacheObjectId()`
(resolves to `DirDataIndex("object_id")`). Layout: `uint32 count`,
followed by `count` × `uint32` coordinate keys, via `binary.Write`.
Earlier revisions used gob encoding; the migration off gob and a runtime
sanity check on `count` were added to prevent multi-GB allocations when
the on-disk format drifted (see the comment referencing
amarbel-llc/dodder#68 in `v0/main.go`).

**Thread safety:** `sync.Mutex` protects all map operations.

**Source:** `go/internal/foxtrot/zettel_id_index/v0/main.go`.

## Allocation Modes

Controlled by `configCli.UsePredictableZettelIds()`:

- **Random (default):** `rand.Intn(len(AvailableIds) - 1)`, then iterates
  to the Nth entry.
- **Predictable:** Always picks the smallest available coordinate. Used in
  testing for deterministic output.

## Exhaustion

When the available set is empty, `CreateZettelId()` returns
`ErrZettelIdsExhausted`.

**Source:** `go/internal/echo/zettel_id_provider/errors.go`.

## Store Integration

The `Store` (`go/internal/oscar/store/`) holds a `zettelIdIndex` field.
When a zettel is written, `AddZettelId()` is called to mark it consumed.
`CreateZettelId()` is called when minting new zettels
(`go/internal/sierra/repo_actions/write_new_zettels.go`,
`checkin_haustoria.go`, `op_checkin.go` in `romeo/local_working_copy/`).
The index is flushed as part of `Store.Flush()`.

## Key Source Locations

| Concern | Package |
|---------|---------|
| ZettelId type & parsing | `go/internal/bravo/ids/zettel_id.go` |
| Coordinate mapping | `go/internal/0/coordinates/kennung.go` |
| Yin/Yang provider files | `go/internal/echo/zettel_id_provider/` |
| Index interface & factory | `go/internal/foxtrot/zettel_id_index/main.go` |
| v1 index (bitset, active) | `go/internal/foxtrot/zettel_id_index/v1/main.go` |
| v0 index (map-based, dormant) | `go/internal/foxtrot/zettel_id_index/v0/main.go` |
| Genesis (Yin/Yang copy) | `go/internal/foxtrot/env_repo/genesis.go` |
| Directory layout paths | `go/internal/bravo/directory_layout/v3.go` --- `DirObjectId()`, `FileCacheObjectId()` |
| Exhaustion error | `go/internal/echo/zettel_id_provider/errors.go` |
| Store integration | `go/internal/oscar/store/` |

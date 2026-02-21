---
name: design_pattern-genre-bitfield
description: >
  Use when working with object genres, adding genre filters to queries,
  parsing sigils like :z :t :e :r :b, or modifying genre-based dispatch.
  Also applies when encountering Genre byte type, genre bitwise operations,
  MakeGenre, or optimizedQueries maps.
triggers:
  - genre
  - Genre byte
  - sigil
  - :z :t :e :r :b
  - MakeGenre
  - genre bitfield
  - object genre
  - genre filter
---

# Genre Bitfield System

## Overview

Dodder categorizes objects into genres (Zettel, Tag, Type, Repo, InventoryList,
Blob, Config) using a byte-sized enum. For set operations — "show all zettels
and tags" — a separate `ids.Genre` bitfield type packs multiple genres into a
single byte using power-of-two bit flags. Queries store genre-optimized
sub-queries keyed by genre for efficient dispatch.

## Two Genre Types

### `genres.Genre` (Enum)

Sequential enum values for identifying a single genre:

```go
// charlie/genres/main.go
type Genre byte

const (
    Unknown = Genre(iota)
    Blob        // 1
    Type        // 2
    Tag         // 3
    Zettel      // 4
    Config      // 5
    InventoryList // 6
    Repo        // 7
)
```

### `ids.Genre` (Bitfield)

Bitfield for efficient set membership testing:

```go
// charlie/genres/main.go (internal bit values)
const (
    blob           = byte(1 << iota)  // 2
    tipe                               // 4
    tag                                // 8
    zettel                             // 16
    config                             // 32
    inventory_list                     // 64
    repo                               // 128
)
```

Conversion between enum and bitfield uses `GetGenreBitInt()`.

## Bitfield Operations

```go
// echo/ids/genre.go

// Construction
genre := ids.MakeGenre(genres.Zettel, genres.Tag)

// Add genres
genre.Add(genres.Type)

// Remove genres
genre.Del(genres.Repo)

// Test membership
genre.Contains(genres.Zettel)      // exact match
genre.ContainsOneOf(genres.Zettel) // any bit overlap

// Convert to slice
slice := genre.Slice() // []genres.Genre{Zettel, Tag, Type}
```

## Sigil-to-Genre Mapping

Users specify genres in queries via sigil characters:

| Sigil | Genre | Aliases |
|-------|-------|---------|
| `:z` | Zettel | `zettel` |
| `:t` | Tag | `tag`, `etikett` |
| `:e` | Type | `type`, `typ` |
| `:r` | Repo | `repo`, `kasten` |
| `:b` | Blob | `blob`, `akte` |

Combined: `:z,t` = Zettel + Tag bitfield.

## Genre in ObjectId Parsing

`echo/ids/main.go` determines genre from syntax:

| Syntax | Genre | Example |
|--------|-------|---------|
| `prefix/suffix` | Zettel | `one/uno` |
| `plain-name` or `-name` | Tag | `project`, `-blocked` |
| `!name` | Type | `!md` |
| `/name` | Repo | `/my-repo` |
| `@digest` or `purpose@digest` | Blob | `@abc123` |
| `sec.asec` | InventoryList | `1234.5678` |

## Query Optimization

Queries maintain per-genre sub-queries for fast dispatch:

```go
// november/queries/main.go
for _, g := range genres {
    existing, ok := query.optimizedQueries[g]
    if !ok {
        existing = buildState.makeQuery()
        existing.Genre = ids.MakeGenre(g)
    }
    query.optimizedQueries[g] = existing
}
```

When matching, the query checks only the genre-specific sub-query:

```go
g := genres.Must(sk.GetGenre())
q, ok := qg.optimizedQueries[g]
if !ok || !q.ContainsExternalSku(el) {
    return false
}
```

## Common Mistakes

| Mistake | Correct Approach |
|---------|-----------------|
| Using enum values for bitwise ops | Use `GetGenreBitInt()` to convert enum to bit flag |
| Comparing `ids.Genre` with `==` for set membership | Use `Contains()` or `ContainsOneOf()` |
| Hardcoding genre detection instead of parsing | Use `ValidateSeqAndGetGenre` from `echo/ids/` |

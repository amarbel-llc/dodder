---
status: exploring
date: 2026-03-07
promotion-criteria: all three lock kinds (type, tag, referenced object) round-trip through text, inventory list, binary, and JSON formats; migration tests pass for stores created before referenced object locks existed
---

# Object Locks

## Problem Statement

When dodder commits an object, it pins the current signatures of the object's
type and tags into the object's metadata. This prevents silent schema drift: if a
type or tag definition changes after the object was committed, the lock records
exactly which version was in effect at commit time.

Today locks cover types and tags but not referenced objects. An object can
reference other objects (e.g. a zettel linking to another zettel), and those
references carry no version pin. If a referenced object changes, there is no
record of which version was in scope when the referencing object was committed.
Adding referenced-object locks closes this gap.

## Interface

### Lock kinds

Each object's metadata carries:

- **Type lock** (`! type@signature`) -- pins the object's type to the signature
  of the type-object at commit time. `!` prefix inspired by shebangs.
- **Tag locks** (`tag@signature`) -- one per tag, pins each tag to its
  tag-object signature at commit time.
- **Referenced object locks** (`< object-ref@signature`) -- one per referenced
  object, pins each reference to the referenced object's signature at commit
  time. Optional alias maps a blob-local name to the fully qualified object ID.
  `<` prefix inspired by shell input redirection.

Referenced objects are discovered by the object's type (which defines how to
extract references from blob content). The metadata stores the result: a map of
fully qualified object IDs (with optional aliases) to their pinned signatures.

### Lock data model

`markl.Lock[KEY, KEY_PTR]` pairs a key (type ID, tag ID, or object ID) with a
`markl.Id` value (the pinned signature):

```
Lock { Key KEY, Value markl.Id }
```

Referenced object locks reuse the existing `containedObject` struct:

- `ContainedObjectType`: `containedObjectTypeReferencedObject` (new value)
- `Lock.Key`: fully qualified object ID (e.g. `one/uno`)
- `Lock.Value`: object signature at commit time
- `Alias`: optional blob-local alias (e.g. `blog-template`)

Stored in a new `References ContainedObjects` field on metadata, separate from
`Tags`.

Lock values are required in all persistent formats (binary, inventory list) and
optional only in user-facing text input.

### Serialization

| Format         | Type lock             | Tag lock           | Referenced object lock                     |
|----------------|-----------------------|--------------------|--------------------------------------------|
| Triple-hyphen  | `! type@signature`    | (in tag lines)     | `< ref@sig` or `< alias = ref@sig`         |
| Inventory list | `!type@signature`     | `tag@signature`    | `<ref@sig` or `<alias=ref@sig`             |
| Binary index   | key + null + fmt + id | same               | same, with ContainedObjectType byte        |
| JSON           | `{ "Lock": { "Type": "sig" } }` | --      | `{ "References": { ... } }`               |

Triple-hyphen examples:

    < one/uno@blake2b256-abc...
    < blog-template = one/uno@blake2b256-abc...
    < "unsafe alias with spaces" = one/uno@blake2b256-abc...

Inventory list box examples:

    [one/dos @digest !md <one/uno@sig <blog-template=one/uno@sig]

Aliases with unsafe characters are quoted using the same escaping rules as
object descriptions.

### Metadata interface

New methods on `Metadata` / `MetadataMutable`:

- `GetReferencedObjects() ContainedObjects`
- `GetReferencedObjectLock(SeqId) IdLock`
- `GetReferencedObjectLockMutable(SeqId) IdLockMutable`
- `AllReferencedObjects()` iterator (like `AllTags()`)

### Commit options

`LockfileOptions` controls failure tolerance:

- `AllowTypeFailure` -- if the type object can't be read, skip its lock
- `AllowTagFailure` -- same for tags
- `AllowReferencedObjectFailure` -- same for referenced objects

### Doddish tokenizer

`<` is registered as a new mixed-sequence operator in `doddish/op.go` (like `!`,
`@`, `%`).

New token matchers:

- `TokenMatcherReferencedObject`: `< identifier @ identifier`
- `TokenMatcherReferencedObjectAlias`: `< identifier = identifier @ identifier`

### Sigil design etymology

- `!` (type) -- inspired by shebangs (`#!/bin/sh`)
- `#` (description) -- inspired by shebangs / comment syntax
- `<` (referenced object) -- inspired by shell input redirection (`< file`)

## Examples

A zettel with type `doc`, tag `project`, and a reference to `one/uno` aliased as
`blog-template`:

Triple-hyphen output:

    # my blog post
    - project
    ! doc@blake2b256-abc...
    < blog-template = one/uno@blake2b256-def...
    ---
    See [blog-template] for the layout.

Inventory list output:

    [ceroplastes/midtown @digest !doc@blake2b256-abc... project@blake2b256-ghi... <blog-template=one/uno@blake2b256-def...]

## Limitations

- Builtin types are not locked (there is a TODO to address this).
- Lock values are not overwritten once set during a commit -- if a lock value
  already exists, the finalizer skips it. This means re-committing an object
  does not update its locks to the latest type/tag signatures unless the lock
  is explicitly cleared first.
- **Reference discovery is out of scope.** Types define how to discover object
  references in blob content, but the discovery mechanism itself is not part of
  this FDR. Types are dynamic and user-defined, so the implementation must
  afford flexibility. Possible approaches include WASM guest modules,
  shelling out to external programs, regexes, Lua scripts, or built-in parsers
  for structured formats (JSON, TOML, etc.). A separate FDR should cover the
  discovery interface.

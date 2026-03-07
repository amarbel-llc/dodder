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

### Current lock kinds

Each object's metadata carries:

- **Type lock** (`! type@signature`) -- pins the object's type to the signature
  of the type-object at commit time.
- **Tag locks** (`tag@signature`) -- one per tag, pins each tag to its
  tag-object signature at commit time.

Locks are written during `WriteLockfile()` in the object finalizer, called as
part of `validateAndFinalize()` before digest calculation.

### Lock data model

`markl.Lock[KEY, KEY_PTR]` pairs a key (type ID, tag ID, or object ID) with a
`markl.Id` value (the pinned signature):

```
Lock { Key KEY, Value markl.Id }
```

Lock values are required in all persistent formats (binary, inventory list) and
optional only in user-facing text input.

### Serialization

| Format         | Type lock             | Tag lock           |
|----------------|-----------------------|--------------------|
| Triple-hyphen  | `! type@signature`    | (in tag lines)     |
| Inventory list | `!type@signature`     | `tag@signature`    |
| Binary index   | key + null + fmt + id | same               |
| JSON           | `{ "Lock": { "Type": "signature" } }` | --  |

### Commit options

`LockfileOptions` controls failure tolerance:

- `AllowTypeFailure` -- if the type object can't be read, skip its lock
- `AllowTagFailure` -- same for tags

## Examples

A zettel with type `doc` and tag `project` committed when `doc` has signature
`blake2b256-abc...` and `project` has signature `blake2b256-def...`:

Triple-hyphen output:

    ! doc@blake2b256-abc...
    project
    ---
    blob content

Inventory list output:

    [ceroplastes/midtown !doc@blake2b256-abc... project@blake2b256-def...]

## Limitations

- Builtin types are not locked (there is a TODO to address this).
- Lock values are not overwritten once set during a commit -- if a lock value
  already exists, the finalizer skips it. This means re-committing an object
  does not update its locks to the latest type/tag signatures unless the lock
  is explicitly cleared first.

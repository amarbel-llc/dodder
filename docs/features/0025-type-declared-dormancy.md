---
status: proposed
date: 2026-09-29
promotion-criteria: a type blob can declare terminal values on an enum
  field; an object whose projected field holds a terminal value is dormant
  on every commit path that projects fields (create, update, import,
  init-from-lists) with no hook and no tag involved; the built-in
  actionable types declare their terminal states and the actionable hook no
  longer archives; bats covers !task done/cancelled, !chore done (not
  dormant) and cancelled (dormant), and an init-from-lists import of a done
  !task.
---

# Type-Declared Dormancy

## Problem Statement

The built-in actionable types (`!task`, `!chore`, `!habit`) make an object
dormant through a side effect. The `on_commit_fields` hook in
`embedded/actionable/actionable-common.lua` adds a `zz-archive` tag when
the object reaches a terminal status, and genesis seeds `zz-archive` into
the dormant index, but only under `-include-builtin-actionable-types`
(`type_blobs.ArchiveTag`, `romeo/local_working_copy/genesis.go`).

So dormancy for these objects depends on:

- the hook running (any path that commits without it leaves terminal
  objects active),
- a tag that was a personal convention, baked into built-in behavior, and
- the dormant index happening to contain that tag.

The terminal-ness of a state is a property of the type, not of a tag, so
the type should declare it (dodder#399).

## Interface

### Declaring terminal values

`FieldDefinition` (type blob `[[fields]]` entries) gains an optional
`terminal` list. It is valid only on `kind = "enum"` fields, and every
entry must also appear in `values`:

    [[fields]]
    name = "status"
    kind = "enum"
    values = ["todo", "in_progress", "done", "cancelled"]
    default = "todo"
    terminal = ["done", "cancelled"]

Semantics: an object is **type-dormant** when any of its projected fields
has a value listed in that field's `terminal` list.

The key is `omitempty`, following the `required` precedent: type blobs
that don't declare it stay byte-identical, so no type blob version bump.

### Evaluating dormancy

Dormancy stays a single persisted bit on the object's index
(`index.Dormant`), and the `?` sigil and dormant filtering keep reading
only that bit. What changes is how the bit is computed in
`applyDormantAndRealizeTags` (the stream index's pre-write step):

    dormant = dormantIndex.ContainsSku(object) || typeDormant(object)

`typeDormant` walks the object's projected index fields. Each projected
field already records the digest of the type blob it was projected
against (`fields.Field.TypeBlobDigest`), so the terminal sets are looked
up by `(TypeBlobDigest, field name)` from a per-store cache of parsed type
blobs. Objects with no projected fields pay nothing.

Tag-based dormancy (the dormant index, `dormant-add`, `dormant-remove`)
is unchanged and ORed in. It remains the mechanism for objects without a
state field (e.g. `!md` notes tagged `zz-archive*`).

### Built-in actionable types

| type      | `status` terminal values |
|-----------|--------------------------|
| `!task`   | `done`, `cancelled`      |
| `!chore`  | `cancelled`              |
| `!habit`  | `cancelled`              |

For `!chore` and `!habit`, `done` is not terminal. The recurrence hook
still advances `due` and resets `status` to `todo`; because dormancy is
computed after hooks run, the committed object is `todo` and active.
Retirement is `cancelled` until extensible enums (dodder#398) allow a
distinct state.

### Hook changes

`actionable-common.lua` stops adding `zz-archive`. It keeps:

- the recurrence behavior (advance `due`, reset to `todo`), and
- stamping today into an empty `due` when a `!task` reaches `done` or any
  actionable object reaches `cancelled` (the completed-on date). This is
  not dormancy, so it stays in the hook.

### Genesis

Genesis stops seeding `zz-archive` into the dormant index, and
`type_blobs.ArchiveTag` is removed. Existing repos keep whatever their
dormant index already holds, so their `zz-archive`-tagged objects stay
dormant.

## Examples

    $ dodder new -type task -edit=false ...   # status = "todo"
    $ dodder show :z                          # listed
    # edit status to "done", checkin
    $ dodder show :z                          # not listed
    $ dodder show :?z                         # listed; carries no zz-archive tag

An `init-from-lists` consolidation of a `!task` whose blob has
`status = "done"`, carrying no tags, produces a dormant object in the new
repo (covered by `init_from_lists.bats`).

## Limitations

- **Dormancy follows field projection.** Projected fields are not stored
  in inventory lists; they're re-derived by the type's fields-reader.
  Create, update, import, init-from-lists, pull and clone project them
  (`RunHooks`); reindex re-projects them without running hooks
  (`ProjectFields`, dodder#403), logging and skipping a projection that
  fails. Covered by `type_dormancy_transfer.bats`.
- **Dormant objects only transfer when asked for.** Pull/clone queries
  without the `?` sigil skip dormant objects, so a done `!task` travels
  only with e.g. `+?z,t,e`.
- **Same-batch type resolution.** Projection needs the object's type
  object. An import (e.g. init-from-lists) commits types and the objects
  that use them in one unflushed batch, and type resolution read only the
  persisted probe index, so imported objects silently got no fields. Type
  resolution (by id and by type lock) now checks the stream index's
  addition probes first, as `WriteLockfile` already did. This is scoped to
  types: making every object lookup see unflushed objects breaks history
  imports (push/pull), whose conflict detection expects earlier versions
  from the same batch to be invisible.
- **Changing a type's terminal list doesn't re-evaluate existing
  objects** until they're rewritten (reindex or a new commit). This is the
  same as `dormant-add` today.
- **Existing repos keep the old built-in type blobs.** Their `!task`
  objects gain type-dormancy only after the type is updated to declare
  `terminal`, and they keep the `zz-archive` dormant-index entry either
  way.

## More Information

- dodder#399 (this feature), dodder#398 (extensible enums), dodder#16
  (take4 personal-data consolidation, the first consumer)
- FDR 0017 (type-defined field index), FDR 0022 (dang)

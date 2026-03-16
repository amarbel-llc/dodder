---
status: exploring
date: 2026-03-08
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
- **Referenced object locks** (`- ref@signature`) -- one per referenced object,
  pins each reference to the referenced object's signature at commit time.
  Optional alias uses `<` as inline separator: `- alias < ref@signature`. Uses
  the same `-` prefix as tags (unified syntax).

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
| Triple-hyphen  | `! type@signature`    | (in tag lines)     | `- ref@sig` or `- alias < ref@sig`         |
| Inventory list | `!type@signature`     | `tag@signature`    | `<ref@sig` or `<alias=ref@sig`             |
| Binary index   | key + null + fmt + id | same               | same, with ContainedObjectType byte        |
| JSON           | `{ "Lock": { "Type": "sig" } }` | --      | `{ "References": { ... } }`               |

Triple-hyphen examples:

    - one/uno@blake2b256-abc...
    - blog-template < one/uno@blake2b256-abc...
    - "unsafe alias with spaces" < one/uno@blake2b256-abc...

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

References share the `-` prefix with tags. `<` is used as an inline alias
separator within a `-` line. Both are registered as mixed-sequence operators in
`doddish/op.go`.

### Sigil design etymology

- `!` (type) -- inspired by shebangs (`#!/bin/sh`)
- `#` (description) -- inspired by shebangs / comment syntax
- `-` (tag or reference) -- list item
- `<` (alias separator) -- inspired by shell input redirection (`< file`)

## Examples

A zettel with type `doc`, tag `project`, and a reference to `one/uno` aliased as
`blog-template`:

Triple-hyphen output:

    # my blog post
    - project
    ! doc@blake2b256-abc...
    - blog-template < one/uno@blake2b256-def...
    ---
    See [blog-template] for the layout.

Inventory list output:

    [ceroplastes/midtown @digest !doc@blake2b256-abc... project@blake2b256-ghi... <blog-template=one/uno@blake2b256-def...]

## Consumers

### Edge expansion in filtered pull (workspace-as-repo)

When pulling a filtered subset of objects from a remote repo, the transfer set
may omit types, tags, or referenced objects that the pulled objects depend on.
`expandEdges` (`sierra/local_working_copy/expand_edges.go`) walks all three lock
edge kinds on the pulled objects and transitively includes them in the transfer:

1. **Type edges** -- `object.GetType()` → fetch the type object
2. **Tag edges** -- `object.AllTags()` → fetch each tag object
3. **Referenced object edges** -- `object.GetMetadata().AllReferencedObjects()` →
   fetch each referenced object

Traversal runs up to 5 levels deep (referenced objects may themselves have types,
tags, and references). Objects already in the transfer set or missing from the
remote are skipped silently.

This is used by `PullQueryGroupFromRemote` in
`sierra/local_working_copy/local_op_pull.go` and is exercised by
`init-workspace -experimental-repo` (FDR-0005) and the `clone` command when the
query filters to a subset of objects.

Integration tests: `zz-tests_bats/current_version/workspace_repo.bats`
(`workspace_repo_clone_filtered_by_tag`, `workspace_repo_init_experimental_repo`).

## Limitations

- Builtin types are not locked (there is a TODO to address this).
- Lock values are not overwritten once set during a commit -- if a lock value
  already exists, the finalizer skips it. This means re-committing an object
  does not update its locks to the latest type/tag signatures unless the lock
  is explicitly cleared first.
- **Reference discovery** is covered by a separate design:
  `docs/plans/2026-03-07-object-reference-discovery-design.md`. First
  implementation uses external commands (blob piped to stdin, references on
  stdout), with Lua hooks as future work.

## Exploration Findings: Type Locks and the Formatter Pipeline

This section records what we learned while exploring how type locks interact with
the formatter and reference discovery systems on the `light-maple` branch.

### Reference discovery works with pandoc

Type-driven reference discovery (`[object-references]` in `toml-type-v1`) is
implemented and tested with three approaches:

1. **Shell pipeline** -- `grep -oP` + `sed` extracts `[[wiki-link]]` references.
   Simple but fragile (regex can't handle nested brackets or escaped content).
2. **Pandoc Lua writer** -- `pandoc --from markdown+wikilinks_title_after_pipe
   --to discover-refs.lua`. Walks the AST for `Link` elements with `wikilink`
   class. Structure-aware: handles edge cases that regex misses.
3. **Pandoc CodeBlock handler** -- Same Lua writer also extracts `!type`
   references from fenced code block class attributes (e.g., `` ```!md ``).

All three are covered by integration tests in
`zz-tests_bats/current_version/show.bats` under the `referenced_objects` tag.
The Lua writer source is at `zz-pandoc-refs/discover-refs.lua`.

### Unlocked types in format-object/format-blob -stdin

Pandoc filters call `format-object -stdin !md` at render time, when the type has
no lock (locks are added at commit time, not render time). This originally
panicked in `ReadTypeObject` because the lock value was null.

**Fix (commit 1cb296a):** `GetBlobFormatter` now accepts a resolved
`*sku.Transacted` instead of a `TypeLock`. Callers resolve the type object
themselves:

- **Non-stdin callers:** `store.ReadObjectTypeAndLockIfNecessary(object)` handles
  both locked and unlocked types transparently.
- **Stdin callers:** Branch on whether `typeLock.GetValue().IsNull()`:
  - Null → `store.ReadOneObjectId(typeLock.GetKey())` (resolve latest version)
  - Present → `store.ReadTypeObject(&typeLock)` (use pinned version)

Key files: `sierra/local_working_copy/op_get_blob_formatter.go`,
`victor/commands_dodder/format_blob.go`, `victor/commands_dodder/format_object.go`.

### Formatter selection has three inconsistent default paths

`GetBlobFormatter` in `op_get_blob_formatter.go` has three selection paths:

| Scenario | formatId | utiGroup | Effective behavior |
|----------|----------|----------|--------------------|
| Non-stdin, no args | `""` | `""` | Fallback: try `text-edit`, then `text` |
| Stdin, 1 arg | `"text"` | `""` | Direct: look up `text` only |
| UTI group flag | UTI string | group name | Indirection: UTI → formatter name |

The non-stdin path prefers `text-edit` over `text`, but the stdin path hardcodes
`"text"`. This means `format-blob one/uno` and
`format-blob -stdin !type <<< content` can silently pick different formatters for
the same type definition.

### UTI group ergonomics are awkward

Using a UTI group requires both a `-uti-group` flag AND passing the UTI as a
positional argument:

    format-blob -stdin -uti-group text-edit public.utf8-plain-text !my-type

The positional arg is a UTI identifier (`public.utf8-plain-text`), not a
formatter name. This isn't obvious from the CLI surface. The pandoc filters
(`dodder-edit.lua`, `dodder-render.lua`) bypass UTI groups entirely and pass
formatter names directly.

When a UTI group maps to a non-existent formatter, the code silently returns a
nil formatter and prints "no matching format id" to stderr. This hides
configuration bugs.

### The real !md type has a naming mismatch

Inspected via `der show -format json ':t'`:

```toml
# Real !md type (toml-type-v0 format, uses "formatter-uti-groups")
[formatter-uti-groups.text-edit]
"public.utf8-plain-text" = "text-edit"    # maps to formatter "text-edit"

[formatter-uti-groups.text-render]
"public.utf8-plain-text" = "text-render"  # maps to formatter "text-render"

[formatters.text]                          # ← the ACTUAL edit formatter
shell = ["pandoc", "-dzit-edit"]

[formatters.text-render]                   # ← render formatter (exists)
shell = ["pandoc", "-dzit-render", "--to=markdown"]
```

The `text-edit` UTI group maps `public.utf8-plain-text` to formatter
`"text-edit"`, but **no `[formatters.text-edit]` exists**. The edit formatter is
actually named `"text"` (uses `pandoc -dzit-edit`). The `text-render` UTI group
works correctly because `[formatters.text-render]` does exist.

### Version split between toml-type-v0 and toml-type-v1

- **TomlV0:** UTI groups field is `formatter-uti-groups` (TOML key)
- **TomlV1:** UTI groups field is `uti-groups` (TOML key)
- Both expose the same Go interface via `GetFormatterUTIGroups()`
- The real repo's `!md` type still uses v0 naming; new test types use v1

### Current test coverage

All tests in `zz-tests_bats/current_version/show.bats`, tags `format_stdin` and
`referenced_objects`:

| Test | What it covers |
|------|---------------|
| `show_zettel_with_referenced_object_lock` | Manual `- ref` in metadata gets locked at commit |
| `show_zettel_with_discovered_references` | Shell-based `[object-references]` discovers `[[wiki-links]]` |
| `show_zettel_with_pandoc_discovered_references` | Pandoc Lua writer discovers `[[wiki-links]]` via AST |
| `show_zettel_with_pandoc_discovered_code_block_type_references` | Pandoc discovers `!type` refs in code block classes |
| `format_blob_stdin_resolves_type_with_and_without_lock` | Pandoc formatter works with both locked and unlocked types |
| `format_blob_stdin_selects_formatter_via_uti_group` | UTI groups `text-edit`/`text-render` route to different pandoc output formats |
| `format_blob_prefers_text_edit_over_text` | Default fallback prefers `text-edit` formatter over `text` |

All formatter tests use pandoc (not cat/sed) to exercise the real pipeline.
Pandoc is a devshell dependency in `go/default.nix`.

### Future: blob references

Referenced object locks pin object-to-object relationships. A parallel need
exists for object-to-blob relationships: an object's content may embed or refer
to a specific blob by its `markl.Id` digest (e.g., an image, a code snippet, an
attachment). Today there is no metadata-level record of these blob dependencies.

Blob references would reuse the same alias mechanism as object references but
with `markl.Id` as the key instead of `SeqId`:

- **Literal:** `- @blake2b256-abc...` — pins a blob by digest
- **Aliased:** `- hero-image < @blake2b256-abc...` — blob-local name for a digest

This would let types define blob discovery scripts (analogous to
`[object-references]`) that extract embedded blob digests from content. Use cases
include markdown images referencing blob-store assets, config files embedding
other configs by digest, and ensuring blob garbage collection doesn't delete
blobs still referenced by live objects.

A key use case is type-defined actions that generate blobs for an object to
reference. Types could offer methods that produce blobs and wire them into the
object's metadata automatically. Examples:

- A `!bookmark` type with an action that snapshots the URL, storing each snapshot
  as a blob keyed by date: `- 2026-03-08 < @blake2b256-...`
- A `!music-album` type where each track is a blob reference:
  `- 01-overture < @blake2b256-...`
- The `!md` type itself referencing a pre-packaged pandoc binary as a blob,
  making the type fully self-contained rather than depending on `pandoc` being in
  the host environment

The alias mechanism already supports arbitrary key types via
`containedObject.Alias`. Extending it to `markl.Id` keys requires a new
`ContainedObjectType` value and corresponding serialization in all formats.

### Open questions

- Should the stdin path's default formatId be `""` (triggering the `text-edit` →
  `text` fallback) instead of hardcoded `"text"`, for consistency with
  non-stdin?
- Should UTI group resolution that maps to a non-existent formatter return an
  error instead of silently returning nil?
- Is the UTI group abstraction worth its complexity, given that the pandoc
  filters don't use it?
- Should the real `!md` type's `text-edit` UTI group map to `"text"` (the
  formatter that actually exists) instead of `"text-edit"`?

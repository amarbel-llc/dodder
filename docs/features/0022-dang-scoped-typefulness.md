---
status: draft
date: 2026-06-30
---

# Dang: Scoped, Pinned Typefulness for String Fields

## Abstract

A typed-field object can carry, inside one of its string fields, content that is
itself of a *different* type — the motivating case being the actionable types
(FDR-0017 / the merged-actionable-types work), whose TOML blob holds a `body`
string field containing markdown. Today the field's content type is expressed by
an unresolved first-line convention (`#!dang md`, stripped by the formatter; see
issue #296). `dang` makes that convention real: a string field declares its
content type through a **pinned** reference to a type object, named by a short
**identifier**, bound in the object's metadata, and referenced by a
`#! dang <identifier>` shebang on the field's first line. The identifier is
*scoped* (it means only what the object's binding table says) and the reference
is *pinned* (a content/identity lock, optionally signature-versioned), so the
content type survives renames and is verifiable — built on the same
named-pinned-reference substrate dodder already uses for blob and object
references. Longer term, `dang` becomes a real delivered interpreter binary and
the shebang becomes an executable dodder-resolution invocation (deferred; see
Future work).

## Motivation

The merged actionable types store prose in a TOML `body` field. The field's
*declared* type is `string`, but its *content* is markdown (`!md`). The formatter
needs to know "render this string as `!md`," and an editor wants `!md` syntax for
that region. The Phase-1 stand-in is a magic first line:

```
yq -p toml -r '.body' | sed '1{/^#!dang/d}' | pandoc …
```

The `md` in `#!dang md` is never resolved — it is stripped and the formatter just
*assumes* markdown. That has three problems:

1. **Unresolved.** `md` is a bare string, not a reference to the `!md` type. A
   different identifier, a typo, or a rename of `!md` is undetectable.
2. **Unpinned.** Nothing ties the content type to a specific, stable type
   identity. Two repos' `!md` could diverge; a public type repo (dodder.net)
   can't guarantee the consumer resolves the same `!md`.
3. **Single, implicit.** Only one content type is expressible, and only by
   convention baked into the formatter script.

`dang` resolves the identifier through a metadata binding to a *pinned* type
object, making the content type explicit, verifiable, and (optionally)
cryptographically stable.

## Requirements language

The key words MUST, MUST NOT, SHOULD, MAY are to be interpreted as in RFC 2119.

## The mechanism

### Binding table

An object's metadata carries zero or more **dang bindings**, each a pair:

```
<identifier>  →  <pinned type reference>
```

where `<identifier>` is a short local name (e.g. `md`) and the pinned type
reference is a type object identity locked via `markl.Lock` — either unversioned
(`!md`, trust the namespace) or signature-pinned (`!md@ed25519_sig-…`, immutable
content identity). This is the *same shape* as an existing named reference's
`(alias, TypeLock)` pair, minus the blob digest — a dang binding references a
**type**, not a blob (see Storage).

### Shebang

A string field whose content is dang-typed begins with a shebang line, in the
OS-interpreter form (note the space after `#!`):

```
#! dang <identifier>
```

as the field's first line. This deliberately mirrors a Unix interpreter line
(`#! /usr/bin/env python`): `dang` is the interpreter token and `<identifier>` its
argument. In the near-term (this FDR) dodder recognizes the `dang` token and
resolves `<identifier>` through the object's binding table; longer term `dang` is
a real delivered binary the shebang can invoke (Future work), so the two forms
are intended to be interchangeable. The shebang MUST be the field's first line,
literal `#! dang ` prefix (with the single space) followed by the identifier,
with no leading indentation. It is metadata about the field's body and is
stripped before the body is handed to the content type's tooling.

### Resolution

To resolve a dang-typed string field:

1. Read the first line; if it is `#! dang <id>`, take `<id>`. Otherwise the field
   has an inherited type-provided default binding (below) or no dang type.
2. Look up `<id>` in the object's dang binding table, falling back to the
   type-provided default binding for that field.
3. The bound, pinned type reference is the field content's type. Strip the
   shebang line; the remainder is content of that type.

An identifier that resolves to **no binding is a hard error at commit**: an
object MUST NOT be committed carrying a `#! dang` shebang whose identifier has no
binding (per-object or type-default). This catches typos and stale references at
write time and keeps the store free of dangling dang references — unlike the
Phase-1 convention, which silently strips.

### Pinning

The binding's type lock reuses `markl.Lock[ids.SeqId, *ids.SeqId]` — the same
lock blob references and the object's own type use. Pinning is expressed in the
hyphence form (RFC 0001): `<type-id>` unversioned, or `<type-id>@<signature>`
pinned to a specific signed type version. Pinning is **optional**: a binding MAY
be unversioned or signature-pinned, exactly as blob and object references are
today. Signature pinning is RECOMMENDED for portability (a type pulled from a
public repo resolves to a verifiable identity), but is not required — local
convenience does not force signing.

## Storage

A dang binding SHOULD reuse the existing **object-reference + alias** substrate:
a named, pinned reference to another *object*, which for dang is a *type* object.
That is the closest semantic match and is already persisted in the stream-index
binary codec, so no new binary format is introduced.

An implementation MAY instead reuse the **blob-reference + alias** substrate
(`delta/objects/blob_reference.go`, whose `(TypeLock, alias)` half is exactly a
dang binding), leaving the blob digest unused. This is permitted but not
preferred, since the digest slot is vestigial for a type binding.

A new, dedicated dang-binding metadata structure is deliberately NOT chosen: it
would require the four-file `india/stream_index` codec work (issue #38) for no
semantic gain over the object-reference substrate. Either permitted substrate
round-trips through commit/read like every other metadata reference.

## Type-provided default bindings

Requiring every object to repeat `#! dang md` plus a `md → !md` binding is poor
ergonomics. A **`dang` key on the field definition**
(`FieldDefinition`, `alfa/type_blobs`) declares that a field is dang-typed and
names its default binding — e.g. the `!task` type declares its `body` field with
`dang = "md"`, bound to a pinned `!md`. An object then needs neither the binding
nor (optionally) the shebang; it inherits the type's default. A per-object
binding or an explicit `#! dang <id>` shebang overrides the type default. This
keeps the common case zero-friction while preserving per-object expressiveness.
(The exact serialization of the default's pinned target on the field definition
is an implementation detail — see Open questions.)

## Consumption

- **Formatters.** Resolve the binding and render the field's body as the bound
  type (e.g. the `!md` formatter / pandoc), replacing the strip-and-assume
  pipeline. The formatter no longer hardcodes "markdown."
- **Editors.** Use the bound type to drive syntax highlighting / filetype for the
  dang-typed region (the editor integration reads the binding rather than
  guessing).
- **Validation.** Optionally validate the field's content against the bound type
  (a future capability; not required by this FDR).

### Type layering

The type's blob-level `file-extension` / `vim-syntax-type` describe the **blob**
(e.g. `toml`). A dang binding describes the **field's content** (e.g. `md`).
Within a dang-typed field's region the **dang type is authoritative**; the
blob-level type applies everywhere else. Consumers (editors, formatters) MUST
prefer the dang type inside the region and the blob-level type outside it, so a
TOML blob with a markdown `body` highlights/formats as TOML with a markdown
island.

## Scope (this FDR)

This FDR specifies dang for **string fields** (the actionable `body`), with a
first-line `#! dang <identifier>` shebang, a metadata binding table, and a
type-provided default binding via a `FieldDefinition` key. It does NOT cover:

- dang on a whole **blob** (vs a string field) — deferred to future work;
- `dang` as an executable interpreter binary — deferred (below);
- multiple dang regions within one string;
- content validation against the bound type.

The shipped Phase-1 convention (formatter strips `#!dang`) is the migration
predecessor: once dang resolution lands, the formatter resolves the binding
instead of blindly stripping; the on-disk shebang line is forward-compatible
(the parser accepts the OS-interpreter `#! dang <id>` form).

## Future work

- **`dang` as a delivered interpreter binary.** The end state: `dang` is a real
  executable dodder ships in its output/materialized tree, and `#! dang <id>`
  (or `#! /path/to/dang <id>`) is an actual shebang that *invokes* it to perform
  dodder execution-resolution — resolving `<id>` to its pinned type and
  running the appropriate render/execute path. This turns a dang-typed string
  into something directly executable via the standard interpreter mechanism.
  Deferred; this FDR's near-term resolution is dodder-internal (dodder recognizes
  the `dang` token), designed to be forward-compatible with the binary form.
- **Whole-blob dang** — a blob (not just a field) carrying the shebang + binding,
  a generic typed-string container.
- **Content validation** against the bound type at commit.

## Open questions

- **Default-binding serialization.** The `dang` key on `FieldDefinition` names the
  identifier; how the default's *pinned target* is expressed on the field
  definition (bare type id, or a pinned lock) needs settling in the
  implementation plan.
- **Object-reference substrate specifics.** Confirm the object-reference+alias
  API can hold a *type-object* reference (vs a zettel/object reference) cleanly,
  or whether a light adapter is needed.
- **Shebang escaping.** How a body whose real first line legitimately begins with
  `#! dang ` is represented (escape, or rely on the field-default binding so no
  in-body shebang is needed).

## Alternatives considered

- **Keep the strip-only convention (Phase 1).** No resolution, no pinning, one
  implicit type. Rejected — dang exists to make the content type explicit, named,
  and stable.
- **Markdown-with-frontmatter.** Put the body as a markdown blob with a
  frontmatter block carrying the structured fields. Rejected during the
  actionable-types design (the blob is TOML-structured; prose lives in a typed
  string field — see `docs/plans/2026-06-29-merge-actionable-types-design.md`).
- **A dedicated dang-binding metadata structure.** Rejected in favor of reusing
  the object-reference substrate, to avoid binary-codec churn (issue #38).
- **A generic nested type system.** Full recursive typing of any value. Far
  heavier than the need; dang is the minimal "this string is content of type T"
  affordance.

## Related work

- The `zz-pandoc/filters/dodder-edit.lua` code-block mechanism formats fenced code
  blocks whose class matches `^!` (e.g. `!task`) by piping them through
  `dodder format-object`. That is the *inverse* direction — typed blocks embedded
  within a document — and is orthogonal to dang (a string field that *is* typed
  content), but the two share the goal of scoped typefulness and could eventually
  share resolution machinery (and, later, the `dang` interpreter binary).

## References

- RFC 0001 — hyphence format (the pinned `<id>@<sig>` form a dang binding uses).
- RFC 0002 — markl IDs (the lock/signature identities).
- FDR-0017 — type-defined field index (the fields a dang-typed string lives in).
- `delta/objects/blob_reference.go`, `hotel/stream_index/binary_{encoder,decoder}.go`,
  `romeo/local_working_copy/blob_tree_materializer.go` — the named-pinned-reference
  substrate.
- `docs/plans/2026-06-29-merge-actionable-types-design.md` — the actionable
  `body` field and the Phase-1 `#!dang` convention this FDR replaces.
- Issue #296.

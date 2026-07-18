# dodder#374(b): `_base` pinning and the base/patch/live three-way merge

## Context

cutting-garden RFC 0015 ("The organize document dialect", revised
2026-07-18, `docs/rfcs/0015-organize-dialect.md` in the cutting-garden
repo) specifies dodder's organize as the reference substrate for an
upstreamed document dialect. Three of its four dodder-alignment items
are already implemented and merged on this branch: (a) space-separated
headings (comma dropped, no legacy acceptance), (c) `_dry-run=true` as
a `_`-reserved settings field with `% dry-run:true` kept as a
deprecated read-only alias, (d) organize-text(7)'s removal-semantics
documentation corrected to describe today's actual behavior (deleting
a line strips the *invoking query's* selection tag(s), not the
`-group-by` grouped dimension).

This plan is item (b): organize documents pin their generated ground
form as a content-addressed blob (`- _base=@<digest>`), and apply
becomes a base/patch/live three-way merge, superseding today's
live-only fork-overlay diff (`ChangesFromResults` /
`RemoveFromTransacted` in `go/internal/kilo/orgie/changes.go` and
`metadata.go`). Mandatory, no legacy mode: an organize document
without `_base` is invalid.

Sasha's 2026-07-18 ruling (relayed via pennywise, now in RFC 0015's
"Deletion semantics by grouped-ness" section) settles the deletion
question this plan had flagged as open:

- **Grouped document** (generated with `-group-by`): line deletion
  empties the **grouped dimension only** (membership-∅ for
  `write:many`, no-op for `write:one`). Selection predicates (the
  query used to invoke `organize`) are **never** written by a grouped
  document.
- **Ungrouped document**: line deletion means **removal from the
  selection**, evaluated per selection term through the mapping's
  declared writability. A `write:many` term (dodder's
  `organize tag-5` workflow) is removed — today's actual behavior,
  confirmed correct and principled, not a quirk to fix. A non-writable
  or underdetermined term (a field predicate) produces an
  **unresolved intent** instead of being silently dropped or flatly
  rejected.

## Out of scope

- The mergetool UI for interactively resolving base/live drift
  conflicts and unresolved intents (RFC 0015 "Deferred": "Mergetool
  (near)"). v1 reports both as structured rejections; this plan
  specifies that shape but not an interactive resolver.
- Aliased short object ids (RFC 0015: "reserved-but-unspecified").
- Combined filter+edit mode (RFC 0015: "Filter-mode documents are
  selection-only in v1").
- Cross-dimension nesting (RFC 0015: "far-future/never").
- `PartialTerm` / dependent-dimension heading sugar (`date=` /
  `=2026-07-22` composition) — RFC 0015's headings section describes
  this generally for the cutting-garden dialect; dodder's grouping
  today is tag-only (`-group-by` takes tag prefixes,
  `go/internal/kilo/orgie/options.go:90-109`), and this plan does not
  extend dodder's own facet system to non-tag dimensions. Out of scope
  until dodder has a non-tag groupable dimension to hang it on.
- Migrating `hide` or `checkin`'s `delete` OptionComments to the
  `_`-field spelling. Only `_base` and `_allow-deletion` are added as
  new settings fields by this plan; `_dry-run` is already done.
- Full RFC 0002 field-line grammar (arbitrary `- key=value` /
  `- key=value @digest` typed edges) for organize object lines.
  Existing box-format field syntax on object lines
  (`- [one/uno !task status=done]`) is untouched; this plan only adds
  the two new **document-level** `_`-fields (`_base`, `_allow-deletion`).

## 1. Grammar prerequisite inventory

Two new document-level field shapes must parse, both `_`-prefixed
settings fields per the mechanism dodder#374(c) already built
(`go/internal/kilo/orgie/option.go`'s `OptionCommentSettingsField`
interface, `metadata.go`'s `addTagOrSettingsField` dispatch):

- **`- _base=@<digest>`** — an **id-less, digest-valued** field. This
  is hyphence RFC 0002's `FieldRHS = DigestTerm` alternative (no
  `FieldValue` before the `@digest` — contrast the *locked* form
  `key=value @digest`, which has both). RFC 0002 itself flags this
  form as forward-declared against a trellis grammar change (`Value`
  gaining a `DigestTerm` alternative) that pennywise's 2026-07-18
  message confirms has since merged upstream. dodder does not need to
  adopt trellis grammar to support this — dodder's own dash-line
  dispatch (`metadata.go`) only needs to recognize `_base=@<hash>` as
  a distinct shape from `_dry-run=true`, which is a **new,
  net-new parsing case**: today's `addTagOrSettingsField` treats
  everything after `=` as an opaque string handed to
  `OptionCommentSet.Set` → `values.Bool.Set` (dry-run's own `Set`).
  `_base` needs a digest-shaped value, not a bool.
- **`- _allow-deletion=true`** — a plain boolean settings field,
  structurally identical to `_dry-run=true`. No new grammar; reuses
  the exact `_key=true` mechanism dodder#374(c) built, with a new
  `OptionCommentAllowDeletion` type (see §7).

**What already exists to build on** (confirmed by direct source read,
not assumed):

- **Blob-write-and-digest primitive**: `writeBlobContent` pattern in
  `go/internal/sierra/repo_actions/update_object.go:124-148` —
  `repo.GetEnvRepo().GetDefaultBlobStore().MakeBlobWriter(nil)`, write
  bytes, `blobWriter.GetMarklId()` returns the digest directly. Used
  by `write_new_zettels.go:95`, `update_object.go:91`,
  `oscar/store/field_writer.go:186-198`. Directly reusable for writing
  the `organize-base-v1` blob at generation time.
- **`@digest` parsing/formatting**: doddish's `OpMarklId` (`@`) is
  wired for hyphence blob-reference lines already —
  `echo/object_metadata_fmt_hyphence/text_parser2.go:77-78,138-232`
  (parse) and `formatter_components.go:269-271` (format). It is
  **not** wired for field *values* — field value parsing/encoding
  today is string-only (confirmed: `alfa/type_blobs/field_definition.go`
  parses `Kind` as one of `string | enum | bool | u32 | s32 |
  list<string>`, `0/fields/main.go:19-26` — no markl/digest kind
  exists). `_base=@digest` does not need a *type-declared field kind*
  though — it's a **document-metadata settings field**
  (`OptionComment`-family), not a projected object field, so it can be
  parsed directly in `metadata.go`'s dash-line dispatch without
  touching the field-kind system at all. Scope this narrowly: a new
  `OptionCommentBaseDigest` type whose `Set(v string)` parses `v` as
  `@<hash>` via `markl.Id.Set` (or equivalent), not a general
  digest-valued field kind. Extending `go/internal/0/fields` to have a
  markl-valued `Kind` is out of scope for this plan.
- **`ids.TagStruct`-style `Set` validation**: N/A for `_base` (not a
  tag); the new `OptionCommentBaseDigest.Set` validates the `@digest`
  shape itself and returns a clear parse error otherwise (mirroring
  `OptionCommentDryRun.Set`'s use of `values.Bool.Set` for validation).

## 2. `organize-base-v1` as a dodder type

**Confirmed (independent research, cross-checked against
`docs/rfcs/0003-cutting-garden-receipt-ingest.md:85-86`, which states
verbatim: "It is created as an ordinary type object and MUST NOT be
added as a builtin — no change to `internal/bravo/ids/types_builtin.go`
is required."):** `organize-base-v1` is a **user-space type object**,
created the same way `!img`, `!pdf`, etc. are — a `.type` file checked
in via `dodder checkin`, or (for a type dodder itself depends on
structurally) via genesis, the same pattern `!task`/`!chore` use
(`docs/plans/2026-04-06-task-type-genesis-and-haustoria-fields.md`,
`local_working_copy/genesis.go:195-298`,
`type_blobs.DefaultTaskType()`-style constructor). **No change to
`go/internal/bravo/ids/types_builtin.go`.**

Genesis vs. ad-hoc creation: `organize-base-v1` is **not** needed by
every dodder repo — a repo that never runs `organize` never needs the
type. Following the `!task`/`!chore` precedent's *opt-in* posture
(`BigBang.IncludeBuiltinActionableTypes`, default `false`), this plan
proposes a symmetric `BigBang.IncludeOrganizeBaseType` flag (default
`false`) rather than a hard genesis dependency, OR — simpler, avoiding
a new genesis flag entirely — **lazy creation**: the first
`dodder organize` invocation that needs to write a base blob checks
whether `!organize-base-v1` exists (`show '!organize-base-v1:t'`) and
creates it if not, the same way a user would `dodder checkin` a new
`.type` file. This defers the genesis-flag decision entirely and
matches "organize is ephemeral action" (RFC 0015's own framing) better
than baking a structural type into every fresh repo.

**Resolved (pennywise/Sasha review, 2026-07-18): lazy creation,
approved.** Implementation note from the review: generation already
writes (the base blob itself), so the implicit type-create sits at the
same point in the flow — make the check-then-create **idempotent, not
racy**: if `!organize-base-v1` doesn't exist, attempt creation, but
treat "already exists" (a concurrent `organize` invocation created it
first) as success rather than an error, not a
check-then-create-unconditionally sequence that assumes exclusive
access.

**Type-definition blob content** (`TomlV2`,
`go/internal/alfa/type_blobs/toml_v2.go:9-24`):

```toml
file-extension = "organize"
vim-syntax-type = "markdown"
binary = false
```

**Amended per pennywise's 2026-07-18 review**: the base blob's content
is the serialized organize/espalier form (§9), not TOML — the
type-definition's `file-extension`/`vim-syntax-type` must describe
that, not toml-for-non-toml. `organize` extension (matching the
document format's own name) and `markdown`-ish syntax (headings +
box-format object lines, closest existing highlighter) are reasonable
defaults; exact values are a small, low-stakes implementation
decision, not load-bearing for the rest of this plan.

The base blob's own content format is **not** re-derived here — see
§9 (espalier serialization checklist) for what it actually contains.
No `[[fields]]`, no `fields-reader`/`fields-writer` — the base blob is
opaque, dereferenced whole (RFC 0015: "base ... what the user was
shown"), not field-projected into the index like `!task`.

## 3. Base blob lifecycle

**Write path (generation):**

1. `organize`'s existing document-generation step
   (`go/internal/kilo/orgie/options.go:189-245`, `(Options).Make()`)
   already builds the full `Assignment` tree from the query result
   before any text is emitted.
2. **New step**: serialize that tree (or the flat espalier-stream
   projection of it — see §9) to bytes in the base blob's canonical
   form, write via the `writeBlobContent` pattern (§1) against
   `repo.GetEnvRepo().GetDefaultBlobStore()`, obtain the digest.
3. `Metadata.WriteTo` (`metadata.go:161-...`) emits
   `- _base=@<digest>` via the same
   `OptionCommentSettingsField`-driven dispatch dodder#374(c) already
   generalized — a new `OptionCommentBaseDigest` implementing
   `IsSettingsField() bool { return true }`, added via
   `AddPrototypeAndOption` unconditionally (not gated on a CLI flag
   like `-dry-run` is — `_base` is **required**, not optional; see
   §8).

**Which store**: the repo's **default blob store**
(`GetDefaultBlobStore()`), same as any other object's blob — not a
separate "organize session" store. The base blob is content-addressed
and immutable like every other dodder blob, written as a **bare
blob with no owning object** (no referencing object of type
`organize-base-v1` or otherwise — see the GC discussion below).

**Resolved (pennywise/Sasha review, 2026-07-18): bare blob,
collectable, no owning object.** Creating a referencing object per
organize session was rejected as polluting history with ephemera,
against "organize is ephemeral action." A GC'd base is exactly the
designed `ErrBaseUndereferenceable` path (§3 below) — regenerate and
retry is the acceptable worst case, not a failure mode to engineer
around.

The review asked for one confirming research line before this is
final: **does dodder's blob store actually collect unreferenced blobs
today, and on what trigger?** Checked directly (`fsck.go:219-261`,
`clean.go:113`, `oscar/store/reindex.go:32-229`, plus a targeted
search for any `DeleteBlob`/`RemoveBlob`/`ExpireBlob`-shaped function
across the whole `go/` tree): **no such mechanism exists.** `fsck` is
verification-only (checks blob existence and reports dangling
references, per issue #330's fix — never deletes).
`clean` deletes checked-out **working-copy files**, never touches the
content-addressed blob store. `reindex` rebuilds indexes from the
inventory list, no reachability sweep. There is no mark-and-sweep, no
scheduled or triggered blob deletion anywhere in dodder today. **In
practice this means the risk window this plan was designed to guard
against (§3's `ErrBaseUndereferenceable`) does not currently exist for
locally-written blobs** — a base blob, once written, persists
indefinitely until dodder gains an actual blob-GC feature. The design
(bare blob, no owning object, loud rejection on
undereferenceable-digest) is still correct and forward-compatible with
a future GC landing; it just isn't exercised by anything that exists
today. `ErrBaseUndereferenceable` remains worth implementing (a remote
peer's copy of the repo, a blob store that was never synced, or manual
blob-store surgery could still produce it) — the risk is real, just
not GC-triggered under current dodder behavior.

**Dereference at apply**: `-mode commit-directly` / `interactive`'s
read path (`repo_actions/read_organize_file.go`,
`organize_roundtrip.go`) gains a step: parse `_base=@<digest>` from
the patch's metadata, then **read the blob at that digest** from the
blob store to reconstruct `base` (the third input to the three-way
diff, alongside `patch` = the just-parsed document and `live` = a
fresh query against the store — see §4).

**Failure mode for an undereferenceable `_base`** (stale/GC'd/typo'd
digest): **loud, structured rejection** — a distinct error type (e.g.
`ErrBaseUndereferenceable{Digest: ...}`), not a generic blob-not-found
error, so the apply engine's error-reporting path (§6, §8) can render
it with the "how to regenerate" guidance §8 requires. Surfaced at the
very start of apply, before any diff computation, since nothing else
can proceed without `base`.

## 4. Three-way engine placement

**What changes in `changes.go`/`metadata.go`:**

- `ChangesFromResults` (`changes.go:172-214`) currently takes
  `OrganizeResults{Before, After, Original}` where `Before` is a
  **freshly regenerated** document (not the base blob) and computes
  `c.Removed` = `Before − After`, then calls
  `results.Before.RemoveFromTransacted(sk)` (query-selection-tag
  stripping, dodder#374(d)'s documented current behavior) for each
  removed object.
- This plan replaces that two-input diff (`Before`/`After`) with the
  three-input diff RFC 0015 specifies: **`base`** (dereferenced
  `_base`, §3), **`patch`** (the edited document, today's `After`),
  **`live`** (a fresh query against the store at apply time — today's
  `Before` conflates "freshly regenerated" with "current store state,"
  which are the same thing only when nothing else has changed the
  store between generation and apply; RFC 0015 treats them as
  distinct so drift is detectable).
- `patch − base` = intent (structural: moves, membership changes,
  field edits, creations, adoptions) — replaces `Removed`/`Changed`
  computation in `changes.go:191-211`.
- `live − base` = drift. Drift on fields the patch also touches is a
  loud-rejection conflict in v1 (§6); drift on untouched fields merges
  silently; convergent edits are idempotent no-ops.
- `RemoveFromTransacted` (`metadata.go:73-83`) is **not deleted** —
  the ungrouped-document, `write:many`-term-removal case (§6) still
  needs "strip these specific tags from this object," which is what
  `RemoveFromTransacted` already does. It becomes one execution path
  within the new deletion-branch logic (§6), invoked conditionally
  instead of unconditionally.
- **Composition with dodder#374(a)/(c)/(d)**: no conflict. (a)'s
  space-separated headings and (c)'s `_dry-run`/`_base` settings-field
  parsing both operate one layer below this (`Metadata.ReadFrom`,
  already generalized via `OptionCommentSettingsField` so `_base` and
  `_allow-deletion` are pure additions, not further special-casing).
  (d)'s documented current removal semantics becomes the *ungrouped*
  branch's `write:many` case verbatim — the doc already describes it
  precisely correctly for that one case; it needs a "grouped documents
  behave differently" addendum once (b) ships (tracked as a
  documentation follow-up in Implementation order, not blocking this
  plan).

**New home for the three-way logic**: a new file,
`go/internal/kilo/orgie/three_way.go` (or similar), rather than
growing `changes.go` further — `ChangesFromResults` becomes a thin
wrapper that dereferences `_base` (§3) and delegates to the new
engine, keeping the base/patch/live diff itself independently
testable without the surrounding command plumbing.

## 5. Grouped-detection

Per Sasha's ruling (Context, above) and RFC 0015 explicitly: **read
grouped-ness from the BASE blob's own heading structure, never
inferred from the patch.** The base blob is the canonical "what did
generation produce" record — if it was generated with `-group-by`, its
serialized form (§9) carries that heading tree; if generated without
`-group-by` (flat/ungrouped), it doesn't. This is a property of
`base`, checked once at the start of apply (§4), and threaded through
to the deletion-branch decision (§6). It is **not** re-derived from
counting headings in `patch` — a user could delete every heading from
their edit, and that must still be treated as "this was a grouped
document" for deletion-semantics purposes (RFC 0015: "moves and
memberships changes ... computed structurally" against base, not
guessed from what patch happens to still contain).

**Resolved per pennywise's 2026-07-18 review (OQ3)**: recorded as a
`_`-framework settings field in the base blob's OWN metadata — the
base blob is itself a hyphence document (type line `! organize-base-v1`
last, per RFC 0001 canonical order), so its generation parameters are
document-level settings fields, the exact same mechanism as `_base`/
`_dry-run`/`_allow-deletion` on the organize document proper:

```
- _group-by="priority,w"
```

Single field, value quoted and comma-joined to preserve `-group-by`'s
order (RFC 0001's order-independence rule for metadata lines forbids
spreading an ordered list across repeated `_group-by=...` lines — a
decoder is free to reorder metadata lines, so the ordering must live
inside one field's value, not across lines). Absent = ungrouped, no
new mechanism beyond what `_dry-run`/`_allow-deletion` already
established. New `OptionCommentGroupBy` (`option.go`), string-valued
(not bool like `_dry-run`/`_allow-deletion`), `IsSettingsField() →
true`.

## 6. Deletion branch + unresolved intents

Implements RFC 0015's "Deletion semantics by grouped-ness" (revised
2026-07-18) directly:

```
if base.wasGrouped {
    // write:many (dodder's only grouping mechanism today — tags):
    // absence from patch = membership-∅ for the grouped dimension.
    // Selection predicates (the query that produced this session)
    // are NEVER written here.
    for each object in (base - patch):
        clear only the grouped-dimension tag(s) this object had
        // (the -group-by-matching tag(s), NOT RemoveFromTransacted's
        // query-selection tags)
} else {
    // ungrouped: deletion = removal from selection, evaluated
    // per selection term through mapping writability.
    for each object in (base - patch):
        for each selection term (the query used to invoke organize):
            if term is write:many (a tag predicate):
                RemoveFromTransacted-style: strip that specific tag
                // (today's dodder#374(d)-documented behavior,
                // preserved verbatim as the principled case)
            else:
                // field predicate, or otherwise non-writable/
                // underdetermined
                emit UnresolvedIntent{Object, Term, Options}
}
```

**Dodder has no non-tag selection terms today** (doddish queries are
tag/type/id predicates; dodder#374(c)'s field-match syntax
(`key=value`) exists but isn't yet a common `organize <query>`
invocation shape) — so in practice, v1's `UnresolvedIntent` path may
see little real traffic in dodder specifically, unlike cutting-garden
substrates with field-predicate-heavy queries (jira, caldav). Still
implement it per spec (RFC 0015 treats this as substrate-agnostic
apply-engine behavior, not dodder-specific), since a future
field-predicate-driven `dodder organize 'status=done'` invocation
would hit exactly this path.

**`UnresolvedIntent` shape** (v1, non-interactive — RFC 0015: "v1
reports them as structured rejections enumerating each intent with its
options"):

```go
type UnresolvedIntent struct {
    Object  ids.ObjectId
    Term    string // the selection term that couldn't be resolved to a write
    Options []string // e.g. ["supply a replacement value", "skip this object", "abort"]
}
```

**Forward-compat note (pennywise's 2026-07-18 review)**: `Options
[]string` is prose-only, fine for v1's structured-rejection output,
but cutting-garden#147 is pushing toward typed error identities with
stable identifiers rather than free-text options. Shape v1's output so
that upgrade is additive, not breaking: keep `Options` as the
human-readable rendering, but don't derive it by string-formatting
inside the diff engine — build each option from a small internal enum/
struct first (`type resolutionOption struct { Kind string; Label
string }`-shaped, kinds like `"supply-replacement"`, `"skip"`,
`"abort"`) and render `Options []string` from that at the output
boundary. Adding a stable `Kind` identifier to `UnresolvedIntent`
itself later is then a field addition, not a rewrite of how options
get produced.

Batchability (RFC 0015: "identical intent shapes are batchable — one
prompt resolving the same question across N objects") is a
presentation-layer concern for the eventual mergetool (out of scope,
§ Out of scope) — v1's structured-rejection output should still group
identical `(Term, Options)` shapes when reporting, so the *data* is
batch-ready even though v1 has no interactive resolver to consume it
that way.

## 7. `_allow-deletion` gating

New `OptionCommentAllowDeletion` (`option.go`), structurally identical
to `OptionCommentDryRun` (bool-valued, `IsSettingsField() → true`),
registered unconditionally (not CLI-flag-gated like `-dry-run` — a
document either declares it or doesn't; there's no
`repo.GetConfig().IsAllowDeletion()`-style ambient state to gate on,
unlike dry-run which mirrors a real CLI flag).

**Deletion gates** (RFC 0015 "Deletion" section, all must pass):

1. Document carries `- _allow-deletion=true`.
2. Apply computes the deletion set: objects in **both** base and the
   live substrate, absent from patch, **beyond** membership-∅
   semantics (i.e. the object isn't just losing its grouped-dimension
   tag or a selection tag per §6 — it's wholly gone from the document
   with no remaining trace, AND `_allow-deletion` is set, which is
   what elevates "gone from the document" from a §6 tag-removal to an
   actual substrate deletion).
3. **Explicit post-editor confirmation** required (interactive mode:
   a prompt after the editor closes, before commit — mirrors the
   existing `repo.Confirm(...)` pattern already used for
   large-change-count confirmation,
   `organize_options.go:60-72`).
4. In `commit-directly` mode, an **additional CLI flag** is required
   (RFC 0015: "double assertion for the scripted path") — proposed
   `-confirm-deletion` on the `organize` command, checked in addition
   to the document's `_allow-deletion=true` field.

**Without `_allow-deletion`**: line absence **never** deletes from the
substrate — it only ever means §6's tag-removal/membership-∅/
unresolved-intent outcomes. This is the safe default; `_allow-deletion`
is the opt-in escape hatch into actual object deletion.

**`-dry-run` interaction (resolved, pennywise/Sasha review,
2026-07-18): orthogonal, as this plan originally assumed.** `_base`
presence-validation (§8), dereference (§3), and the full three-way
diff (§4) always run regardless of `-dry-run` — dry-run skips only the
**write** phase. This has a direct consequence for gate 3 above: the
post-editor deletion confirmation is *also* skipped under `-dry-run`,
since there's nothing to confirm when nothing will be written — but
the deletion **set still computes and reports** (a dry-run apply shows
what *would* be deleted, same as it shows what *would* be
tagged/committed today). RFC 0015's invocation modes (interactive /
commit-directly / output-only) are an orthogonal axis from
`-dry-run`; this composes without special-casing.

## 8. Hard cutover UX

"`_base` or GTFO" (Sasha, via the originating issue) — a document
without `- _base=@<digest>` in its metadata is **invalid**, full stop,
no legacy fork-overlay fallback. The error must be immediately
actionable, not a generic parse failure:

```
error: this organize document has no `_base` field.

organize documents are ephemeral action, not durable artifacts --
edits can only be applied against the exact document `organize`
generated. Regenerate with:

    dodder organize <your original query>

then make your edits in the freshly generated document.
```

Implementation: a dedicated check at the very start of the
read/apply path (`read_organize_file.go` or
`organize_roundtrip.go`), before any other parsing — fails fast with
this message rather than surfacing as a downstream "unresolved intent"
or diff error. Distinct from §3's "undereferenceable digest" error
(that's "you have a `_base`, but it's stale/wrong"; this is "you have
no `_base` at all") — both need the same "regenerate" guidance but are
different failure points (missing-field vs. field-present-but-broken).

## 9. Espalier serialization checklist

cutting-garden FDR 0022 (`docs/features/0022-trellis.md:114-120` in
the cutting-garden repo — confirmed this is cutting-garden's own FDR
0022, unrelated to dodder's FDR 0022 "Config as a non-object") flags
four "in-hoc" (discovered against real streams, not designed upfront)
unknowns for nested espalier-stream serialization. Position per item,
for the `organize-base-v1` blob's specific use case (a snapshot of one
`organize` session's generated tree — objects plus the heading
structure that grouped them, per §5):

1. **Node deduplication when reachable via multiple in-edges** — a
   dodder object appearing under two headings (RFC 0015's confirmed
   `write:many` clone-per-match rendering, §
   "Write descriptors") is the concrete case here. **Position**: the
   base blob does NOT deduplicate — it mirrors what the generator
   actually rendered, i.e. the object appears once per heading it was
   grouped under, exactly as the *displayed* document does (dodder's
   `Subset`/clone-per-match mechanism confirmed in earlier research,
   `set_prefix_transacted.go:202-260`). Deduplication would make the
   base blob NOT a faithful record of "what the user was shown," which
   RFC 0015 requires ("base ... what the user was shown").
2. **Inline-vs-by-reference children** — N/A for dodder's case in v1:
   organize's tree has no object-to-object containment edges the way
   a caldav/newsblur substrate might (a story with a content child).
   Every object line in a dodder organize document is a leaf
   (box-format interior); the only "nesting" is heading structure,
   not object-to-object edges. Position: no inline-vs-reference
   decision needed for dodder's base blob; revisit if dodder objects
   ever gain reference-valued fields serialized into organize output.
3. **Cycle representation** — N/A for the same reason as (2): no
   object-graph edges in scope, so no cycles possible in dodder's base
   blob.
4. **Ordering significance** — **position: ordering is NOT
   semantically significant** for the base blob's diff purposes (the
   three-way diff in §4 operates on the base's *tag membership and
   heading structure*, not line order), but the base blob's canonical
   serialization SHOULD still be deterministically ordered (e.g. sort
   objects within a heading the same way `writer.go`'s existing
   `a.Objects.Sort()` does, `writer.go:92`) purely so re-serializing
   identical content produces a byte-identical digest — not because
   order carries meaning to the diff engine.

## 10. Test strategy

Bats lanes (`zz-tests_bats/current_version/organize.bats` and
possibly a new `organize_base.bats` given the scope — follow this
file's own file-size/focus precedent, split if `organize.bats` would
grow unwieldy):

- **Round-trip**: generate → note `_base` digest → apply unedited
  (patch == base) → idempotent no-op, live unchanged.
- **Grouped deletion**: generate with `-group-by`, delete a line,
  apply → grouped-dimension tag cleared, query-selection tags
  untouched (contrast with dodder#374(d)'s documented ungrouped
  behavior — this is the NEW grouped case).
- **Ungrouped deletion, writable term**: generate without
  `-group-by` from a tag query, delete a line, apply → today's
  `organize_dry_run`-style tag-stripping behavior, now routed through
  the three-way engine instead of the old two-input diff — same
  observable outcome, different code path, needs its own coverage
  since dodder#374(d)'s existing tests exercise the OLD path.
- **Ungrouped deletion, unresolved intent**: generate from a field-
  predicate query (`dodder organize 'status=done'`), delete a line,
  apply → structured rejection, not silently dropped or hard error.
- **Drift/conflict**: generate, mutate the live object out-of-band
  (simulating another process/session), edit the patch touching the
  SAME field, apply → loud rejection. Edit an UNTOUCHED field
  concurrently → silent merge.
- **`_allow-deletion` gates**: wholly-remove a line without
  `_allow-deletion` → no substrate deletion (falls to §6 tag-removal
  instead). With `_allow-deletion=true` but no post-editor
  confirmation (simulate decline) → no deletion. With confirmation
  → deletion. `commit-directly` without `-confirm-deletion` CLI flag
  → no deletion even with the field set.
- **Cold-cutover error**: feed `-mode commit-directly` a document with
  no `_base` field at all → the exact regenerate-guidance error text
  from §8, not a generic parse failure.
- **Undereferenceable `_base`**: feed a document with
  `_base=@<digest-that-does-not-exist>` → the distinct
  `ErrBaseUndereferenceable`-style loud rejection from §3.
- **Espalier round-trip**: base blob written at generation,
  dereferenced at apply, produces byte-identical re-serialization
  (pins §9's ordering-determinism position).

Go unit tests (mirroring `go/internal/kilo/orgie/option_test.go`'s
pattern from dodder#374(c)) for the three-way diff engine itself
(§4's new file) — `patch - base`, `live - base`, and the
grouped/ungrouped branch dispatch (§5, §6) — independently of the
surrounding command/blob-store plumbing, using in-memory
`Assignment`/`Metadata` fixtures rather than a full bats round trip
for the pure-diff-logic cases.

## Open Questions — all resolved (pennywise/Sasha review, 2026-07-18)

All four questions originally raised here were reviewed and ruled on
in full; the resolutions are folded inline into their originating
sections rather than repeated here:

1. **`organize-base-v1` creation** — resolved in §2: lazy, idempotent
   check-then-create.
2. **Base blob GC reachability** — resolved in §3: bare blob, no
   owning object; confirmed by direct research that dodder has no
   blob-GC mechanism today, so the risk window is currently
   theoretical (still worth implementing `ErrBaseUndereferenceable`
   for non-GC undereferenceable cases).
3. **`-group-by` value recording** — resolved in §5: a
   `_group-by="priority,w"` settings field on the base blob's own
   metadata, same mechanism as every other `_`-field this plan and
   dodder#374(c) use.
4. **`_base`/`-dry-run` interaction** — resolved in §7: orthogonal,
   confirmed. Validation/diff always run; only the write (and its
   dependent post-editor deletion confirmation) is skipped.

Plan approved for implementation with these rulings plus two
amendments (§2's type-definition content corrected to match the
organize/espalier serialization rather than TOML; §6's
`UnresolvedIntent.Options` given a forward-compat note for
cutting-garden#147's typed error identities) — both folded into their
sections above.

## See also

- cutting-garden RFC 0015 (`docs/rfcs/0015-organize-dialect.md`,
  cutting-garden repo, revised 2026-07-18) — the specification this
  plan implements.
- hyphence RFC 0002 (`docs/rfcs/0002-content-grammar.md`, hyphence
  repo) — the `_base=@digest` id-less field-value grammar.
- cutting-garden FDR 0022 (`docs/features/0022-trellis.md`,
  cutting-garden repo) — the espalier-serialization checklist §9
  answers.
- dodder#374 (parent tracking issue), #373 (`_genre`, unrelated but
  adjacent), #375 (format-organize wiring bug, discovered as a
  byproduct of #374(c), not blocking this plan).
- `docs/plans/2026-04-06-task-type-genesis-and-haustoria-fields.md` —
  precedent for opt-in genesis of a user-space type (§2's `!task`/
  `!chore` comparison).
- `go/internal/kilo/orgie/CLAUDE.md` — package overview for the
  `changes.go`/`metadata.go`/`option.go` files this plan modifies.

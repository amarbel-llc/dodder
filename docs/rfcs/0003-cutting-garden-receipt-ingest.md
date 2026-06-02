---
date: 2026-06-02
status: proposed
---

# Cutting-Garden Receipt Ingest

## Abstract

Cutting-garden captures sources --- filesystem directories, yt-dlp videos, and
(future) chrest web archives --- into madder blob stores, emitting a
deterministic, content-addressed *receipt* blob per captured object. This
document specifies how dodder ingests those receipts as versioned objects: each
captured object becomes one dodder zettel, and each subsequent capture of the
same object becomes a new historical version of that same zettel. Re-capturing
a directory or URL behaves like editing a file and committing it in git --- the
latest receipt wins, prior receipts remain in history, and an unchanged
re-capture is a no-op.

## Introduction

Cutting-garden and dodder are sibling tools over a shared substrate: both
depend on madder, the content-addressed blob store, and neither depends on the
other. A capture run writes a receipt blob (type
`cutting_garden-capture_receipt-fs-v1`) into a madder store and records an
audit line in `captures.log`. Today those receipts have no home in a user's
knowledge graph: there is no way to ask "what have I captured from this
directory over time?" or to treat a series of captures of one URL as the
version history of a single thing.

Dodder already models exactly this shape. Every object (`sku.Transacted`)
carries a predecessor link (`sigMother`) forming a git-like version DAG, blobs
are content-addressed and deduplicated, and committing metadata identical to an
object's current version is a no-op. The missing piece is a binding that maps a
capture's *target* --- a directory path or a URL --- onto a stable dodder object
identity, so repeated captures land as successive versions rather than
unrelated objects.

The integration runs in one direction: dodder ingests cutting-garden output.
Crucially, dodder never decodes a receipt; it records a reference to it. This
keeps cutting-garden's wire formats private to cutting-garden and reduces the
coupling to two public surfaces: the `captures.log` text contract and the
madder blob addressed by its markl ID.

## Requirements Language

The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL NOT", "SHOULD",
"SHOULD NOT", "RECOMMENDED", "MAY", and "OPTIONAL" in this document are to be
interpreted as described in RFC 2119.

## Specification

### The Foreign Key

Each capture targets one *object*, identified by cutting-garden's `Root`:

- a filesystem capture's object is the resolved **directory path**;
- a yt-dlp capture's object is the source **URL**;
- a chrest capture's object (future) is the archived **URL**.

This identifier is the *foreign key* that binds a capture to a dodder object.

The foreign key MUST NOT be recovered from the receipt blob. Cutting-garden
applies a single-root collapse (`internal/capture/capture.go`) that normalizes
`Root` to `"."` inside single-root receipts, so the original path or URL is
absent from the receipt bytes. The authoritative source for the foreign key is
cutting-garden's audit trail, `$XDG_STATE_HOME/cutting-garden/captures.log`
(`internal/capture/capture_log.go`), whose NDJSON records carry it directly:

    {"ts": "...", "receipt_id": "...", "store_id": "...", "roots": ["..."]}

`roots` holds the capture-root arguments in order, before collapse or
normalization. For a single-root receipt the foreign key is `roots[0]`.

### Object Model

For each captured object dodder maintains exactly one **zettel** (genre `z`,
with an auto-allocated identifier). The zettel's stable identity across captures
is what makes repeated captures successive versions rather than new objects ---
matching the intuition that a same-directory re-capture "overwrites the given
zettel" while history is preserved.

The zettel is typed `!cutting_garden-receipt`, a user-space dodder TOML type
(`type_blobs.TomlV2` schema: `file-extension`, `mime-type`, optional `fields`).
It is created as an ordinary type object and MUST NOT be added as a builtin ---
no change to `internal/bravo/ids/types_builtin.go` is required.

The zettel's **blob** is a small TOML "join" document, the authoritative,
human-readable record of the binding:

    foreign_key  = "https://www.youtube.com/watch?v=..."  # or a directory path
    source_kind  = "ytdlp"                                # file | ytdlp | chrest
    store_id     = ""                                     # "" = default store
    captured_at  = "2026-06-02T00:00:00Z"
    receipt_id   = "<markl-id of the receipt blob>"

`receipt_id` is a human-readable mirror of the machine reference described next.

### Receipt Linkage

The receipt blob is linked from the zettel's metadata via `BlobReferences`
(`go/internal/delta/objects/blob_reference.go`), which is dodder's facility for
referencing an external content-addressed blob by digest plus its own type:

Linking is a two-call sequence (there is no single "add reference with type and
alias" constructor):

- `Add(id markl.Id, typeLock markl.Lock[ids.SeqId, *ids.SeqId])` --- the type
  lock carries the receipt's own type tag,
  `cutting_garden-capture_receipt-fs-v1`. This is the "the receipt already has a
  type, use it directly" property: dodder records the receipt's native type
  rather than reinterpreting its bytes.
- `SetAlias(id markl.Id, alias string) error` --- a stable alias (`"receipt"`).
  This MUST be called after `Add`; it errors if the key was not already added.

Callers typically reach these through the `objects.metadata` facade
(`AddBlobReference`, `SetBlobReferenceAlias`) rather than the raw
`BlobReferences` type.

`BlobReferences` is the correct mechanism rather than `ContainedObjects`: the
latter references *in-store* dodder objects by `SeqId`, whereas a receipt is an
*external* content-addressed blob keyed by markl ID.

This design adds no field to `objects.metadata` or `blobReferenceEntry`, and so
requires no binary-codec change (see issue #38). The `blobReferenceEntry` triple
--- key, type lock, and alias --- is verified to round-trip through the binary
stream index: the encoder/decoder in `hotel/stream_index`
(`binary_encoder.go`, `binary_decoder.go`) serializes and restores all three
parts, and the alias survival is covered end-to-end by the
`pull_direct_blob_reference_alias_survives` bats test (`pull.bats`) and the
box-format round-trip in `show.bats`. (Note: the codec lives in
`hotel/stream_index`, not `india/stream_index`, which does not exist; older
references --- including this repo's CLAUDE.md and issue #38 --- name the stale
path.) Coverage of the alias through the *binary* codec is currently
integration-test-only; a direct Go round-trip unit test would be cheap
insurance for the importer.

### Object Identity: Create vs. Update

To make a re-capture land on the existing zettel rather than a new one, *some*
binding from foreign key to object identity is required. Automatic resolution
--- deriving the existing zettel from the foreign key alone --- is explicitly
**out of scope for v1** and is deferred to its own feature design record (see
Future Work).

v1 instead uses dodder's existing **create/update** semantics, which require no
new resolution machinery:

- **Create** (no object-id supplied) --- the importer writes a new zettel via
  `repo_actions.WriteNewZettels` / `sku.Proto`.
- **Update** (object-id supplied) --- the importer loads and rewrites that exact
  zettel via `repo_actions.UpdateObject`, and `store.Commit` chains the new
  version onto the old (see Versioning and Deduplication).

In v1 the foreign-key-to-object-id mapping is therefore the **caller's**
responsibility: a re-capture becomes a new version only when the ingest carries
the object-id of the zettel that earlier captured the same foreign key. The
join blob's `foreign_key` remains the authoritative record of *what* was
captured; nothing in v1 indexes it for lookup.

### Future Work: Foreign-Key Resolution

Automatic resolution belongs in a dedicated FDR
(`docs/features/0017-type-defined-field-index.md`). The motivating idea is a
**type-system-defined field index**: an index, incrementally maintained as
objects are committed, over fields a type declares as indexable (e.g.
`foreign_key` on `!cutting_garden-receipt`). With that in place, an ingest could
resolve foreign key to object-id with no caller-supplied id, and a bare
re-capture of the same URL or directory would find its zettel automatically.

Candidate implementations to weigh in that FDR, retained here for the record:

1.  **Derived lookup tag.** A short digest of the normalized foreign-key string,
    e.g. `cg-source-<digest>`, added to the zettel; resolution is a tag-plus-type
    query against the existing query system. Note the constraint: `ids.Tag`
    validates against `^%?[-a-z0-9_]+$` and lowercases on parse
    (`go/internal/bravo/ids/tag.go`), so the digest MUST be encoded as lowercase
    `[a-z0-9_-]` (hex or lowercased base32, not a raw blech32/base64 markl
    digest, which would be rejected or silently lowercased).
2.  **Linear scan.** Enumerate `!cutting_garden-receipt` zettels and match
    `foreign_key`. Simplest; O(n) per import. Acceptable at small scale.
3.  **Type-defined field index.** The recommended direction: a foreign-key-keyed
    index generated incrementally and driven by type-declared indexable fields.
    Most direct lookup, the heaviest change, and the reason this is its own FDR.

### Versioning and Deduplication

Versioning is dodder-native and requires no special handling. Committing a
zettel whose identifier already exists causes `store.Commit`
(`go/internal/oscar/store/mutating.go`) to fetch the prior version
(`fetchMotherIfNecessary`) and chain `sigMother` to it, extending the history
DAG. If the new metadata is identical to the current version, the commit
short-circuits and no new version is written --- so re-importing an unchanged,
deterministic receipt is a no-op.

The import flow is therefore:

1.  Ensure the `!cutting_garden-receipt` type object exists, creating it (a
    checked-in TOML type blob) if absent.
2.  Read `captures.log` (or accept explicit `receipt-id` / `roots` / `store-id`
    for scripted use); for each single-root entry derive the foreign key from
    `roots[0]`.
3.  `HasBlob`-check the receipt against the repo's read blob store; warn and skip
    if unreachable (see Blob Store Reachability).
4.  **Update** (object-id supplied): load and rewrite that zettel via the
    `repo_actions.UpdateObject` pattern
    (`go/internal/sierra/repo_actions/update_object.go`), replacing the
    `BlobReferences` entry with the new `receipt_id` and rewriting the join
    blob's `captured_at` and `receipt_id`, then commit. `sigMother` chains
    automatically; identical metadata short-circuits to a no-op.
5.  **Create** (no object-id): write a new zettel
    (`repo_actions.WriteNewZettels` / `sku.Proto`) with the type, join blob, and
    blob reference.

### Blob Store Reachability

The referenced receipt blob MUST be readable from the dodder repo's madder blob
store. `captures.log`'s `store_id` names the store the receipt landed in (empty
string for the default store). The importer SHOULD `HasBlob`-check the receipt
before committing a reference and SHOULD warn when it is unreachable. Capturing
into a store the dodder repo is not configured to read produces danging
references and is out of scope for automatic handling.

### Producer Contract

This integration promotes `captures.log` from a private audit trail to a
**consumed public interface**. Its NDJSON record shape ---
`ts`, `receipt_id`, `store_id`, `roots` --- is the contract dodder depends on.
Cutting-garden SHOULD document this shape as stable and version it if it
changes. (A short companion note in cutting-garden recording this is
RECOMMENDED but is out of scope for this document.)

### Ingest Surface

The ingest is exposed as a dodder subcommand, proposed as

    dodder cutting-garden-import [captures.log]

reading the log (or accepting an explicit `receipt-id` / `roots` / `store-id`
for scripted use) and applying the flow above through the `repo_actions` layer.
Implementation MUST observe dodder's pool rules: never dereference a
`sku.Transacted`; use `ResetWith` / `CloneTransacted`. A new subcommand MUST be
registered in the `complete_subcmd` bats test.

### Resolved Decisions

1.  **Granularity: one versioned zettel per object.** Each captured object is a
    single zettel; re-capture chains a new version via `sigMother`. The
    "each receipt is its own immutable object" alternative was rejected because
    it discards the "edit and commit" history model that motivates the
    integration.
2.  **Direct blob vs. join wrapper: join wrapper.** The foreign key is absent
    from the receipt blob and must be stored explicitly, so the zettel blob is
    the TOML join document. The degenerate "receipt blob *is* the zettel blob"
    variant would require carrying the foreign key entirely in a tag or
    description and is not adopted.
3.  **Resolution: deferred.** v1 uses explicit create/update semantics
    (object-id presence); automatic foreign-key resolution is its own FDR (see
    Object Identity and Future Work).
4.  **Multi-root receipts: single-root only in v1.** A multi-root receipt is
    skipped with a warning in v1. The eventual direction is one zettel per root
    (each referencing the shared receipt); that fan-out is captured in the
    resolution FDR alongside the field-index design.
5.  **Type bootstrapping: auto-create if absent.** The importer checks in the
    `!cutting_garden-receipt` type object on first run when it is missing.

### Open Questions

None blocking v1. The remaining design work --- the type-defined field index for
automatic resolution, and the multi-root one-zettel-per-root fan-out --- is
consolidated into FDR 0017 (`docs/features/0017-type-defined-field-index.md`).

## More Information

- FDR 0017: Type-Defined Field Index --- the deferred automatic-resolution and
  multi-root design this document points to.

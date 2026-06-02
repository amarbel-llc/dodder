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

- `BlobReferences.Add(receiptMarklId, typeLock)` --- the type lock carries the
  receipt's own type tag, `cutting_garden-capture_receipt-fs-v1`. This is the
  "the receipt already has a type, use it directly" property: dodder records the
  receipt's native type rather than reinterpreting its bytes.
- `BlobReferences.SetAlias(receiptMarklId, "receipt")` --- a stable alias.

`BlobReferences` is the correct mechanism rather than `ContainedObjects`: the
latter references *in-store* dodder objects by `SeqId`, whereas a receipt is an
*external* content-addressed blob keyed by markl ID.

This design adds no field to `objects.metadata` or `blobReferenceEntry`, and so
requires no binary-codec change (see issue #38). It does, however, depend on the
`blobReferenceEntry` triple --- key, type lock, and alias --- round-tripping
through `india/stream_index` encode/decode. An implementer MUST verify this
before relying on the alias; if the alias is not yet serialized, either omit it
or extend the codec per the `india/stream_index` 4-file checklist.

### Foreign-Key Resolution

To make a re-capture land on the existing zettel, the importer MUST resolve
foreign key to object identity. The foreign key lives in the blob (the source of
truth), but blobs are not indexed for content lookup, and foreign keys (URLs,
absolute paths) are not legal tag strings.

The RECOMMENDED resolution is a derived, tag-safe lookup tag: a short digest of
the foreign-key string, e.g. `cg-source-<digest>`, added to the zettel.
Resolution is then a tag-plus-type query against the existing query system
(which supports tag and genre filters). The tag is a lookup accelerator only;
the blob's `foreign_key` remains authoritative, and the digest MUST be computed
from the exact normalized foreign-key string the producer recorded.

Alternatives, retained for the record:

1.  **Linear scan.** Enumerate `!cutting_garden-receipt` zettels and match
    `foreign_key`. Simplest; O(n) per import. Acceptable at small scale.
2.  **Dedicated index or genre.** A foreign-key-keyed index, or a custom genre
    whose identifier deterministically encodes the foreign key. Most direct
    lookup, but the heaviest change.

### Versioning and Deduplication

Versioning is dodder-native and requires no special handling. Committing a
zettel whose identifier already exists causes `store.Commit`
(`go/internal/oscar/store/mutating.go`) to fetch the prior version
(`fetchMotherIfNecessary`) and chain `sigMother` to it, extending the history
DAG. If the new metadata is identical to the current version, the commit
short-circuits and no new version is written --- so re-importing an unchanged,
deterministic receipt is a no-op.

The import flow is therefore:

1.  Read `captures.log`; for each entry derive the foreign key from `roots`.
2.  Resolve the foreign key to a zettel.
3.  **Found:** load it (`store.ReadOneObjectId`, via the
    `repo_actions.UpdateObject` pattern in
    `go/internal/sierra/repo_actions/update_object.go`), replace the
    `BlobReferences` entry with the new `receipt_id`, rewrite the join blob's
    `captured_at` and `receipt_id`, and commit. `sigMother` chains
    automatically.
4.  **Not found:** create a new zettel
    (`repo_actions.WriteNewZettels` / `sku.Proto`) with the type, join blob,
    blob reference, and lookup tag.

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

### Open Questions

1.  **Direct blob vs. join wrapper.** The join wrapper is specified because the
    foreign key is absent from the receipt blob and must be stored explicitly. A
    degenerate "receipt blob *is* the zettel blob" variant is possible only if
    the foreign key is carried entirely in a tag or description; it is not
    recommended.
2.  **Resolution strategy.** Derived tag vs. linear scan vs. dedicated index ---
    see Foreign-Key Resolution. The derived tag is recommended pending real
    scale data.
3.  **Multi-root receipts.** A capture group with multiple roots produces one
    receipt for several foreign keys. Whether such a receipt maps to one zettel
    per root (each referencing the shared receipt) or a single composite zettel
    is left open; the single-root case is the primary target.
4.  **Per-receipt vs. per-object granularity.** This document binds one zettel
    per object with the receipt as its versioned blob reference. An alternative
    that stores each receipt as its own immutable object and links them is
    possible but discards the "edit and commit" history model that motivates the
    integration.

---
date: 2026-05-03
promotion-criteria: dodder ingests a cutting-garden receipt blob into
  zettels via a single specified ingestion path that is plugin-agnostic;
  re-ingesting an unchanged source receipt is a no-op (same receipt
  markl-id → no new zettel commits); a per-entry type-mapping config
  lives in the workspace config and is read by the ingestion path; one
  BATS test ingests an FS-receipt and one ingests a non-FS-receipt
  (web-capture or caldav) through the same code path
status: proposed
---

# Capture-Protocol Ingestion

## Problem Statement

[FDR-0013](0013-cutting-garden-haustoria.md) reframes haustoria as
plugins on top of madder's cutting-garden protocol. Plugins emit
cutting-garden receipt blobs --- hyphence-wrapped NDJSON manifests of
type `cutting_garden-capture_receipt-*-vN` whose entries reference
content-addressable payload blobs. That FDR specifies *what the
substrate looks like*. It does not specify how dodder turns a receipt
into zettels.

The decisions left open:

- **Receipt-to-zettel mapping.** A receipt has a body of N entries.
  Does dodder commit one zettel per entry, one zettel per receipt with
  N referenced sub-objects, or both depending on the receipt subtype?
- **Identity.** Each entry has a plugin-side identity (a CalDAV UID, a
  nebulous spec-artifact markl-id, a filesystem path-within-root).
  How does dodder bind that to a zettel id without reinventing
  FDR-0007's per-object external GUID table?
- **Idempotence.** Receipts are content-addressed; re-ingesting an
  unchanged source must produce no new commits. The mechanism for
  detecting "I have already ingested this receipt" needs a home.
- **Type schemas.** Each plugin's entry type
  (`!task`, `!web-archive`, `!fs-entry`, ...) needs a declaration
  somewhere. Hard-coding a per-plugin Go switch defeats the FDR-0013
  goal of plugin independence.
- **Sub-object decomposition.** FDR-0007's CalDAV design decomposed
  VTODOs into `!task` + `!alarm` + blob references. The decomposition
  knowledge has to land somewhere when the source is a receipt rather
  than an in-tree haustoria call.

This FDR specifies a single plugin-agnostic ingestion path that
answers all five.

## Design

### The Receipt Is a Zettel

The receipt blob itself is committed as a zettel of type
`cutting_garden-capture_receipt-<plugin>-vN`. The receipt's markl-id
is its blob digest. The zettel's identity is allocated from the
workspace's normal id pool.

This is the workspace's anchor. Every subsequent ingestion in this
workspace records a new receipt zettel; the chain of receipt zettels
is the workspace's history relative to the external source. Diffing
two receipt zettels is equivalent to diffing the external source
between two capture runs.

  ----------------------------------------------------------------------
  Concept              Representation in dodder
  -------------------- -------------------------------------------------
  Receipt blob         A zettel typed
                       `cutting_garden-capture_receipt-<plugin>-vN`,
                       blob = the receipt bytes, description = the
                       `- store/<id> < <markl-id>` hint plus a
                       human-readable label.

  Receipt entry        A zettel typed by the plugin's per-entry type
                       (e.g. `!task`, `!web-archive`), referenced from
                       the receipt zettel via a referenced-object link
                       (FDR-0001), with the entry's payload blob
                       referenced via a typed blob reference.

  Capture run          The act of committing a new receipt zettel and
                       reconciling the entry zettels (see Idempotence).

  Source binding       The receipt zettel's blob digest is the
                       binding. There is no separate sync state.
  ----------------------------------------------------------------------

The "receipt is a zettel" rule means cutting-garden ingestion uses no
out-of-band tables, no `.dodder/sync-state/manifest.json` parallel to
the workspace's object store. Everything that needs to be remembered
across runs is itself an object in the graph.

### Per-Entry Type Mapping

The workspace config declares which dodder type each receipt subtype's
entries map to:

``` toml
[ingest]
plugin = "cutting-garden-caldav"
plugin-config = "@blake2b256-..."

[ingest.entry-type-map]
# receipt-subtype field → dodder type
# For cutting_garden-capture_receipt-caldav-v1, the entry's "component"
# field discriminates VTODO vs VEVENT.
"VTODO" = "!task"
"VEVENT" = "!event"

[ingest.entry-type-map.fields."!task"]
# JSON-pointer-into-entry → dodder field path.
# Identical to FDR-0007 mappings; same field declarations apply.
summary  = "/summary"
status   = "/status"
priority = "/priority"
due      = "/due"
body     = "/description"
```

The mapping lives in the workspace config (V2 from FDR-0005, extended
here). Rationale matches FDR-0007's: the same `!task` type ingests
from CalDAV in one workspace and from a tasks-on-disk receipt in
another. The type defines what the object *is*; the workspace
defines *how this plugin's entries are interpreted*.

For receipts whose body schema does not need per-entry discrimination
(the FS receipt is the canonical example: every entry is "a file",
"a directory", or "a symlink" --- the plugin's domain has no further
type ramification), the entry-type-map's key is the entry's `type`
field literal:

``` toml
[ingest.entry-type-map]
"file"    = "!fs-file"
"dir"     = "!fs-dir"
"symlink" = "!fs-symlink"
```

Entries whose discriminator is not in the map are skipped with a
notice. Entries whose discriminator is `other` (capture-receipt(7)'s
catch-all for devices/fifos/sockets) are always skipped, matching
the substrate's restore-side rule.

### Per-Entry Identity

Each entry in a receipt body has a *plugin-side identity*. The
identity is what makes "the same task across two capture runs" the
same task. The discriminator is plugin-specific:

  ----------------------------------------------------------------------
  Receipt subtype                          Identity field(s)
  ---------------------------------------- -----------------------------
  `cutting_garden-capture_receipt-fs-v1`   `(root, path)` --- the
                                           filesystem location.

  `cutting_garden-capture_receipt-caldav-v1`
                                           `uid` --- the iCalendar
                                           UID property.

  `cutting_garden-capture_receipt-web-v1`  the spec artifact's
                                           markl-id --- nebulous's
                                           canonical capture identity
                                           per
                                           [nebulous RFC 0001][nebulous-rfc-0001].

  `cutting_garden-capture_receipt-browser-v1`
                                           `(profile, item-type, item-id)`
                                           --- chrest's bookmark/tab
                                           addressing.
  ----------------------------------------------------------------------

The plugin's receipt subtype documents which fields constitute the
identity tuple. Dodder constructs a *per-entry binding key* by
canonicalizing the identity tuple (JCS, per
[RFC 8785][rfc-8785]) and hashing the result. The binding key lives
on the entry's zettel as a `cutting-garden-binding` metadata field
on the type lock (i.e., a referenced-object lock with a
`?binding=<key>` qualifier --- exact syntax is open, see Limitations).

Lookup is: given a receipt entry's identity tuple, hash it, look up
the zettel in the workspace whose `cutting-garden-binding` matches.
Found → update. Not found → create.

The binding key is workspace-local. The same external task in two
workspaces gets two zettel ids. This is intentional and matches
FDR-0007's reasoning: bindings are between a workspace and a source,
not between an object and an abstract external identity.

### Idempotence

Receipts are content-addressed. The receipt's markl-id is the
deterministic digest of its byte representation, which under
[madder RFC 0003][madder-rfc-0003] is sorted-by-`(root, path)` and
free of timestamps/hostnames/owners. Identical inputs to identical
plugin versions produce identical receipts.

The ingestion path's first action on every run:

1.  Spawn the plugin, capture the new receipt blob, compute its
    markl-id.
2.  Look up the workspace's last receipt zettel for this plugin.
3.  If the new receipt's markl-id equals the last receipt's markl-id,
    log "no changes" and exit. No new commits, no diff, no entry
    iteration.

When the markl-ids differ, the path computes a per-entry diff between
the new receipt's body and the previous receipt's body (both are
NDJSON, both are sorted, so the diff is a streaming merge-join keyed
on identity). The diff classifies entries as `created`, `updated`,
`deleted`, or `unchanged`, and emits a plan in the FDR-0006 sense.

The plan goes through FDR-0006's two-stage commit. The receipt zettel
itself is committed in the same plan as the entry zettels it
discovers, so partial-commit on crash leaves either zero or all
zettels from this run in the workspace.

### Sub-Object Decomposition

When a receipt entry has its own internal references (a CalDAV VTODO
with sub-VALARMs, a web-capture entry with envelope and spec
artifacts), the plugin's receipt subtype declares which of those are
*sub-entries* (recursively ingested as their own zettels) and which
are *blob references* (referenced by markl-id from the parent
zettel without further decomposition).

The declaration lives in the receipt subtype's documentation, not in
dodder. Dodder's ingestion path reads the entry, applies the entry-type
map, walks any declared sub-entries (recursively), and creates the
referenced-object/blob-reference graph per FDR-0001.

Decomposition depth is plugin-defined. A plugin author who decides
that VALARMs are sub-entries (independent objects) versus inline
JSON-in-the-VTODO-entry's-body controls that decision once for all
consumers. Dodder does not get a per-workspace knob to override the
plugin's decision; that lever proved to be a sharp edge in FDR-0007
(workspaces with different decomposition depths produced incompatible
graphs in the parent repo).

### Push-Back Ingestion

For plugins with a write-back story (CalDAV, the obvious case), the
ingestion path is read-only by design. Mutations to entry zettels in
dodder do not propagate to the source. Push-back is the inverse
operation and depends on the cutting-garden update protocol
([FDR-0013 § Bidirectional Plugins](0013-cutting-garden-haustoria.md))
or on per-plugin write APIs.

When that protocol matures, this FDR will gain a § Push-Back section.
Until then, write-back haustoria from FDR-0007 (the shipping
`haustoria_caldav`) keep their existing path; this ingestion FDR is
the read side that grows alongside it.

### Type Family Recognition

The `cutting_garden-capture_receipt-*-vN` type family must be
recognized by dodder's type system as a *family*, not as N unrelated
types. The marker is the type-id prefix `cutting_garden-capture_receipt-`.

The recognizer uses the prefix to dispatch to the cutting-garden
ingestion path; it does not validate the receipt's body schema (the
plugin's per-subtype documentation covers that). Validation that
*does* live in dodder's recognizer:

- The blob parses as hyphence with the declared type tag.
- The body is well-formed NDJSON.
- The metadata block, if present, contains a syntactically-valid
  store-hint line per [madder RFC 0003 § Receipt Metadata][madder-rfc-0003].

Per-subtype body validation (e.g., "every CalDAV entry has a `uid`
field") is the plugin's responsibility; dodder enforces only the
substrate-level invariants.

The prefix-recognition mechanism is a small extension to FDR-0010's
type system. Its details are an open question (see Limitations); the
contract here is that the type family is recognized as such, not how.

## Examples

Ingesting a fresh receipt (first run):

    $ dodder sync
    plugin: cutting-garden-caldav
    captured: receipt blake2b256-9ft3m74l5t...
    no prior receipt for this plugin in workspace
    plan: 47 created (47 !task), 0 updated, 0 deleted
    committing... done
    ingested 47 zettels under receipt blake2b256-9ft3m74l5t...

Ingesting a re-capture of an unchanged source:

    $ dodder sync
    plugin: cutting-garden-caldav
    captured: receipt blake2b256-9ft3m74l5t...
    receipt unchanged since last sync (blake2b256-9ft3m74l5t...)
    no changes

Ingesting a re-capture with deltas:

    $ dodder sync
    plugin: cutting-garden-caldav
    captured: receipt blake2b256-3wp380jqj2z...
    diff against blake2b256-9ft3m74l5t...:
      4 entries changed by uid
      1 entry added
      0 entries removed
    plan: 1 created, 4 updated, 0 deleted
    committing... done
    ingested 5 zettel changes under receipt blake2b256-3wp380jqj2z...

Ingesting a multi-entry-type receipt (web-capture):

    $ dodder sync
    plugin: cutting-garden-web (nebulous)
    captured: receipt blake2b256-pwjrvfg3wp...
    entry-type-map applied:
      web-archive (1) → !web-archive
    sub-entries: spec artifacts as blob refs, envelopes as blob refs,
                 payloads as blob refs (depth=0; nebulous receipts do
                 not nest)
    plan: 1 created (1 !web-archive)
    committing... done

## Limitations

- **Binding key syntax.** The exact way an entry zettel records its
  `cutting-garden-binding` is open. Three options:
  (a) a metadata field on the type lock; (b) a tag of the form
  `_cutting-garden-binding/<hash>` (excluded from user-visible tag
  queries); (c) a sidecar object referenced from the entry zettel.
  Option (a) is the cleanest model but requires extending the
  metadata codec ([dodder issue #38](https://github.com/amarbel-llc/dodder/issues/38)).
- **Type-family recognizer.** The mechanism for declaring "type-id
  prefix `cutting_garden-capture_receipt-` is a recognized family"
  could be hard-coded in the recognizer, declared via a meta-type
  per FDR-0000, or declared via a registry blob. Picking the right
  mechanism interacts with FDR-0000's still-evolving meta-type
  story.
- **Plugin discovery.** This FDR assumes the workspace config names
  the plugin binary explicitly (or via a convention like
  `cutting-garden-<plugin>` on PATH). The substrate-side decision
  lives in cutting-garden; dodder follows.
- **Cross-workspace deduplication.** Two workspaces ingesting the
  same source from the same plugin allocate independent zettel ids
  for the same external entries. There is no cross-workspace
  binding-key index. Push-to-parent therefore creates duplicate
  zettels at the parent level if both workspaces push. The right
  fix probably lives at the parent level (a parent-side
  binding-key index) but is out of scope for this FDR.
- **Receipt history retention.** Every capture run commits a new
  receipt zettel. A long-running workspace accumulates a chain of
  receipts, most of which are obsoleted by their successors. The
  garbage-collection / retention story for receipt zettels is
  unspecified.
- **Non-deterministic plugins.** A plugin whose output is not
  deterministic (e.g., a screenshot capturer whose output varies
  per-run) defeats the markl-id-based idempotence check. The
  detected diff will be 100% updated entries every run. This is a
  plugin bug, not an ingestion bug, but the workflow consequences
  are visible at the ingestion layer (constant churn). A future
  FDR may add a "treat re-captures as no-op when payload-only
  changes match a normalized signature" mode; the current design
  treats every byte-different receipt as a real diff.
- **Push-back is unspecified.** Read-only ingestion is the entire
  scope of this FDR. Write-back follows the cutting-garden update
  protocol when it lands.

## More Information

- [FDR-0013: Cutting-Garden as the Haustoria Protocol](0013-cutting-garden-haustoria.md)
  --- the substrate this FDR consumes.
- [FDR-0007: Pluggable Checkout Stores](0007-checkout-bridges.md) ---
  prior in-tree design; binding-key concept derives from its
  external-GUID binding.
- [FDR-0001: Object Locks](0001-object-locks.md) --- referenced
  objects and typed blob references used to model receipt entries
  and their payloads.
- [FDR-0005: Workspace-as-Repo](0005-workspace-as-repo.md) ---
  workspace config V2, extended by `[ingest]` and
  `[ingest.entry-type-map]` here.
- [FDR-0006: Two-Stage Commit](0006-two-stage-commit.md) --- batch
  commit used for receipt + entry zettel commits.
- [FDR-0010: Core Types](0010-core-types.md) --- type-blob field
  declarations the entry-type-map references.
- [madder RFC 0003: Capture / Restore Operational Rules][madder-rfc-0003]
  --- substrate spec.
- [capture-receipt(7)][capture-receipt-7] --- receipt body schema.
- [nebulous RFC 0001: Web Capture Archive Protocol][nebulous-rfc-0001]
  --- web-capture plugin's source design.

[madder-rfc-0003]: https://github.com/amarbel-llc/madder/blob/master/docs/rfcs/0003-capture-restore-rules.md
[capture-receipt-7]: https://github.com/amarbel-llc/madder/blob/master/docs/man.7/capture-receipt.md
[nebulous-rfc-0001]: https://github.com/amarbel-llc/nebulous/blob/master/docs/rfcs/0001-web-capture-archive-protocol.md
[rfc-8785]: https://www.rfc-editor.org/rfc/rfc8785

---
date: 2026-05-03
promotion-criteria: dodder consumes a cutting-garden plugin (any of the
  reference set --- CalDAV, orgmode, web-capture, browser bookmarks) end
  to end via the cutting-garden protocol, with no dodder-internal
  Haustoria implementation in the call path; the `cutting_garden-` blob
  type family is recognized by dodder's type system and produces
  CheckedOut state through the unified query path; one round-trip BATS
  integration test exercises the dodder ←→ cutting-garden plugin
  boundary
status: proposed
supersedes: 0007-checkout-bridges.md
---

# Cutting-Garden as the Haustoria Protocol

## Problem Statement

[FDR-0007](0007-checkout-bridges.md) defined a `Haustoria` interface
inside dodder's Go tree and proposed shipping per-store implementations
(`haustoria_caldav`, `haustoria_orgmode`, future `haustoria_chrest`,
`haustoria_nebulous`) compiled into the dodder binary. The `!task`
CalDAV haustorium under that design is in master and serving real
workloads; the design works.

What it does not do is travel. The `Haustoria` interface lives in
dodder, so a CalDAV ↔ object-graph translator is dodder-only. It cannot
be reused by `cutting-garden capture/restore` (which already speaks a
filesystem-tree dialect of the same problem), by `nebulous` (which has
its own three-tier capture protocol from
[nebulous RFC 0001][nebulous-rfc-0001]), or by any future tool that
wants to read a CalDAV calendar without depending on dodder. Each of
those tools either reinvents the format-translation half or links
dodder.

The substrate problem is the same in every case: take an external
system's view of some content (a filesystem subtree, a CalDAV
collection, a web page, an org-mode file, a browser bookmark tree),
produce a content-addressed manifest plus per-entry blobs, and let a
consumer materialize them on the other side. Madder already solved
this for filesystem trees with `cutting-garden` and the
`cutting_garden-capture_receipt-fs-v1` blob type
([madder RFC 0003][madder-rfc-0003], [capture-receipt(7)][capture-receipt-7]).
The receipt is a hyphence-wrapped NDJSON manifest that names every
entry by content-addressable blob id; the producer/consumer rules
(root scoping, store-hint resolution, path sanitization, per-type
materialization) are normative, language-agnostic, and have nothing
dodder-specific in them.

The opportunity is to grow cutting-garden into the canonical haustoria
protocol --- one substrate, many plugins, multiple consumers --- and
have dodder become *one consumer among several* rather than the owner
of the interface.

## Design

### Roles

  -----------------------------------------------------------------------
  Role         Lives in     Responsibility
  ------------ ------------ ---------------------------------------------
  Substrate    `madder`     Defines the cutting-garden producer/consumer
                            contract: receipt blob types
                            (`cutting_garden-capture_receipt-*-vN`),
                            store-hint metadata, path sanitization, and
                            per-type materialization rules. Owns the
                            reference producer (`cutting-garden capture`)
                            and consumer (`cutting-garden restore`).

  Plugin       per-tool     Implements the substrate's producer/consumer
                            roles for a non-filesystem source. Each
                            plugin defines its own receipt subtype
                            (`cutting_garden-capture_receipt-caldav-vN`,
                            `cutting_garden-capture_receipt-web-vN`,
                            etc.) layered on the cutting-garden contract.
                            Plugins ship as standalone binaries usable
                            without dodder.

  Consumer     `dodder`,    Reads receipts and materializes their
               other tools  contents. Dodder's consumer path is
                            specified in
                            [FDR-0014](0014-capture-protocol-ingestion.md);
                            it ingests receipt blobs as zettels rather
                            than restoring them to disk.
  -----------------------------------------------------------------------

The substrate spec is normative for all three roles. Dodder does not
get to extend the contract; it implements the consumer rules verbatim.
Plugin authors do not get to invent new metadata block conventions; they
allocate a new receipt type within the existing schema family and
follow [madder RFC 0003][madder-rfc-0003]'s rules.

### Plugins, Not Implementations

The reference plugin set, in maturity order:

1.  **`cutting-garden` itself** --- the filesystem-tree case
    ([madder RFC 0003][madder-rfc-0003]). Already shipping.
2.  **`nebulous` web capture** --- three-tier orchestrator/capturer/writer
    pipeline producing spec + envelope + payload artifacts per URL
    ([nebulous RFC 0001][nebulous-rfc-0001]). Reframed as a
    cutting-garden plugin: each archive run emits a receipt of type
    `cutting_garden-capture_receipt-web-v1` referencing the existing
    artifacts as blob ids, with the spec artifact as the canonical
    entry identity. The 3-tier writer protocol becomes the plugin's
    internal mechanism for *populating* the receipt; the receipt itself
    is the cutting-garden contract.
3.  **CalDAV** --- VTODO/VEVENT collections as receipts of type
    `cutting_garden-capture_receipt-caldav-v1`. The translation logic
    currently in `haustoria_caldav` and the iCalendar
    parser/serializer in `internal/hotel/caldav/` move into a
    `cutting-garden-caldav` plugin binary outside dodder.
4.  **`chrest` browser state** --- bookmark trees, tab snapshots, and
    `/items` mutations
    ([chrest content-tree](https://github.com/amarbel-llc/chrest)) as
    receipts of type `cutting_garden-capture_receipt-browser-v1`.
5.  **orgmode** --- file-trees of `.org` files as either
    `cutting_garden-capture_receipt-fs-v1` (using the existing FS
    receipt type) or a richer `cutting_garden-capture_receipt-org-v1`
    when sub-tree decomposition is wanted.
6.  **maneater corpora** --- semantic-search index manifests pointing
    at madder-stored embedding blobs. Listed for completeness; not
    in the initial ingestion set.

Each plugin is a separate repo, a separate binary, and a separate
release cadence. Dodder takes a runtime dependency on the plugin
binaries the user has installed, not a build-time dependency on their
implementations.

### What Dodder Owns

Dodder's responsibilities under this reframing collapse to:

1.  **Recognize** the `cutting_garden-capture_receipt-*-vN` type family
    as a first-class type lineage in the type system, with each
    subtype defining the field schema for its receipt body.
2.  **Ingest** receipt blobs as zettels (the consumer-side translation
    of "captured external state" into "object in the dodder graph").
    The full ingestion design lives in [FDR-0014](0014-capture-protocol-ingestion.md).
3.  **Track** the binding between a workspace and its source receipt.
    This replaces the per-object external-GUID binding from FDR-0007's
    sync state. The receipt's markl-id *is* the binding.
4.  **Surface** captured state through the unified query path. A
    receipt-derived zettel appears as `CheckedOut` (with the receipt
    blob id as its source coordinate) just like an FS-checked-out
    zettel does today. No command-level branching.

Dodder does *not* own the format translation, the transport, the
authentication, the conflict-resolution policy, or the plugin
discovery story. Those belong to the plugin and the substrate.

### What Cutting-Garden Owns

The cutting-garden protocol owns:

- The receipt type family (`cutting_garden-capture_receipt-*-vN`) and
  the rule that new fields require a new type version, not in-place
  schema evolution
  ([capture-receipt(7) § Versioning][capture-receipt-7]).
- The hyphence metadata block conventions, including `- store/<id> < <markl-id>`
  store hints ([madder RFC 0003 § Receipt Metadata][madder-rfc-0003]).
- The path-sanitization rules and per-type materialization contract
  for filesystem-shaped receipts. Non-FS plugins specify their own
  per-type materialization in their subtype's documentation but inherit
  the metadata-block conventions and the determinism requirements.
- The producer/consumer split, including the rule that consumers must
  not trust producer conformance and must re-validate.

Dodder's FDR tree references these specs; it does not duplicate them.

### Bidirectional Plugins

FDR-0007 specified compile/decompile (push-back) for CalDAV. Read-only
ingestion is the easy half; round-trip is the substantive design
work. Cutting-garden's RFC 0003 is currently capture-only (the
restore-side rules are the consumer half of one direction). For
plugins that need round-trip semantics --- CalDAV is the canonical
example, since users expect to mark tasks complete in tasks.org and
have dodder pick that up --- one of two extensions is needed:

- **`cutting-garden update`** as a sibling of `restore`: take a
  modified receipt, diff against a base receipt, and push the deltas
  back to the source. The plugin implements the push side; the substrate
  defines the diff/merge rules. This is a cutting-garden RFC, not a
  dodder FDR.
- **Per-plugin push protocols** outside cutting-garden's scope. A
  CalDAV plugin could ship its own write API while still using
  cutting-garden's read receipts. Less elegant but unblocks plugins
  whose write semantics don't fit a uniform diff model.

Dodder does not need to pick between these to consume the read side.
The push-back design lives in cutting-garden's RFC tree when it
matures; this FDR is forward-compatible with either choice.

### Migration from FDR-0007's Implementation

The shipping `haustoria_caldav` becomes a transitional adapter:

1.  **Phase 1 (now → cutting-garden-caldav v0):** existing
    `haustoria_caldav` keeps working unchanged. No regression.
2.  **Phase 2:** extract the iCalendar translation and CalDAV transport
    into a `cutting-garden-caldav` plugin binary outside dodder. The
    in-tree `haustoria_caldav` becomes a thin shim that spawns the
    plugin binary and ingests its receipt output via the FDR-0014
    pipeline. Sync state moves from per-object GUID bindings to
    per-receipt markl-id bindings.
3.  **Phase 3:** dodder removes the `haustoria_caldav` shim once all
    workspaces have migrated to consuming the plugin's receipts
    directly through the generic cutting-garden ingestion path. The
    in-tree `Haustoria` interface either retires or remains as a
    Go-level abstraction over the receipt ingestion pipeline (an
    implementation detail, not a public contract).

The shipped `!task` type, its typed fields (`status`, `priority`,
`due`), and the genesis flag from FDR-0007 PR #100 are unaffected ---
they describe what a task *is* in the dodder graph, which is
plugin-independent. The CalDAV-specific compile/decompile glue is
what migrates.

The orgmode haustorium (also referenced in
[FDR-0012](0012-native-messaging-hosts.md)) follows the same migration
path: extract into a `cutting-garden-orgmode` plugin, dodder ingests
its receipts.

### Relationship to FDR-0012 (Native Messaging Hosts)

FDR-0012 explores how a haustoria's *implementation* runs out of
process (WASM + WASI host pipe, or Hashicorp go-plugin). That question
is orthogonal to this FDR. Under the cutting-garden framing it
becomes: how does dodder spawn and communicate with a
*cutting-garden plugin* binary? The plugin is the out-of-process
implementation FDR-0012 was reaching for, with the cutting-garden
receipt as the wire format. FDR-0012's two candidate mechanisms
(WASM-codec-with-host-pipe vs go-plugin) collapse into one cleaner
question: subprocess spawn + stdin/stdout JSON, where the JSON is the
cutting-garden batch input/receipt-output that the substrate already
specifies. FDR-0012's exploration of in-tree codec WASM is preserved
for the cases where dodder wants format-translation logic to live in
content-addressable blobs (a separate concern from haustoria
plumbing).

## Examples

A workspace ingesting CalDAV via the cutting-garden plugin:

    $ dodder init-workspace -experimental-repo \
        -cutting-garden-plugin cutting-garden-caldav \
        -plugin-config @blake2b256-... \
        project-tasks '+task project-alpha'

    workspace-repo created at .dodder/
    cutting-garden plugin: cutting-garden-caldav (v0.1.0)
    spawned plugin, captured 47 entries
    receipt: blake2b256-9ft3m74l5t...
    ingested 47 zettels (12 with !alarm subobjects)

A subsequent sync:

    $ dodder sync
    plugin: cutting-garden-caldav
    capturing... receipt: blake2b256-3wp380jqj2z...
    diff against last receipt (blake2b256-9ft3m74l5t...):
      4 entries changed, 1 entry added, 0 removed
    ingesting deltas...
    sync complete: 4 updated, 1 created

A workspace whose haustorium is the FS-tree case (cutting-garden's
canonical use):

    $ dodder init-workspace -experimental-repo \
        -cutting-garden-plugin cutting-garden \
        -plugin-args 'capture ./vendor' \
        vendor-tree '+vendor'

    workspace-repo created at .dodder/
    cutting-garden plugin: cutting-garden (built-in fs)
    captured 1043 entries from ./vendor
    receipt: blake2b256-pwjrvfg3wp380...
    ingested 1043 zettels

## Limitations

- Cutting-garden's bidirectional/push semantics are not yet specified.
  Until they are, plugins requiring write-back (CalDAV is the
  immediate one) either keep an out-of-band push API or wait on a
  cutting-garden RFC for the update protocol. The dodder side does
  not block on this --- read-only ingestion is the foundational
  capability and unblocks every plugin's first usable mode.
- Plugin discovery is not specified here. The examples assume the
  user names a plugin binary explicitly. A naming convention
  (`cutting-garden-<plugin>` on PATH, mirroring `git-<subcommand>`)
  is the obvious sketch but is left to a follow-up FDR or a
  cutting-garden-side decision.
- Three-way merge (FDR-0007's hyphence-text merge) maps onto the
  receipt model as "diff two receipts, three-way against a base
  receipt." The mechanics differ from object-level merge because the
  receipt's NDJSON body is sorted and stable; per-entry merge is
  cleaner than text-level merge. The detailed design lives with
  whichever side (cutting-garden or dodder) eventually owns the
  push-back semantics.
- The `cutting_garden-capture_receipt-*-vN` type family must be
  recognized by dodder's type system as a *family* with shared
  ingestion rules, not as 47 unrelated types. The mechanism for
  declaring "this type is a cutting-garden receipt subtype" is an
  open question --- it could be a special prefix the type system
  recognizes, a meta-type per FDR-0000, or an explicit type-blob
  field. This is the first concrete dependency this FDR places on
  FDR-0000's meta-type story.
- Plugins that capture from authenticated remote systems (CalDAV,
  proprietary APIs) need credentials. Credential management lives
  outside cutting-garden's contract; plugins use whatever the host
  OS provides (keychain, environment, credential helpers). Dodder
  passes through opaque plugin-config blobs without inspecting them.
  This deliberately punts credential governance to plugins; if a
  cross-plugin pattern emerges, a follow-up FDR can address it.

## More Information

- [FDR-0007: Pluggable Checkout Stores](0007-checkout-bridges.md) ---
  the original in-tree `Haustoria` design. Superseded by this FDR but
  the implementation-status section documents shipping behavior.
- [FDR-0014: Capture-Protocol Ingestion](0014-capture-protocol-ingestion.md)
  --- the consumer-side rules for turning receipts into zettels.
- [FDR-0012: Native Messaging Hosts](0012-native-messaging-hosts.md)
  --- out-of-process haustoria mechanism, reframed by this FDR as
  "spawn the cutting-garden plugin binary."
- [FDR-0010: Core Types](0010-core-types.md) --- typed blob locks
  used by receipt subtypes to declare their schema.
- [madder RFC 0003: Capture / Restore Operational Rules][madder-rfc-0003]
  --- normative substrate spec.
- [capture-receipt(7)][capture-receipt-7] --- receipt body schema and
  determinism rules.
- [nebulous RFC 0001: Web Capture Archive Protocol][nebulous-rfc-0001]
  --- the web-capture plugin's source design, to be reframed as a
  cutting-garden plugin per this FDR.

[madder-rfc-0003]: https://github.com/amarbel-llc/madder/blob/master/docs/rfcs/0003-capture-restore-rules.md
[capture-receipt-7]: https://github.com/amarbel-llc/madder/blob/master/docs/man.7/capture-receipt.md
[nebulous-rfc-0001]: https://github.com/amarbel-llc/nebulous/blob/master/docs/rfcs/0001-web-capture-archive-protocol.md

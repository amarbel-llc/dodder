---
status: proposed
date: 2026-06-12
promotion-criteria: implementation merged with V16 migration passing the
  previous_versions conformance suite; no chain-verification or
  log-compaction lever adjustments needed for 2 weeks of daily use
---

# Config as a non-object

## Problem Statement

Config has been a full store object: its own genre, a singleton `konfig`
object id, signed commits, mother-signature history, and special-casing
in the read path that bypasses the stream index anyway. All that object
machinery buys nothing for a value that is repo-local, unsynced, and
mutated only by `edit-config` — while costing genre-surface complexity
in queries, abbr indexes, remote transfer, and import planning.

## Interface

Config stops being an object. A config state is a bare-TOML blob typed
`!toml-config-v2` (unchanged from today). History lives in a repo-local
file, `.dodder/config`, which is a literal inventory list: one
box-format entry appended per change. Append order is the history; the
last entry is the head, and bootstrap reads only this file.

Entries chain by digest, exactly like bona fide inventory lists but
with digests where signatures would be: each entry carries the `konfig`
label, the blob digest, tai, the type, a self-digest computed by the
object finalizer, and a mother field holding the previous entry's
self-digest. Signature fields are empty. The chain is not load-bearing
for reads — it provides provenance (clone) and integrity (verifiable by
an explicit fsck-style check).

Commands:

- `show-config` — print the latest config (bare TOML).
- `show-config <digest>` — print that historical state's blob.
- `show-config -history` — list entries oldest→newest as box lines.
- `edit-config` — unchanged editor UX; on save, writes the blob and
  appends a chained entry under the repo lock. No commit, no
  signature, no index.

Genre removal: `config`/`konfig` no longer parse as a genre or object
id anywhere in the query surface. The tokens produce a targeted error
(`config is no longer an object; use show-config / edit-config`).
Genre byte 5 remains reserved and decodable for old store versions
only.

Sync: config is repo-local; push/pull do not transfer it. Clone seeds
guidance: the source's latest config blob is copied and the new repo's
log starts with one entry whose mother names the source's head
self-digest — a dangling, shallow-style pointer reported as "history
continues in the source repo".

Migration: store version bump (V15 → V16) converts the old konfig
object history oldest→newest into chained log entries pointing at the
already-existing blobs. No blobs are written or rewritten; old
inventory lists are preserved untouched.

## Examples

Show the current config:

    $ dodder show-config
    [compiled TOML body]

List history, then inspect an older state:

    $ dodder show-config -history
    [konfig @blake2b256-aaa... <tai> !toml-config-v2 ...]
    [konfig @blake2b256-bbb... <tai> !toml-config-v2 ...]

    $ dodder show-config blake2b256-aaa...
    [that state's TOML body]

Old query surface redirects:

    $ dodder show :konfig
    error: config is no longer an object; use show-config / edit-config

## Limitations

- History is primarily append-order in a local file. The digest chain
  makes the sequence tamper-evident only when explicitly verified;
  normal reads trust the file.
- A cloned repo holds only the seeded head state; older states resolve
  only if their entries/blobs are obtained from the source.
- No back-migration tool: migration is additive (old konfig objects
  survive in old inventory lists), so down-migration is possible in
  principle but not built.
- **Two inventory-list-style logs now exist**: `FileInventoryListLog`
  (real inventory lists) and `.dodder/config` (config states). They
  share the append + MultiWriter mechanism and box coders but are
  separate files with separate hooks. There may be value in
  consolidating the log machinery into one shared component — flagged
  as follow-up, not in scope here.

## Tuning Levers

| Lever | Current | Rationale | Change signal |
|---|---|---|---|
| chain verification on read | off | local file is trusted; verification is an explicit check | any real-world corruption report |
| log compaction | none | config changes are rare; file stays tiny | a `.dodder/config` measured in megabytes |

## More Information

- Design doc: `docs/plans/2026-06-12-config-non-object-design.md`
- Inventory list log mechanism:
  `go/internal/india/inventory_list_store/blob_store_v1.go`
- Mother-slot reuse: markl ids are purpose-tagged, so the mother field
  holds a digest where real objects hold a signature.

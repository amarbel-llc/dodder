---
status: proposed
date: 2026-06-12
promotion-criteria: implementation merged with V16 migration passing the
  previous_versions conformance suite; no log-compaction lever
  adjustments needed for 2 weeks of daily use
---

# Config as a non-object

## Problem Statement

Config has been a full store object: its own genre, a singleton `konfig`
object id, commits through the index/remote-transfer path, and
special-casing in the read path that bypasses the stream index anyway.
All that object machinery buys nothing for a value that is repo-local,
unsynced, and mutated only by `edit-config` — while costing
genre-surface complexity in queries, abbr indexes, remote transfer, and
import planning.

## Interface

Config stops being an object. A config state is a bare-TOML blob typed
`!toml-config-v2` (unchanged from today). History lives in a repo-local
file at `FileConfigLog()` (layout name `config_log`) which is **literally
an inventory list**, reusing the existing `!inventory_list-v2` type and
signed coder verbatim — distinguished from the object inventory-list log
only by file name/location and the genre of its entries. One box-format
entry is appended per change. Append order is the history; the last
entry is the head, and bootstrap reads only this file.

Each entry is a bona fide signed inventory-list entry: the `konfig`
object id, the blob digest, tai, the type, the repo public key, the
object signature, and a mother signature = the previous entry's object
signature. This is the same signature chain bona fide inventory lists
already use; config is repo-local and signs with the repo key. The
chain is not load-bearing for reads — it provides provenance (clone)
and tamper-evidence via the standard inventory-list verify path.

Commands:

- `show-config` — print the latest config (bare TOML).
- `show-config <digest>` — print that historical state's blob.
- `show-config -history` — list entries oldest→newest as box lines.
- `edit-config` — unchanged editor UX; on save, writes the blob and
  appends a signed entry to the config log under the repo lock. No
  object commit into the store, no stream-index participation.

Genre removal: `config`/`konfig` no longer parse as a genre or object
id anywhere in the query surface. The tokens produce a targeted error
(`config is no longer an object; use show-config / edit-config`).
Genre byte 5 remains reserved and decodable for old store versions
only.

Sync: config is repo-local; push/pull do not transfer it. Clone seeds
guidance: the source's latest config blob and head entry are copied
into the new repo's config log. The copied entry verifies against the
source's embedded pubkey; its mother references an entry the clone
lacks — a dangling, shallow-style pointer reported as "history
continues in the source repo". Later local edits sign with the local
key, chaining from the copied entry.

Migration: store version bump (V15 → V16) re-emits the old konfig
object history oldest→newest as signed config-log entries pointing at
the already-existing blobs. No blobs are written or rewritten; old
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

- History is append-order in a local file; the signature chain makes
  the sequence tamper-evident via the standard inventory-list verify
  path, but normal reads trust the file.
- A cloned repo holds only the seeded head state; older states resolve
  only if their entries/blobs are obtained from the source.
- No back-migration tool: migration is additive (old konfig objects
  survive in old inventory lists), so down-migration is possible in
  principle but not built.
- **Two inventory-list logs now exist**, both `!inventory_list-v2`:
  `FileInventoryListLog` (object inventory lists) and `FileConfigLog`
  (config states). They now share the exact same type and coder,
  differing only by file. There is clear value in consolidating the
  append/read machinery into one shared component — flagged as
  follow-up, not in scope here.

## Tuning Levers

| Lever | Current | Rationale | Change signal |
|---|---|---|---|
| log compaction | none | config changes are rare; file stays tiny | a `config_log` measured in megabytes |

## More Information

- Design doc: `docs/plans/2026-06-12-config-non-object-design.md`
- Inventory list log mechanism:
  `go/internal/india/inventory_list_store/blob_store_v1.go`
- Config entries reuse the `!inventory_list-v2` signed coder; the only
  thing config-specific is the separate log file.

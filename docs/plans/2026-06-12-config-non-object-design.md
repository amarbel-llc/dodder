# Config as a Non-Object — Design

Date: 2026-06-12
Status: approved (design); implementation plan to follow

## Goal

Remove config's status as a store object: delete the Config genre from
the user-facing surface, stop committing config changes as signed
objects, and add a `show-config` command. Config history becomes a
repo-local, inventory-list-formatted log whose entries chain by digest
(mother digests, not signatures), git-style.

## Current state (what this replaces)

-   `genres.Config` (byte 5) is a full genre: parseable in queries
    (`show :konfig`, `+konfig`), present in the genre bitfield,
    `file_extensions`, abbr indexes, `import_plan`, `remote_transfer`.
-   The singleton object id `konfig` (`ids/konfig.go`) is committed
    through the normal object path: signatures, mother *signatures*,
    tai, stream-index special-casing (`store/reader.go:198` bypasses
    the index and reads from in-memory `store_config`).
-   The config blob is bare TOML typed `!toml-config-v2`
    (`repo_configs` V0/V1/V2 coders).
-   Mutation: `edit-config` (and the dormant-edit flow, which shares
    the konfig-edit helper) → `UpdateKonfig(blobDigest)` → object
    commit under the LockSmith filesystem mutex.

## Decisions

### Config blobs are unchanged

A config state remains a bare-TOML blob typed `!toml-config-v2` in the
repo's blob store. No new blob type, no hyphence wrapper at the blob
level, no mother line inside the blob. Migration never rewrites a
config blob.

### `.dodder/config` is an inventory-list-format log

The local file `.dodder/config` (exact name/placement to fit
`env_repo`'s layout, settled during implementation) is a literal
inventory list: hyphence doc with an inventory-list type header, body
appended one box-format entry per config change, using the same
append + MultiWriter mechanism as `FileInventoryListLog`
(`inventory_list_store/blob_store_v1.go`).

Each entry carries:

-   the literal `konfig` label in the object-id slot (cosmetic,
    file-local; nothing parses it as an object id),
-   `@<blob-digest>` of the config TOML blob,
-   tai,
-   the type (`!toml-config-v2`),
-   a self-digest computed by the existing `object_finalizer`
    (covering the entry's fields including the mother slot),
-   a mother field holding the **previous entry's self-digest**
    (the markl purpose-tagged mother slot holds a digest where real
    objects hold a signature),
-   empty signature fields.

This is a git-commit chain from existing parts: self-digest ≈ commit
hash, blob digest ≈ tree, mother ≈ parent.

Coder: the box format and append mechanism are reused as-is; the only
delta is a coder registration whose hooks finalize **without**
signature verification (digest-only assert) — the bona fide
`!inventory_list-v2` hooks demand non-null signatures
(`inventory_list_coders/main.go:52-79`).

### History = append order; the chain is extra sauce

The order of entries in the log IS the history; the last entry is the
latest config (the head). Bootstrap reads the local file only — no
blob-store read. `show-config -history` is a sequential file read
printed with the existing `BoxTransacted` formatter. The
mother/self-digest fields are not load-bearing for normal reads; they
exist for provenance (clone) and integrity (verifiable by an
fsck-style check when wanted).

### CLI surface

-   `show-config` — print the latest config blob (bare TOML).
-   `show-config <digest>` — fetch that blob and print it.
-   `show-config -history` — list entries oldest→newest as box lines.
-   `edit-config` — unchanged UX (editor on bare TOML); on save:
    write blob, append chained entry to the log under the existing
    LockSmith mutex. No object commit, no signature, no index.
    The shared konfig-edit helper keeps serving the dormant-edit flow.
-   New subcommand added to `complete.bats` `complete_subcmd`.

### Genre removal

-   `genres.Config` leaves the user surface: genre parsing no longer
    accepts `config`/`konfig`; removed from the sigil/genre query
    bitfield, `file_extensions`, abbr indexes, `import_plan`,
    `remote_transfer`, and the `store/reader.go` special case.
-   Byte value 5 stays reserved and decodable in legacy codecs only —
    old inventory lists containing konfig entries must still decode
    (no-removal rule for old store versions). Excluded from all-genre
    iteration; never emitted in new data.
-   Query tokens `konfig`/`config` produce a targeted error:
    `config is no longer an object; use show-config / edit-config`.
-   `!toml-config-v0/v1/v2` stay registered; v2 remains the emitted
    config blob type.

### Sync and clone

-   Config is repo-local. Push/pull no longer transfer config.
-   Clone seeds guidance: copy the source's latest config blob, write
    a one-entry log whose mother field names the source's head
    self-digest. The dangling mother is tolerated and reported as
    "history continues in the source repo" (shallow-clone style).
-   `init` writes the initial config blob and a root entry (no
    mother).

### Migration (V15 → V16)

Store version bump. Walk the old konfig object history oldest→newest
via the retained old codecs; append one chained entry per historical
state (blob digest unchanged, tai preserved from the old object,
self-digest fresh, mother = previous entry's self-digest). After
migration, Config-genre entries are never emitted into new inventory
lists or the stream index.

Open items to pin during implementation planning (not yet verified):

1.  Where this hooks into the existing store-version migration path.
2.  That migration preserves old inventory lists untouched (the
    design REQUIRES this — old konfig objects must survive so
    back-migration remains possible).
3.  How pulls from old-version remotes whose lists contain konfig
    entries behave (expected: decode and skip).

## Rollback

Dual-architecture inside one store is not feasible (a store is either
pre- or post-migration). Posture instead:

-   Migration is additive: old konfig objects remain in old inventory
    lists; only the new log file is created. A back-migration tool is
    possible but not built now.
-   Before migration: rollback is trivial (old binary, untouched
    store).
-   Legacy decode paths are kept indefinitely per existing
    convention.

## Testing

-   BATS: update `edit_config.bats`, `dormant_edit.bats`; new
    `show_config.bats` (latest, by-digest, `-history`, post-clone
    shallow case); `complete.bats` subcommand list; `show.bats`
    targeted-error assertions for konfig queries; `export.bats` /
    `push.bats` lose konfig entries for new stores; migration
    conformance via `previous_versions/main.bats` with the snapshot →
    bump VCurrent → regenerate-fixtures workflow.
-   Go: unit tests for the config log (append, head read, chain
    fields, sig-less coder hooks) plus genre-removal fallout.

## Tuning levers

-   **Chain verification on read**: off by default (trust the local
    file; verify only via explicit check). Signal to revisit: any
    real-world corruption report.
-   **Log compaction**: none (config changes are rare). Signal: a log
    measured in megabytes.

## Follow-up flagged for the FDR

With this change the repo carries two inventory-list-style log files:
`FileInventoryListLog` (real inventory lists) and `.dodder/config`
(config states). There may be value in consolidating the append/read
machinery — captured in the FDR for this feature.

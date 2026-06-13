# Config as a Non-Object — Design

Date: 2026-06-12
Status: approved (design); implementation plan to follow

## Goal

Remove config's status as a store object: delete the Config genre from
the user-facing surface, stop committing config changes through the
object/index/remote-transfer path, and add a `show-config` command.
Config history becomes a repo-local inventory list in its own file —
reusing the existing `!inventory_list-v2` type and signed coder
verbatim, distinguished only by file name/location.

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

### The config log is literally an inventory list

The local file at `FileConfigLog()` (layout name `config_log`, sibling
of `inventory_lists_log`) **is** an inventory list, reusing the
existing `!inventory_list-v2` type and coder verbatim. The only things
that distinguish it from the object inventory-list log are its file
name/location and the genre of the entries it carries (config states,
not zettels/types). Append uses the same MultiWriter mechanism as
`FileInventoryListLog` (`inventory_list_store/blob_store_v1.go`):
`FinalizeAndSign` with the repo private key, then write to both the
blob store and the log file.

Each entry is a bona fide signed inventory-list entry carrying:

-   the `konfig` object id (genre Config, internal-only),
-   `@<blob-digest>` of the config TOML blob,
-   tai,
-   the type (`!toml-config-v2`),
-   the repo public key, the object signature, and the **mother
    signature** = the previous entry's object signature.

This is the same signature chain bona fide inventory lists already
use: object-sig ≈ commit hash, blob digest ≈ tree, mother-sig ≈
parent. No new type, no new coder, no new markl purpose, and no
`box_format` change — the existing signed path works as-is because the
entries are signed.

Earlier drafts used unsigned entries with a digest-valued mother slot
("mother digests, not signatures"). That was abandoned: the box format
gates mother emission behind a non-null object signature
(`box_format/transacted.go:264`), so unsigned entries would lose the
mother on encode. Signing the entries — config is repo-local and the
repo key is on hand at write time — sidesteps the gate and makes the
log indistinguishable in format from a real inventory list.

### History = append order; the chain is extra sauce

The order of entries in the log IS the history; the last entry is the
latest config (the head). Bootstrap reads the local file only — no
blob-store read. `show-config -history` is a sequential file read
printed with the existing `BoxTransacted` formatter. The mother
signature is not load-bearing for normal reads; it provides provenance
(clone) and tamper-evidence via the standard inventory-list verify
path.

### CLI surface

-   `show-config` — print the latest config blob (bare TOML).
-   `show-config <digest>` — fetch that blob and print it.
-   `show-config -history` — list entries oldest→newest as box lines.
-   `edit-config` — unchanged UX (editor on bare TOML); on save:
    write blob, append a signed entry to the config log under the
    existing LockSmith mutex. No object commit into the store, no
    stream-index participation. The shared konfig-edit helper keeps
    serving the dormant-edit flow.
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
-   Clone seeds guidance: copy the source's latest config blob and the
    source's head entry verbatim into the new repo's config log. The
    copied entry carries the source's pubkey + signature and verifies
    against them; its mother references an entry the clone doesn't have
    — tolerated and reported as "history continues in the source repo"
    (shallow-clone style). The cloned repo's later edits are signed by
    the local key with mother = the copied entry's signature.
-   `init` writes the initial config blob and a root entry (signed, no
    mother).

### Migration (V15 → V16)

Store version bump. Walk the old konfig object history oldest→newest
via the retained old codecs; append one entry per historical state to
the config log (blob digest unchanged, tai preserved from the old
object). The old konfig objects were already signed, so migration
re-emits each as a signed config-log entry with mother = previous
entry's signature. After migration, Config-genre entries are never
emitted into new object inventory lists or the stream index.

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
-   Go: unit tests for the config log (append, head read, signed entry
    + mother-sig chain) plus genre-removal fallout.

## Tuning levers

-   **Log compaction**: none (config changes are rare). Signal: a log
    measured in megabytes.

## Follow-up flagged for the FDR

With this change the repo carries two inventory-list log files,
both `!inventory_list-v2`: `FileInventoryListLog` (object inventory
lists) and `FileConfigLog` (config states). Since they now share the
exact same type and coder and differ only by file, there is clear
value in consolidating the append/read machinery — captured in the FDR
for this feature.

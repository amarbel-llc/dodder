# TODO

- [ ] migrate to `bats_load_library bats-island` (replace inline set_xdg/setup_test_home/chflags_and_rm)

- [ ] Debug `just test-bats-update-fixtures` failure: bats succeeds when run directly from `zz-tests_bats/` but fails when invoked through the justfile recipe chain. Likely a working directory or environment variable propagation issue. The `cp` command can't find `.dodder` in the bats temp dir, suggesting the fixture generation test silently fails or the temp dir path extraction breaks.

## Temporary backwards-compat: `!` blob path fallback in triple-hyphen parser

- [ ] Remove the `strings.Contains(typeString, "/")` fallback in `text_parser2.readType` (`india/object_metadata_fmt_triple_hyphen/text_parser2.go`). Old checked-out zettel files used `! <path>` for blob references; new format uses `@ <path>`. Once all workspaces have been re-checked-out, this shim can be deleted.

## `der import` bugs (found during migration validation 2026-02-22)

- [ ] `der import` crashes with "all FD's are empty" when importing an object whose ID already exists in the store under a different pubkey. The error originates in `store_fs/merge.go:327` (`checkoutOneForMerge`) and cascades recursively through `Import.Run`. Workaround: exclude already-existing objects from the import file.

- [ ] `der import` silently skips objects that share a blob hash with another entry in the same batch. Only the first object per unique blob hash is imported; the rest are dropped with no error and exit 0. Workaround: import shared-blob objects individually (one entry per file) or run multiple passes.

- [ ] `der import` silently skips blobless type definitions (e.g. `[!opml 2097748458.73047 !toml-type-v1]` — no `@sha256-...` blob ref, no pubkey, no sig). These entries produce no output and no error. This causes downstream failures when importing objects that depend on those types ("failed to read current lock object").

## Probe index panic on truncated page entries

- [ ] `page_reader_probe.go:86` panics with `unexpected EOF` when a probe page file is shorter than the cursor's `ContentLength`. Should return an error instead of panicking so `fsck` can report it and continue. Reproduces with production repo at `/home/sasha/workspaces/dodder-index-test` running `dodder fsck` without `-skip-probes`.

## Synthetic tai disambiguation for `der import`

- [ ] FDR: Add sub-second tai disambiguation during import to eliminate objectId+tai collisions (10,731 in production repo). Intervene in `remote_transfer/main.go` before `importNewObject()` commit: group by objectId+tai, assign incrementing attosecond offsets for duplicates, re-sign affected objects. Caveat: changes object digests vs source repo, requires `OverwriteSignatures`.

## `go mod tidy` resolution errors

- [ ] `go mod tidy` fails resolving `code.linenisgreat.com/dodder/go/src/bravo/ohio` and `code.linenisgreat.com/dodder/go/src/bravo/ui` — imported transitively via `chrest/go/src/bravo/server`. The `src/` path prefix doesn't exist in the current dodder module. Likely a stale import path in chrest that needs updating.

## Archive store foreign digest support

- [ ] Implement `BlobForeignDigestAdder` for inventory archive stores. Idea: use symlinks in the embedded loose blob directory pointing to packed blob entries, so `HasBlob(foreignDigest)` resolves via the loose store fallback. Requires solving the read path (symlink target is a packfile, not a single blob file). See `docs/plans/2026-02-23-sync-cross-hash-design.md`.

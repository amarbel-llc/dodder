# TODO

## V14 stream index exploration

- [ ] explore using sigil as continuation operator in v14 fixed stream index
- [ ] explore using content offsets instead of signatures in v14 fixed stream
  index

## Performance validation

- [ ] run performance tests on packfiles
- [ ] run performance tests on v14 fixed stream index

## Encryption

- [ ] verify encryption support in packfiles
- [ ] add support for n / m sapir recovery for piv encryption

## Temporary backwards-compat: `!` blob path fallback in triple-hyphen parser

- [ ] Remove the `strings.Contains(typeString, "/")` fallback in `text_parser2.readType` (`india/object_metadata_fmt_triple_hyphen/text_parser2.go`). Old checked-out zettel files used `! <path>` for blob references; new format uses `@ <path>`. Once all workspaces have been re-checked-out, this shim can be deleted.

## `der import` bugs (found during migration validation 2026-02-22)

- [ ] `der import` crashes with "all FD's are empty" when importing an object whose ID already exists in the store under a different pubkey. The error originates in `store_fs/merge.go:327` (`checkoutOneForMerge`) and cascades recursively through `Import.Run`. Workaround: exclude already-existing objects from the import file.

- [ ] `der import` silently skips objects that share a blob hash with another entry in the same batch. Only the first object per unique blob hash is imported; the rest are dropped with no error and exit 0. Workaround: import shared-blob objects individually (one entry per file) or run multiple passes.

- [ ] `der import` silently skips blobless type definitions (e.g. `[!opml 2097748458.73047 !toml-type-v1]` — no `@sha256-...` blob ref, no pubkey, no sig). These entries produce no output and no error. This causes downstream failures when importing objects that depend on those types ("failed to read current lock object").

## Probe index panic on truncated page entries

- [ ] `stream_index/page_reader_probe.go:86` panics with `unexpected EOF` when a probe page file is shorter than the cursor's `ContentLength`. Should return an error instead of panicking so `fsck` can report it and continue.

## Synthetic tai disambiguation for `der import`

- [ ] FDR: Add sub-second tai disambiguation during import to eliminate objectId+tai collisions (10,731 in production repo). Intervene in `remote_transfer/main.go` before `importNewObject()` commit: group by objectId+tai, assign incrementing attosecond offsets for duplicates, re-sign affected objects. Caveat: changes object digests vs source repo, requires `OverwriteSignatures`.

## `go mod tidy` — circular dependency dodder ↔ chrest

- [x] `go mod tidy` fails resolving `code.linenisgreat.com/dodder/go/src/bravo/ohio` and `code.linenisgreat.com/dodder/go/src/bravo/ui` — fixed by updating chrest upstream

## WASM workspace modules

- [ ] WASM interface for repo/domain ops (blob store, config) — store_fs currently gets these from env_repo.Env
- [ ] WASM-compatible replacement for `exec.Command` in `RunMergeTool` (`store_fs/merge.go`) — interactive merge tool invocation
- [ ] WASM-compatible replacement for `files.OpenFiles` in `OpenFiles.Run` (`store_fs/open_files.go`) — opens files in user's editor
- [ ] WASM-compatible replacement for `os.Stdin/Stdout/Stderr` in `RunMergeTool` (`store_fs/merge.go`) — terminal I/O for interactive merge

## Gob removal follow-ups

- [ ] FDR: evaluate making `delta/objects` metadata fields private after full gob removal
- [ ] Evaluate removing `hotel/log_remote_inventory_lists` entirely (has TODO suggesting deprecation)

## CLAUDE.md improvements (from transcript analysis)

- [ ] add instruction: ALWAYS use `just test*` recipes; NEVER run bats/go-test/fixture-generation directly
- [ ] add instruction: BATS fixture tests use `get_fixture_type_sig` for signatures; fresh-store tests use `--regexp`
- [ ] add instruction: NEVER call `errors.Is` when err might be EOF; use `errors.IsEOF()` guard first
- [ ] add instruction: when bumping store version, do NOT remove old version's codec/gob support
- [ ] add instruction: document "lock" dual meaning — content locks (metadata) vs filesystem mutex (LockSmith)
- [ ] add instruction: trailing whitespace matters in dodder output; use `xxd` to debug invisible mismatches

## Blob store ordering from config

- [ ] Wire blob-stores list from konfig into BlobStoreEnv ordering (replace alphabetical sort in setupStores)

## Archive store foreign digest support

- [ ] Implement `BlobForeignDigestAdder` for inventory archive stores. Idea: use symlinks in the embedded loose blob directory pointing to packed blob entries, so `HasBlob(foreignDigest)` resolves via the loose store fallback. Requires solving the read path (symlink target is a packfile, not a single blob file). See `docs/plans/2026-02-23-sync-cross-hash-design.md`.

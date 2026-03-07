# TODO

- [ ] fix rebase / merge issues caused by `sc merge`:
  $ sc merge smart-rowan 
  TAP version 14
  not ok 1 - rebase smart-rowan
    ---
    message: git rebase master -i: exit status 1
    output: |
  Auto-merging TODO.md
      CONFLICT (content): Merge conflict in TODO.md
      error: could not apply 2709d1572... refactor: migrate to bats-island shared library for test isolation
      hint: Resolve all conflicts manually, mark them as resolved with
      hint: "git add/rm <conflicted_files>", then run "git rebase --continue".
      hint: You can instead skip this commit: run "git rebase --skip".
      hint: To abort and get back to the state before "git rebase", run "git rebase --abort".
      hint: Disable this message with "git config set advice.mergeConflict false"
      Could not apply 2709d1572... # refactor: migrate to bats-island shared library for test isolation
    severity: fail
- [ ] migrate to purse-first's tap-dancer (drop local go.mod replace directive)
- [ ] confirm usage of modern nix monorepo
- [ ] use latest tap output
- [ ] explore using sigil as continuation operator in v14 fixed stream index
- [ ] expore using content offsets instead of signatures in v14 fixed stream
  index
- [ ] run performance tests on packfiles
- [ ] run performance v14 fixed stream index
- [ ] verify encryption support in packfiles
- [ ] add support for n / m sapir recovery for piv encryption
- [x] migrate to `bats_load_library bats-island` (replace inline set_xdg/setup_test_home/chflags_and_rm)

- [x] Debug `just test-bats-update-fixtures` failure: fixed by removing `rm -rf` from bats-island's `chflags_and_rm` (renamed to `chflags_nouchg`) so `--no-tempdir-cleanup` is respected.

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

- [x] `go mod tidy` fails resolving `code.linenisgreat.com/dodder/go/src/bravo/ohio` and `code.linenisgreat.com/dodder/go/src/bravo/ui` — fixed by updating chrest upstream

## WASM workspace modules

- [ ] WASM interface for repo/domain ops (blob store, config) — store_fs currently gets these from env_repo.Env
- [ ] WASM-compatible replacement for `exec.Command` in `RunMergeTool` (`store_fs/merge.go`) — interactive merge tool invocation
- [ ] WASM-compatible replacement for `files.OpenFiles` in `OpenFiles.Run` (`store_fs/open_files.go`) — opens files in user's editor
- [ ] WASM-compatible replacement for `os.Stdin/Stdout/Stderr` in `RunMergeTool` (`store_fs/merge.go`) — terminal I/O for interactive merge

## Gob removal: store_config compiled struct

- [ ] Redesign `oscar/store_config` compiled struct persistence to use a non-gob format (the struct must remain persisted for performance; needs a marshaling redesign)
- [ ] FDR: evaluate making `delta/objects` metadata fields private after full gob removal
- [ ] Evaluate removing `hotel/log_remote_inventory_lists` entirely (has TODO suggesting deprecation)

## Archive store foreign digest support

- [ ] Implement `BlobForeignDigestAdder` for inventory archive stores. Idea: use symlinks in the embedded loose blob directory pointing to packed blob entries, so `HasBlob(foreignDigest)` resolves via the loose store fallback. Requires solving the read path (symlink target is a packfile, not a single blob file). See `docs/plans/2026-02-23-sync-cross-hash-design.md`.

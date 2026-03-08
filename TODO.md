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

## `der import` bugs — phase 3: fix root causes

Error reporting for all 4 bugs landed in `420a114d0` (phase 1+2: rich error
types, `-continue-on-error` flag, error logfile). Root causes remain.

- [x] **Bug 1 — cross-pubkey merge crash.** Fixed: `importLeaf` now skips
  `MergeCheckedOut` when `parentNegotiator` is nil (always for imports). Without
  a parent negotiator, diff3 uses an empty base and always produces false
  conflicts. Imports accept the remote version directly; use `pull` for proper
  conflict detection with parent negotiation.

- [ ] **Bug 2 — batch dedup drops distinct objects.** Deduper uses
  `PurposeV5MetadataDigestWithoutTai` — two objects with different TAIs but
  identical metadata+blob get the same key, so only the first imports. Now
  reported as `ErrDeduped` with count. **Fix:** include TAI in the dedup
  digest (different Purpose constant) or scope key to ObjectId+TAI. Need to
  check whether TAI-exclusion was intentional for re-timestamped objects.

- [ ] **Bug 3 — blobless type definitions silently dropped.** Type-genre
  objects with null blob/pubkey/sig enter `importLeaf`, get signed by
  `FinalizeAndSignOverwrite`, but produce no stored output. Now detected
  early as `ErrBloblessTypeSkipped`. **Fix:** either (a) import as
  metadata-only type stubs (skip blob requirement for Type genre), or (b)
  reject the inventory list as malformed if type blobs are missing. Option
  (b) is safer since downstream objects need the type blob for lock
  resolution.

- [ ] **Bug 4 — ObjectId+TAI collisions (10,731 in production).** Different
  objects share ObjectId+TAI from sub-second writes in source repo. Now
  detected as `ErrObjectIdTaiCollision` when blob digests differ. Note:
  collision check compares blob digests (not object digests) because
  `FinalizeAndSignOverwrite` produces nondeterministic signatures. **Fix:**
  FDR for synthetic tai disambiguation — assign incrementing attosecond
  offsets during import, re-sign affected objects. Requires
  `OverwriteSignatures`. Alternative: use mother sig as secondary key.

## Semantic diffing to replace diff3

- [ ] FDR: replace filesystem diff3 merge with semantic diffing using the type system. Current merge in `MakeMergedTransacted` checks out objects to temp files and runs diff3 on text-rendered representations. This requires filesystem checkouts even for objects with no workspace presence, and conflates two checkout concepts: workspace (PWD) and temp (merge resolution). Semantic diffing would operate on the in-memory object model directly, eliminating the filesystem dependency. `store_fs` and `env_workspace` need types to discriminate between workspace and temp checkouts until this lands.

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

## Tag/Reference unification

- [ ] FDR: Unify Tags and References into single ContainedObjects collection
- [ ] Use doddish.Scanner to distinguish tags from references in OpTagSeparator parser instead of strings.Contains("/") heuristic


## Archive store foreign digest support

- [ ] Implement `BlobForeignDigestAdder` for inventory archive stores. Idea: use symlinks in the embedded loose blob directory pointing to packed blob entries, so `HasBlob(foreignDigest)` resolves via the loose store fallback. Requires solving the read path (symlink target is a packfile, not a single blob file). See `docs/plans/2026-02-23-sync-cross-hash-design.md`.

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

## `der import` bugs

Error reporting for all 4 bugs landed in `420a114d0` (rich error types,
`-continue-on-error` flag, error logfile). Bug 1 root cause fixed separately.

- [x] **Bug 1 — cross-pubkey merge crash.** Fixed: `importLeaf` now skips
  `MergeCheckedOut` when `parentNegotiator` is nil (always for imports).

- [x] **Bug 2 — batch dedup drops distinct objects.** Resolved: TAI-exclusion
  in dedup key is intentional (same content = same object regardless of
  timestamp). `ErrDeduped` reporting is the correct behavior.

- [ ] **Bug 3 — blobless type definitions silently dropped.** Type-genre
  objects with null blob/pubkey/sig enter `importLeaf`, get signed by
  `FinalizeAndSignOverwrite`, but produce no stored output. Now detected
  early as `ErrBloblessTypeSkipped`. Root-cause handling subsumed by
  two-phase import (see below).

- [ ] **Bug 4 — ObjectId+TAI collisions (10,731 in production).** Different
  objects share ObjectId+TAI from sub-second writes in source repo. Now
  detected as `ErrObjectIdTaiCollision` when blob digests differ. Root-cause
  handling subsumed by two-phase import (see below).

## Two-phase `der import` with topographic processing

- [ ] FDR: Redesign `der import` as a two-phase pipeline: (1) plan+validate the
  inventory list (topographic ordering, blobless type detection, TAI collision
  detection, dedup summary), then (2) review+commit. Topographic processing
  ensures types are imported before objects that reference them. The planning
  phase surfaces all issues (bugs 2-4) upfront instead of mid-stream.

## Semantic diffing to replace diff3

- [ ] FDR: replace filesystem diff3 merge with semantic diffing using the type system. Current merge in `MakeMergedTransacted` checks out objects to temp files and runs diff3 on text-rendered representations. This requires filesystem checkouts even for objects with no workspace presence, and conflates two checkout concepts: workspace (PWD) and temp (merge resolution). Semantic diffing would operate on the in-memory object model directly, eliminating the filesystem dependency. `store_fs` and `env_workspace` need types to discriminate between workspace and temp checkouts until this lands.

## Probe index panic on truncated page entries

- [x] `stream_index/page_reader_probe.go:86` panics with `unexpected EOF` when a probe page file is shorter than the cursor's `ContentLength`. Should return an error instead of panicking so `fsck` can report it and continue.


## `go mod tidy` — circular dependency dodder ↔ chrest

- [x] `go mod tidy` fails resolving `code.linenisgreat.com/dodder/go/src/bravo/ohio` and `code.linenisgreat.com/dodder/go/src/bravo/ui` — fixed by updating chrest upstream

## WASM workspace modules

- [ ] WASM interface for repo/domain ops (blob store, config) — store_fs currently gets these from env_repo.Env
- [ ] WASM-compatible replacement for `exec.Command` in `RunMergeTool` (`store_fs/merge.go`) — interactive merge tool invocation
- [ ] WASM-compatible replacement for `files.OpenFiles` in `OpenFiles.Run` (`store_fs/open_files.go`) — opens files in user's editor
- [ ] WASM-compatible replacement for `os.Stdin/Stdout/Stderr` in `RunMergeTool` (`store_fs/merge.go`) — terminal I/O for interactive merge

## Gob removal follow-ups

- [ ] FDR: evaluate making `delta/objects` metadata fields private after full gob removal
- [x] Evaluate removing `hotel/log_remote_inventory_lists` entirely — **verdict:
  keep for now.** Actively used by `tango/remote_http/{client,server_repo}` for
  deduplicating inventory list transfers. The `v0.go:20` TODO ("not really used")
  is stale; it IS used but only by those 2 modules. Removal would require either
  dropping dedup (accepting duplicate transfers) or inlining the logic. Low
  priority — gob serialization in this package is a better target for eventual
  cleanup.

## CLAUDE.md improvements (from transcript analysis)

- [x] add instruction: ALWAYS use `just test*` recipes; NEVER run bats/go-test/fixture-generation directly
- [x] add instruction: BATS fixture tests use `get_fixture_type_sig` for signatures; fresh-store tests use `--regexp`
- [x] add instruction: NEVER call `errors.Is` when err might be EOF; use `errors.IsEOF()` guard first
- [x] add instruction: when bumping store version, do NOT remove old version's codec/gob support
- [x] add instruction: document "lock" dual meaning — content locks (metadata) vs filesystem mutex (LockSmith)
- [x] add instruction: trailing whitespace matters in dodder output; use `xxd` to debug invisible mismatches

## Blob store ordering from config

- [x] Wire blob-stores list from konfig into BlobStoreEnv ordering (replace alphabetical sort in setupStores)

## Tag/Reference unification

- [ ] FDR: Unify Tags and References into single ContainedObjects collection
- [x] Use doddish.Scanner to distinguish tags from references in OpTagSeparator parser instead of strings.Contains("/") heuristic


## Abbreviation: render format ID separately from abbreviated digest

- [ ] `addMarklIdAbbreviated` in `object_metadata_box_builder` abbreviates the full `markl.Id.String()` (which includes the format HRP, e.g., `blake2b256-...`). The format ID portion should not be abbreviated — only the digest payload after the HRP should go through the tridex. Render as `purposeId@formatId-abbreviatedDigest`.

## Abbreviation store: ad-hoc persistence

- [ ] Extend `store_abbr.InMemoryIndex` to support ad-hoc persistence (write/read to a path without requiring `env_repo.Env`). In-memory is sufficient for the import plan; persistence would allow caching abbreviation indexes across runs.

## Topological sort: include object references

- [ ] Incorporate object references into the import_plan dependency graph, not just type references. References can point to any object or blob (blob references may not yet be implemented). Currently only type→dependent edges are sorted; objects referencing other objects are not ordered.

## UI library refactor

- [ ] refactor env_ui into lib and add huh / lipgloss to it

## Archive store foreign digest support

- [ ] Implement `BlobForeignDigestAdder` for inventory archive stores. Idea: use symlinks in the embedded loose blob directory pointing to packed blob entries, so `HasBlob(foreignDigest)` resolves via the loose store fallback. Requires solving the read path (symlink target is a packfile, not a single blob file). See `docs/plans/2026-02-23-sync-cross-hash-design.md`.

## Purse-first integration

- [ ] FDR: purse-first framework integration for madder MCP server
- [x] Once purse-first install_mcp branch lands: add install-mcp command to madder using `app.InstallMCP()` from go-mcp

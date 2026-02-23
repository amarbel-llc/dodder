# TODO

- [ ] Debug `just test-bats-update-fixtures` failure: bats succeeds when run directly from `zz-tests_bats/` but fails when invoked through the justfile recipe chain. Likely a working directory or environment variable propagation issue. The `cp` command can't find `.dodder` in the bats temp dir, suggesting the fixture generation test silently fails or the temp dir path extraction breaks.

## Temporary backwards-compat: `!` blob path fallback in triple-hyphen parser

- [ ] Remove the `strings.Contains(typeString, "/")` fallback in `text_parser2.readType` (`india/object_metadata_fmt_triple_hyphen/text_parser2.go`). Old checked-out zettel files used `! <path>` for blob references; new format uses `@ <path>`. Once all workspaces have been re-checked-out, this shim can be deleted.

## `der import` bugs (found during migration validation 2026-02-22)

- [ ] `der import` crashes with "all FD's are empty" when importing an object whose ID already exists in the store under a different pubkey. The error originates in `store_fs/merge.go:327` (`checkoutOneForMerge`) and cascades recursively through `Import.Run`. Workaround: exclude already-existing objects from the import file.

- [ ] `der import` silently skips objects that share a blob hash with another entry in the same batch. Only the first object per unique blob hash is imported; the rest are dropped with no error and exit 0. Workaround: import shared-blob objects individually (one entry per file) or run multiple passes.

- [ ] `der import` silently skips blobless type definitions (e.g. `[!opml 2097748458.73047 !toml-type-v1]` — no `@sha256-...` blob ref, no pubkey, no sig). These entries produce no output and no error. This causes downstream failures when importing objects that depend on those types ("failed to read current lock object").

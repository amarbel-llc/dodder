# zz-tests_bats

BATS integration suite. General conventions (assertion strictness, fixture
workflow, which `just` recipes to run) live in the repo-root CLAUDE.md ---
this file holds suite-local patterns.

## Blob store test patterns

Blob store CLI surface is the standalone `madder` binary, driven via the
`run_madder` helper in `lib/common.bash` (sets ceiling dirs and timeout,
mirrors `run_dodder`). The dedicated tests live in `current_version/blob_store_*.bats`.

### Creating a shared store and initializing a repo against it

Under the write_through multi default (FDR-0016 D1 / #223), `dodder init
-blob_store-id <name>` attaches the named store as a digest-pinned
READ-ONLY fallback --- writes land in the repo's scope-shared write store
(`default-local`, or `.default-local` for a CWD-scoped repo), never in the
named store. Tests that need the named store populated must copy into it
explicitly with `madder sync` (canonical example: `current_version/import.bats`):

```bash
setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"
  export output

  set_xdg "$BATS_TEST_TMPDIR"

  # User-scoped store (unprefixed = XDG user); must exist before init
  run_madder init shared
  assert_success

  run_dodder init \
    -yin <(cat_yin) -yang <(cat_yang) \
    -encryption none \
    -blob_store-id shared \
    .default
  assert_success

  create_test_zettels

  # Populate the read-only fallback explicitly
  run_madder sync .default-local shared
  assert_success
}
```

`set_xdg` ensures outer and inner repos share the same XDG namespace, which
is what makes an unprefixed store ID resolve to the same on-disk store
(`$XDG_DATA_HOME/madder/blob_stores/<name>`) across repos.

### Querying store config

```bash
run_madder info-repo compression-type
assert_output 'zstd'

# Two args: store ID + key
run_madder info-repo .archive encryption
```

### Archive stores

```bash
run_madder init-inventory-archive -encryption generate .archive
run_madder write -format tap .archive <(echo content)
run_madder pack .archive
```

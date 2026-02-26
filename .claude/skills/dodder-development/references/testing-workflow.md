# Testing Workflow

Dodder uses Go unit tests for package-level logic and BATS integration tests for
end-to-end CLI behavior. Fixtures committed to git provide reproducible test
data across store versions.

## BATS Test Structure

### Layout

Integration tests live in `zz-tests_bats/` with 40+ `.bats` files. Each file
corresponds to a dodder command or feature area:

- `init.bats`, `new.bats`, `show.bats` -- core command tests
- `organize.bats`, `checkout.bats`, `checkin.bats` -- workspace operations
- `clone.bats`, `push.bats`, `pull.bats` -- remote operations
- `migration/main.bats` -- store version migration tests
- `blob_store_cat_ids.bats`, `blob_store_write.bats` -- blob store operations

### Framework

Tests use BATS (bats-core) with three support libraries:

- **bats-support** -- core test utilities
- **bats-assert** -- assertion functions (`assert_success`, `assert_failure`,
  `assert_output`, `assert_line`)
- **bats-assert-additions** -- project-specific assertion extensions

### Timeout

Each test has a 5-second timeout (`BATS_TEST_TIMEOUT=5` in the justfile).

### Common Setup

Every `.bats` file loads `common.bash` in its `setup()` function:

```bash
setup() {
    load "$(dirname "$BATS_TEST_FILE")/common.bash"
    export output
}
```

`common.bash` handles:

- Pushing into `$BATS_TEST_TMPDIR` for isolation
- Loading bats-support, bats-assert, and bats-assert-additions
- Defining `set_xdg()` for XDG directory overrides
- Providing helper functions (`run_dodder`, `run_dodder_init_disable_age`, etc.)
- Setting default dodder flags for reproducible output

### Teardown

```bash
teardown() {
    chflags_and_rm
}
```

The `chflags_and_rm` function removes immutable file flags (set by dodder's
internal file locking) before cleaning up temporary directories.

## Fixture Lifecycle

### Storage

Fixtures live in `zz-tests_bats/migration/v{VERSION}/.xdg/` where `{VERSION}`
matches the store version number. Current fixture versions include v0 through
v13, plus `yin` and `yang` directories for specific test scenarios.

### Source of Truth

Inventory lists (text format) are the authoritative record. Binary indexes and
caches are derived and rebuilt during fixture generation. When fixtures are
regenerated, changes to inventory list content indicate actual format changes.

### Regeneration Workflow

When code changes alter the store format or binary layout:

1. Build the debug binary: `just build`
2. Regenerate fixtures: `just test-bats-update-fixtures`
3. Review the diff: `git diff -- zz-tests_bats/migration/`
4. Stage the regenerated fixtures: `git add zz-tests_bats/migration/`
5. Commit before running tests on a clean checkout.

Fixture generation requires a working debug binary. The `just
test-bats-update-fixtures` recipe builds automatically as a prerequisite.

### Fixture Generation Internals

The `zz-tests_bats/migration/generate_fixture.bash` script:

1. Reads `DODDER_VERSION` from the environment (set by the justfile to
   `v$(dodder info store-version)`).
2. Creates or overwrites the fixture directory for that version.
3. Initializes a fresh dodder repository with known seed data.
4. Populates objects, tags, types, and blob content.
5. Writes the resulting `.xdg/` directory structure as the fixture.

## Running Subsets

### Specific Files

```sh
just test-bats-targets clone.bats
just test-bats-targets init.bats new.bats show.bats
```

The `test-bats-targets` recipe sets `PATH` and `DODDER_BIN` to point at the
debug binary and forwards arguments to the BATS runner.

### Filter by Tag

```sh
just test-bats-tags migration
```

Tests annotated with `# bats test_tags=migration` (or any other tag) are
selected. Multiple tags combine with comma separation.

### All Integration Tests

```sh
just test-bats
```

This builds first, generates fixtures, then runs all `.bats` files and
`migration/*.bats` with parallel jobs (one per CPU).

### All Tests (Unit + Integration)

```sh
just test
```

Chains `test-go` (unit) and `test-bats` (integration).

## Writing New BATS Tests

### File Setup

Create `zz-tests_bats/{command}.bats`:

```bash
#! /usr/bin/env bats

setup() {
    load "$(dirname "$BATS_TEST_FILE")/common.bash"
    export output
}

teardown() {
    chflags_and_rm
}

function my_test_name { # @test
    run_dodder_init_disable_age

    run_dodder new -edit=false
    assert_success

    run_dodder show :z
    assert_success
    assert_output --partial "expected content"
}
```

### Naming Convention

- Test functions use the form `function descriptive_name { # @test }`.
- File names match the command under test: `{command}.bats`.

### Assertions

Common assertions from bats-assert:

- `assert_success` -- exit code 0
- `assert_failure` -- non-zero exit code
- `assert_output "exact match"` -- full output match
- `assert_output --partial "substring"` -- partial match
- `assert_output --regexp 'pattern'` -- regex match
- `assert_line "exact line"` -- match a specific output line
- `assert_line --index 0 "first line"` -- match by line index
- `refute_output "not this"` -- output does NOT contain

### Tagging Tests

Add a tag comment before the test function for filtering:

```bash
# bats test_tags=migration
function migration_v12_to_v13 { # @test
    # ...
}
```

Filter with `just test-bats-tags migration`.

## Store Version and Fixtures

### Version Detection

```sh
dodder info store-version
```

Returns the current store version number (integer). The fixture directory name
matches this number: `migration/v{N}/.xdg/`.

### Version Bump Implications

Without a version bump, reindex is not triggered and old binary data persists.
When a format change requires new binary layouts, bump the store version in
`charlie/store_version`, regenerate fixtures, and commit both the version change
and the new fixtures together.

### Fixture Directory Structure

```
zz-tests_bats/migration/
    v0/.xdg/
    v1/.xdg/
    ...
    v13/.xdg/
    yin/
    yang/
    generate_fixture.bash
    generate_fixture.bats
    main.bats
```

Each `v{N}/.xdg/` directory contains the full XDG directory tree (data, config,
state, cache, runtime) as populated by a fresh dodder init with known seed data
at that store version.

## Debugging Tips

### Binary Path

The justfile sets `DODDER_BIN` to `go/build/debug/dodder` and prepends
`go/build/debug/` to `PATH`. To run tests against a specific binary manually:

```sh
export PATH="/path/to/debug/binary:$PATH"
export DODDER_BIN="/path/to/debug/binary/dodder"
```

### Debug Build for Pool Poisoning

Ensure the debug binary is on PATH for pool poisoning to be active. Debug
binaries are compiled with `-tags debug`, which enables the `repool_debug.go`
double-repool detection and outstanding borrow tracking.

### Serial Execution

Eliminate test interference by running with a single job:

```sh
BATS_TEST_TIMEOUT=5 bats --jobs 1 some_test.bats
```

Or modify the justfile invocation temporarily.

### Debug Logging

Write debug output to stderr to avoid interfering with BATS output capture:

```go
fmt.Fprintf(os.Stderr, "debug: %v\n", value)
```

BATS captures stdout for assertions. Stderr output appears in the test log on
failure.

### Cleaning Stale Fixtures

If fixtures become stale or corrupted:

```sh
just zz-tests_bats/clean-fixtures
```

This resets the fixture directory for the current store version back to its
committed state using `git checkout`.

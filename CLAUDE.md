# CLAUDE.md

## Build

- `just build` --- builds debug and release Go binaries to `go/build/debug/` and
  `go/build/release/`
- Must run before integration tests if binaries are stale

## Testing

- **All tests:** `just test` (unit + integration). Exit code 0 means all tests
  passed --- do not grep output for "not ok" or "fail" to find failures, as
  those substrings appear in passing test names
  (e.g. `clean_fails_outside_workspace`).
- **Unit tests only:** `just test-go`
- **Integration tests only:** `just test-bats` (builds first, generates
  fixtures, runs BATS)
- **Specific test files:** `just test-bats-targets show.bats`
- **Filter by tag:** `just test-bats-tags migration`

**Always use just recipes from the repo root. Never run bats directly.** The
root recipes set BATS_BIN_DIR, DODDER_VERSION, inject the binary via
`--bin-dir`, and ensure fixtures exist. Running from `zz-tests_bats/` uses the
system `dodder` binary instead of the freshly built one.

**When a test fails**: run ONLY the failing test file with
`just test-bats-targets <file>.bats`. Do NOT re-run the entire suite.

## Test Directory Layout

- `zz-tests_bats/lib/common.bash` -- shared helpers, library loads, fixture
  value getters
- `zz-tests_bats/current_version/*.bats` -- current version's integration tests
- `zz-tests_bats/previous_versions/v14/` -- frozen snapshot: fixtures +
  .fixtures.env + tests
- `zz-tests_bats/previous_versions/main.bats` -- migration conformance test

## Fixture Workflow

Fixtures in `zz-tests_bats/previous_versions/v*/` are committed test data.

### When to Regenerate

Regenerate ONLY when: - Store version bumps (VCurrent changed in
store_version/main.go) - Store format changes that alter persisted data - Test
seed data changes (cat_yin, cat_yang, create_test_zettels)

Do NOT regenerate when: - CLI output format changes (update assertions, not
fixtures) - Adding new tests - Refactoring helpers

### Regeneration Steps

1.  `just test-bats-update-fixtures`
2.  Review: `git diff -- zz-tests_bats/previous_versions/`
3.  Commit fixtures + .fixtures.env together

### Version Bump Workflow

1.  `just test-bats-snapshot-version` (freeze current suite into
    previous_versions/vN/)
2.  Bump VCurrent in `go/internal/alfa/store_version/main.go`
3.  `just test-bats-update-fixtures` (generate new fixtures + .fixtures.env)
4.  Update test assertions for new behavior
5.  `just test` (verify everything passes)

## Bats Test Assertions

- **Fixture-specific values** (signatures, config SHA, type SHA) live in
  `previous_versions/$VERSION/.fixtures.env`, auto-generated during fixture
  creation. Access via helpers: `$(get_konfig_sha)`, `$(get_type_blob_sha)`,
  `$(get_fixture_type_sig)`.
- **Content-addressed blob hashes** (blake2b256-...) ARE deterministic and can
  be hardcoded in assertions.
- **Signatures** (ed25519_sig-...) are NOT deterministic -- ALWAYS use
  `$(get_fixture_type_sig)`.
- **Fresh-store tests** (`run_dodder_init_disable_age`) generate a new key each
  run -- use `assert_output --regexp -` with `! type@.*` for signatures.

## Critical Conventions

- **ALWAYS use `just test*` recipes** --- never run `bats`, `go test`, or
  fixture generation directly. The just recipes set BATS_BIN_DIR,
  DODDER_VERSION, inject the binary, and ensure fixtures exist.
- **BATS fixture tests** use `$(get_fixture_type_sig)` for signatures (not
  deterministic). Fresh-store tests (`run_dodder_init_disable_age`) use
  `assert_output --regexp` with `! type@.*` patterns instead.
- **NEVER call `errors.Is` when err might be EOF** --- use `errors.IsEOF()`
  guard first. The standard `errors.Is` does not handle the custom EOF wrapping.
- **When bumping store version**, do NOT remove the old version's codec/gob
  support. Old versions must remain decodable for migration.
- **"Lock" has two meanings** --- content locks (metadata on objects, managed by
  the lock command) vs filesystem mutexes (`LockSmith` in `env_repo`). Don't
  confuse them.
- **Trailing whitespace matters** in dodder output assertions. Use `xxd` or
  `cat -A` to debug invisible mismatches in BATS tests.
- **Do not recreate existing formatters.** When dodder already has a formatter
  (e.g., `box_format.BoxTransacted`, `sku_fmt` printers), use it via dependency
  injection or direct import --- never hand-build the same output with
  `fmt.Fprintf`/`strings.Builder`.
- **Adding or changing metadata fields requires binary codec updates
  ([#38](https://github.com/amarbel-llc/dodder/issues/38)).** Any field added to
  `objects.metadata`, `containedObject`, or `blobReferenceEntry` is NOT
  automatically serialized. Without encoder/decoder support in
  `india/stream_index`, the field will be populated during commit but silently
  lost on the next read from the store. See
  `go/internal/india/stream_index/CLAUDE.md` for the 4-file checklist.

## Common Issues

- **"dodder: command not found"** --- run `just build` first, or ensure you're
  in the nix devshell
- **BATS tests fail with stale fixtures** --- run
  `just test-bats-update-fixtures`, review diff, commit

## External Integrations (verify before committing)

  Integration          How to verify
  -------------------- -----------------------------------------------------
  pivy-agent ECDH      Round-trip: encrypt blob, decrypt with real token
  ECDSA P256 signing   Sign + verify with real key, not just one direction
  SSH agent protocol   Connect to real agent, list + sign + verify
  age encryption       Encrypt + decrypt round-trip
  WASM guest filters   Build guest, load in host, execute filter

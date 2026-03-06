# CLAUDE.md

## Build

- `just build` — builds debug and release Go binaries to `go/build/debug/` and `go/build/release/`
- Must run before integration tests if binaries are stale

## Testing

- **All tests:** `just test` (unit + integration)
- **Unit tests only:** `just test-go`
- **Integration tests only:** `just test-bats` (builds first, generates fixtures, runs BATS)
- **Specific test files:** `just test-bats-targets show.bats`
- **Filter by tag:** `just test-bats-tags migration`

**Always use just recipes. Never run bats directly.** The recipes set up
DODDER_VERSION, inject the binary via `--bin-dir`, and ensure fixtures exist.

**When a test fails**: run ONLY the failing test file with
`just test-bats-targets <file>.bats`. Do NOT re-run the entire suite.

## Test Directory Layout

- `zz-tests_bats/lib/common.bash` -- shared helpers, library loads, fixture value getters
- `zz-tests_bats/current_version/*.bats` -- current version's integration tests
- `zz-tests_bats/previous_versions/v14/` -- frozen snapshot: fixtures + .fixtures.env + tests
- `zz-tests_bats/previous_versions/main.bats` -- migration conformance test

## Fixture Workflow

Fixtures in `zz-tests_bats/previous_versions/v*/` are committed test data.

### When to Regenerate

Regenerate ONLY when:
- Store version bumps (VCurrent changed in store_version/main.go)
- Store format changes that alter persisted data
- Test seed data changes (cat_yin, cat_yang, create_test_zettels)

Do NOT regenerate when:
- CLI output format changes (update assertions, not fixtures)
- Adding new tests
- Refactoring helpers

### Regeneration Steps

1. `just test-bats-update-fixtures`
2. Review: `git diff -- zz-tests_bats/previous_versions/`
3. Commit fixtures + .fixtures.env together

### Version Bump Workflow

1. `just test-bats-snapshot-version` (freeze current suite into previous_versions/vN/)
2. Bump VCurrent in `go/internal/alfa/store_version/main.go`
3. `just test-bats-update-fixtures` (generate new fixtures + .fixtures.env)
4. Update test assertions for new behavior
5. `just test` (verify everything passes)

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

## Common Issues

- **"dodder: command not found"** — run `just build` first, or ensure you're in the nix devshell
- **BATS tests fail with stale fixtures** — run `just test-bats-update-fixtures`, review diff, commit

## External Integrations (verify before committing)

| Integration | How to verify |
|-------------|---------------|
| pivy-agent ECDH | Round-trip: encrypt blob, decrypt with real token |
| ECDSA P256 signing | Sign + verify with real key, not just one direction |
| SSH agent protocol | Connect to real agent, list + sign + verify |
| age encryption | Encrypt + decrypt round-trip |
| WASM guest filters | Build guest, load in host, execute filter |

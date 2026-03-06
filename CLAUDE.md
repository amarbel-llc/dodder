# CLAUDE.md

## Build

- `just build` — builds debug and release Go binaries to `go/build/debug/` and `go/build/release/`
- Must run before integration tests if binaries are stale

## Testing

- **All tests:** `just test` (unit + integration)
- **Unit tests only:** `just test-go`
- **Integration tests only:** `just test-bats` (builds first, generates fixtures, runs BATS)
- **Specific test files:** `just test-bats-targets clone.bats`
- **Filter by tag:** `just test-bats-tags migration`

## Fixture Workflow

Fixtures in `zz-tests_bats/migration/` are committed test data that integration tests copy and run against.

1. When code changes alter the store format, fixtures must be regenerated: `just test-bats-update-fixtures`
2. Review the diff: `git diff -- zz-tests_bats/migration/`
3. Regenerated fixtures **must** be `git add`ed and committed before integration tests will pass on a clean checkout
4. Fixture generation requires a working `dodder` debug binary (built by `just build`)

## Bats Test Assertions

- **Always verify with `just test`**, not bare `bats` — `just test` rebuilds the
  binary first. The devshell `dodder` may be a different version than the source,
  which changes `DODDER_VERSION`, fixture selection, and output format.
- **Fixture-based tests** (`copy_from_version`) have a fixed signing key — use
  literal `assert_output -` with exact signatures.
- **Fresh-store tests** (`run_dodder_init_disable_age`) generate a new key each
  run — use `assert_output --regexp -` with `! type@.*` for signatures.
- **Trailing whitespace is invisible in TAP output.** If a regex "should" match
  but doesn't, use `xxd` on the actual command output to check for hidden spaces.
- **`@.*`** is the safe regex for the blob hash line — matches both `@ ` (no
  blob, trailing space) and `@ blake2b256-...` (with blob).

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

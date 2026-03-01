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

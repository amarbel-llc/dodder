# Sandcastle --no-tempdir-cleanup

## Problem

Batman's bats wrapper runs tests inside sandcastle, which auto-creates a
`/tmp/sandcastle-XXX` temp directory and cleans it up on exit. Bats'
`--no-tempdir-cleanup` flag preserves the inner `bats-run-*` directory, but
sandcastle deletes the parent, making the bats flag useless.

This breaks dodder's fixture generation workflow, which relies on copying
`.dodder` from bats' preserved temp dir after the test run.

## Design

Three changes across two repos (`purse-first` and `dodder`):

### 1. Sandcastle CLI (purse-first/packages/sandcastle/src/cli.ts)

Restore features from v0.0.37 that were lost during monorepo migration, plus add
the new flag:

- **`--shell <shell>`** -- shell to execute the command with
- **`--tmpdir <path>`** -- override temp directory (user-provided = no
  auto-cleanup)
- **`--no-tempdir-cleanup`** -- skip cleanup of auto-created temp dir on exit
- **Tmpdir lifecycle**: create `sandcastle-XXX`, call
  `SandboxManager.setTmpdir()`, set TMPDIR in sandbox, cleanup on
  `process.exit` unless `--no-tempdir-cleanup` or `--tmpdir` was specified

### 2. Batman bats wrapper (purse-first/lib/packages/batman.nix)

- Parse `--no-tempdir-cleanup` in batman's flag loop (alongside `--bin-dir`,
  `--no-sandbox`)
- When set: pass `--no-tempdir-cleanup` to both sandcastle and bats

### 3. Dodder fixture generation (zz-tests_bats/migration/generate_fixture.bash)

- Replace `--no-sandbox` with `--no-tempdir-cleanup` (keeps sandbox active,
  only preserves temp dirs)

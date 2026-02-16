# Test Lifecycle Improvements Design

**Date:** 2026-02-16
**Branch:** tests-made-easier
**Status:** Approved

## Problem

The dodder testing lifecycle has several issues that make it hard for agents (and humans in worktrees) to build and test:

1. **Worktree devshell is broken** — `stringer`, `bats-support`, `sandcastle`, and `BATS_LIB_PATH` are missing because the root `flake.nix` doesn't expose the `go/` flake's devShell.
2. **Multi-step build-test chain is fragile** — build, fixture generation, and bats testing are separate steps with poor error reporting. `test-bats-update-fixtures` silently swallows fixture generation failures (missing `set -e`).
3. **No fast iteration path** — every test run regenerates fixtures even when they're current.

## Design

### 1. Flake devShells passthrough

Add `devShells` to root `flake.nix` so `use flake .` provides the full development environment. Already implemented and verified: `stringer`, `bats`, `bats-support`, `sandcastle`, and `BATS_LIB_PATH` are all available.

```nix
devShells = dodder-go.devShells.${system};
```

### 2. Root justfile rewrite

Rewrite root justfile test recipes with:

- **Labeled stages** — each pipeline step prints a header so failures are immediately locatable.
- **`set -euo pipefail`** — fixture generation failures are no longer silently swallowed.
- **Preflight checks** — before bats tests, verify `BATS_LIB_PATH`, `sandcastle`, and `DODDER_BIN` are available with actionable error messages.

Recipes:
- `test` — build, unit tests, smart fixture check, bats (full pipeline)
- `test-bats` — build, smart fixture check, bats
- `test-bats-quick` — build, bats without fixture generation (fast iteration)
- `test-bats-targets` — build, run specific bats test files
- `test-bats-tags` — build, run bats tests filtered by tag
- `test-bats-update-fixtures` — force regenerate with proper error propagation

### 3. Smart fixture skip

Compare the binary's store version against the existing fixture directory. Skip regeneration when fixtures are current:

```bash
current_version="v$($DODDER_BIN info store-version)"
fixture_dir="zz-tests_bats/migration/$current_version"
if [[ -d "$fixture_dir/.dodder" ]]; then
  echo "Fixtures up-to-date (store version $current_version), skipping generation"
else
  echo "Generating fixtures for store version $current_version..."
  just zz-tests_bats/test-generate_fixtures
fi
```

- `test` and `test-bats` use the smart skip.
- `test-bats-update-fixtures` always regenerates (explicit intent).

### 4. Generic worktree envrc

Add `~/eng/rcm-worktrees/envrc` with a generic envrc that sweatshop applies to all new worktrees:

```bash
source_env "$HOME"
dotenv ".env"
use flake .
dotenv "$HOME/.env-dev"
```

This works for any project whose root flake exposes a devShell.

## Out of scope

- TOML config loading bug (77 failing bats tests) — separate issue on this branch
- `go/justfile` and `zz-tests_bats/justfile` consolidation — follow-up pass
- Sweatshop per-project overlay framework — separate workstream, captured for future design

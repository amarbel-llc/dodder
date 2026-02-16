# Test Lifecycle Improvements Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make dodder's build-and-test lifecycle work reliably in worktrees with clear error reporting and fast iteration.

**Architecture:** Root flake exposes go/ devShell. Root justfile is the single entry point for all test workflows, with labeled stages, preflight checks, and smart fixture skipping.

**Tech Stack:** Nix flakes, just, bash, bats

---

### Task 1: Flake devShells passthrough

Already done and verified. This task is just the commit.

**Files:**
- Modified: `flake.nix` (line 18 added)

**Step 1: Verify the change**

Run: `nix flake show 2>&1 | grep devShells`
Expected: output includes `devShells.x86_64-linux.default`

**Step 2: Commit**

```bash
git add flake.nix
git commit -m "Expose go/ devShell from root flake

Passes through dodder-go.devShells so 'use flake .' at the repo root
provides the full development environment: Go toolchain with stringer,
bats with bats-support/bats-assert, sandcastle, BATS_LIB_PATH, and all
other dev tools."
```

---

### Task 2: Rewrite root justfile test recipes

**Files:**
- Modify: `justfile` (root)

**Step 1: Write the new justfile**

Replace the entire test section of the root `justfile` with the following. Keep the existing `dir_build`, `default`, and `build` recipes unchanged.

```just

dir_build := absolute_path("go/build")

default: build

#   ____        _ _     _
#  | __ ) _   _(_) | __| |
#  |  _ \| | | | | |/ _` |
#  | |_) | |_| | | | (_| |
#  |____/ \__,_|_|_|\__,_|
#

build:
  just go/build-go

#   _____         _
#  |_   _|__  ___| |_
#    | |/ _ \/ __| __|
#    | |  __/\__ \ |_
#    |_|\___||___/\__|
#

# Run all tests: build, unit tests, fixture generation (if needed), bats.
test: build test-go test-bats

# Run unit tests only.
test-go *flags:
  just go/test-go-unit {{flags}}

# Run bats integration tests, regenerating fixtures only if needed.
test-bats: build _test-bats-preflight _test-bats-ensure-fixtures _test-bats-run

# Run bats integration tests with existing fixtures (no generation).
test-bats-quick: build _test-bats-preflight _test-bats-run

# Run specific bats test files.
test-bats-targets *targets: build _test-bats-preflight
  #!/usr/bin/env bash
  set -euo pipefail
  export PATH="{{dir_build}}/debug:$PATH"
  export DODDER_BIN="{{dir_build}}/debug/dodder"
  just zz-tests_bats/test-targets {{targets}}

# Run bats tests filtered by tag.
test-bats-tags *tags: build _test-bats-preflight
  #!/usr/bin/env bash
  set -euo pipefail
  export PATH="{{dir_build}}/debug:$PATH"
  export DODDER_BIN="{{dir_build}}/debug/dodder"
  just zz-tests_bats/test-tags {{tags}}

# Force-regenerate fixtures. Review diff, then git add + commit.
test-bats-update-fixtures: build _test-bats-preflight
  #!/usr/bin/env bash
  set -euo pipefail
  export PATH="{{dir_build}}/debug:$PATH"
  export DODDER_BIN="{{dir_build}}/debug/dodder"

  echo "==> Regenerating fixtures..."
  just zz-tests_bats/test-generate_fixtures

  echo ""
  echo "==> Fixture changes:"
  git diff --stat -- zz-tests_bats/migration/
  echo ""
  echo "Review changes with: git diff -- zz-tests_bats/migration/"
  echo "Then: git add zz-tests_bats/migration/ && git commit -m 'Update test fixtures'"

# Preflight: verify bats dependencies are available.
[private]
_test-bats-preflight:
  #!/usr/bin/env bash
  set -euo pipefail
  ok=true

  if [[ -z "${BATS_LIB_PATH:-}" ]]; then
    echo "error: BATS_LIB_PATH is not set." >&2
    echo "  Are you in the nix devshell? Run: nix develop" >&2
    ok=false
  fi

  if ! command -v sandcastle &>/dev/null; then
    echo "error: sandcastle is not on PATH." >&2
    echo "  Are you in the nix devshell? Run: nix develop" >&2
    ok=false
  fi

  if ! command -v bats &>/dev/null; then
    echo "error: bats is not on PATH." >&2
    echo "  Are you in the nix devshell? Run: nix develop" >&2
    ok=false
  fi

  if [[ "$ok" != "true" ]]; then
    exit 1
  fi

# Smart fixture generation: skip if fixtures exist for current store version.
[private]
_test-bats-ensure-fixtures $PATH=(dir_build / "debug" + ":" + env("PATH")) $DODDER_BIN=(dir_build / "debug" / "dodder"):
  #!/usr/bin/env bash
  set -euo pipefail
  current_version="v$("$DODDER_BIN" info store-version)"
  fixture_dir="zz-tests_bats/migration/$current_version"

  if [[ -d "$fixture_dir/.dodder" ]]; then
    echo "==> Fixtures up-to-date (store version $current_version), skipping generation"
  else
    echo "==> Generating fixtures for store version $current_version..."
    just zz-tests_bats/test-generate_fixtures
  fi

# Run bats tests (no build, no fixture generation).
[private]
_test-bats-run $PATH=(dir_build / "debug" + ":" + env("PATH")) $DODDER_BIN=(dir_build / "debug" / "dodder"):
  #!/usr/bin/env bash
  set -euo pipefail
  echo "==> Running bats integration tests..."
  just zz-tests_bats/test
```

**Step 2: Verify the preflight check catches missing tools**

Run (outside devshell): `just _test-bats-preflight`
Expected: errors about BATS_LIB_PATH, sandcastle, bats not being available.

**Step 3: Verify the preflight check passes inside devshell**

Run: `nix develop --command just _test-bats-preflight`
Expected: exits 0, no output.

**Step 4: Verify smart fixture skip works**

Run: `nix develop --command just _test-bats-ensure-fixtures`
Expected: "Fixtures up-to-date (store version vNN), skipping generation" (since fixtures already exist from the earlier test run).

**Step 5: Verify test-bats-quick runs without fixture generation**

Run: `nix develop --command just test-bats-quick`
Expected: builds, runs preflight, runs bats directly (no fixture generation step).

**Step 6: Commit**

```bash
git add justfile
git commit -m "Rewrite root justfile test recipes with preflight checks and smart fixtures

- test: full pipeline with labeled stages
- test-bats-quick: skip fixture generation for fast iteration
- Preflight checks verify BATS_LIB_PATH, sandcastle, bats before running
- Smart fixture skip: compare store version against existing fixtures
- All bash recipes use set -euo pipefail
- test-bats-update-fixtures no longer silently swallows generation failures"
```

---

### Task 3: Generic worktree envrc

**Files:**
- Create: `~/eng/rcm-worktrees/envrc`

**Step 1: Write the envrc**

```bash
# vim: ft=direnv
# Generic worktree envrc - applied by sweatshop to new worktrees.
# Requires the project's root flake.nix to expose a devShell.

source_env "$HOME"
dotenv_if_exists ".env"
use flake .
dotenv_if_exists "$HOME/.env-dev"
```

Note: uses `dotenv_if_exists` instead of `dotenv` so it doesn't fail when `.env` or `.env-dev` don't exist. The main dodder repo's `.envrc` uses `dotenv` which would fail — this generic version is more forgiving.

**Step 2: Verify the file is in the right place**

Run: `ls -la ~/eng/rcm-worktrees/envrc`
Expected: file exists.

**Step 3: Verify sweatshop would apply it**

The file lives at `~/eng/rcm-worktrees/envrc`. When sweatshop creates a worktree, `ApplyRcmOverlay` walks `rcm-worktrees/` and symlinks each file as `.<filename>` in the worktree. So `envrc` becomes `.envrc`.

No commit needed — this file lives outside the dodder repo.

---

### Task 4: Apply envrc to current worktree

This worktree already exists, so sweatshop's overlay hasn't run. Manually symlink.

**Step 1: Remove the .envrc we created during exploration**

```bash
rm /home/sasha/eng/worktrees/dodder/tests-made-easier/.envrc
```

**Step 2: Symlink from rcm-worktrees**

```bash
ln -s /home/sasha/eng/rcm-worktrees/envrc /home/sasha/eng/worktrees/dodder/tests-made-easier/.envrc
```

**Step 3: Allow direnv**

```bash
cd /home/sasha/eng/worktrees/dodder/tests-made-easier
direnv allow
```

**Step 4: Verify the devshell loads**

```bash
cd /home/sasha/eng/worktrees/dodder/tests-made-easier
which stringer bats sandcastle
echo $BATS_LIB_PATH
```

Expected: all three tools found, BATS_LIB_PATH set.

---

### Task 5: Clean up TODO-env-issues.md

The issues documented in `go/TODO-env-issues.md` are now resolved by the flake devShells passthrough.

**Files:**
- Delete: `go/TODO-env-issues.md`

**Step 1: Delete the file**

```bash
git rm go/TODO-env-issues.md
```

**Step 2: Commit**

```bash
git commit -m "Remove TODO-env-issues.md, resolved by root flake devShells passthrough"
```

---

### Task 6: End-to-end verification

**Step 1: Run the full test pipeline**

```bash
nix develop --command just test
```

Expected: build succeeds, unit tests pass, fixture check runs (either skips or generates), bats tests run. The 77 TOML-related bats failures are expected (pre-existing code issue, out of scope).

**Step 2: Run the quick test pipeline**

```bash
nix develop --command just test-bats-quick
```

Expected: builds, skips fixture generation entirely, runs bats directly.

**Step 3: Run specific test targets**

```bash
nix develop --command just test-bats-targets init.bats
```

Expected: builds, runs only init.bats tests.

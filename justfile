set output-format := "tap"

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

dir_build := absolute_path("go/build")

# Prevent dodder's findWorkspaceFile from walking above the BATS temp dir and
# discovering .dodder-workspace in the worktree. Without this, tests that expect
# "no workspace" would pass incorrectly when TMPDIR is inside the repo tree.
bats_ceiling := absolute_path("")

default: build test

#   ____        _ _     _
#  | __ ) _   _(_) | __| |
#  |  _ \| | | | | |/ _` |
#  | |_) | |_| | | | (_| |
#  |____/ \__,_|_|_|\__,_|
#

build:
  just go/build-go

#    ____ _               _
#   / ___| |__   ___  ___| | __
#  | |   | '_ \ / _ \/ __| |/ /
#  | |___| | | |  __/ (__|   <
#   \____|_| |_|\___|\___|_|\_\
#

# Run all static checks: vuln, vet, repool, seqerror.
check:
  just go/check

#   _____         _
#  |_   _|__  ___| |_
#    | |/ _ \/ __| __|
#    | |  __/\__ \ |_
#    |_|\___||___/\__|
#

# Run all tests: build, unit tests, fixture generation (if needed), bats.
test: build test-go test-bats test-bats-network

# Run unit tests only.
test-go *flags:
  just go/test-go-unit {{flags}}

# Run bats integration tests, regenerating fixtures only if needed.
test-bats: build _test-bats-ensure-fixtures _test-bats-run

# Run bats integration tests with existing fixtures (no generation).
test-bats-quick: build _test-bats-run

# Run specific bats test files.
test-bats-targets *targets: build
  GOMEMLIMIT=512MiB DODDER_CEILING_DIRECTORIES="{{bats_ceiling}}" BATS_BIN_DIR="{{dir_build}}/debug" just zz-tests_bats/test-targets {{targets}}

# Run bats tests filtered by tag.
test-bats-tags *tags: build
  GOMEMLIMIT=512MiB DODDER_CEILING_DIRECTORIES="{{bats_ceiling}}" BATS_BIN_DIR="{{dir_build}}/debug" just zz-tests_bats/test-tags {{tags}}

# Run bats tests requiring Unix sockets (no sandbox).
test-bats-no-sandbox: build
  GOMEMLIMIT=512MiB DODDER_CEILING_DIRECTORIES="{{bats_ceiling}}" BATS_BIN_DIR="{{dir_build}}/debug" just zz-tests_bats/test-tags-no-sandbox af_unix

# Run bats with race-instrumented binary to detect data races in pool reuse.
test-bats-race: build
  just go/test-bats-race

# Force-regenerate fixtures. Review diff, then git add + commit.
test-bats-update-fixtures: build
  #!/usr/bin/env bash
  set -euo pipefail
  export PATH="{{dir_build}}/debug:$PATH"

  echo "==> Regenerating fixtures..."
  just zz-tests_bats/test-generate_fixtures

  echo ""
  echo "==> Fixture changes:"
  git diff --stat -- zz-tests_bats/previous_versions/
  echo ""
  echo "Review changes with: git diff -- zz-tests_bats/previous_versions/"
  echo "Then: git add zz-tests_bats/previous_versions/ && git commit -m 'Update test fixtures'"

# Snapshot current test suite for future reference.
# Run BEFORE bumping VCurrent in store_version/main.go.
test-bats-snapshot-version: build
  #!/usr/bin/env bash
  set -euo pipefail
  export PATH="{{dir_build}}/debug:$PATH"
  v="v$(dodder info store-version)"
  dest="zz-tests_bats/previous_versions/$v"

  if [[ -f "$dest/lib/common.bash" ]]; then
    echo "==> Snapshot already exists for $v, skipping"
    exit 0
  fi

  echo "==> Snapshotting test suite as $v..."
  mkdir -p "$dest/lib"
  cp zz-tests_bats/lib/common.bash "$dest/lib/common.bash"
  cp zz-tests_bats/current_version/*.bats "$dest/"

  echo "==> Snapshot complete: $dest"
  echo "Now bump VCurrent, run 'just test-bats-update-fixtures', and commit."

#   _____            _
#  | ____|_  ___ __ | | ___  _ __ ___
#  |  _| \ \/ / '_ \| |/ _ \| '__/ _ \
#  | |___ >  <| |_) | | (_) | | |  __/
#  |_____/_/\_\ .__/|_|\___/|_|  \___|
#             |_|

# Start a local Radicale CalDAV server for haustoria prototyping.
# Data stored in /tmp/radicale-dodder/. Runs in foreground (Ctrl-C to stop).
[group('explore')]
explore-radicale:
  #!/usr/bin/env bash
  set -euo pipefail
  data_dir="/tmp/radicale-dodder"
  mkdir -p "$data_dir"

  cat > /tmp/radicale-dodder.toml <<TOML
  [server]
  hosts = 127.0.0.1:5232

  [auth]
  type = none

  [storage]
  filesystem_folder = $data_dir/collections
  TOML

  echo "==> Radicale CalDAV server at http://127.0.0.1:5232"
  echo "==> Data dir: $data_dir"
  echo "==> No auth — any username/password works"
  echo ""
  echo "Export for dodder:"
  echo "  export CALDAV_URL=http://127.0.0.1:5232/dodder/tasks.ics/"
  echo "  export CALDAV_USERNAME=dodder"
  echo "  export CALDAV_PASSWORD=dodder"
  echo ""
  nix run nixpkgs#radicale -- --config /tmp/radicale-dodder.toml

# Create a haustoria workspace pointing at local Radicale.
# Requires Radicale running (just explore-radicale) and env vars set.
# Creates a parent repo + workspace in /tmp/dodder-haustoria-explore/.
[group('explore')]
explore-haustoria-init: build
  #!/usr/bin/env bash
  set -euo pipefail
  export PATH="{{dir_build}}/debug:$PATH"

  if [[ -z "${CALDAV_URL:-}" ]]; then
    echo "Set CALDAV_URL, CALDAV_USERNAME, CALDAV_PASSWORD first."
    echo "See: just explore-radicale"
    exit 1
  fi

  base="/tmp/dodder-haustoria-explore"
  rm -rf "$base"
  mkdir -p "$base/parent" "$base/workspace"

  # Create parent repo
  cd "$base/parent"
  dodder init -repo_id .

  # Create workspace with haustoria
  cd "$base/workspace"
  dodder init-workspace -haustoria caldav -parent "$base/parent" haustoria-ws

  echo ""
  echo "==> Parent repo: $base/parent"
  echo "==> Workspace:   $base/workspace"
  echo "==> Run:"
  echo "  cd $base/workspace && dodder status"
  echo "  cd $base/workspace && dodder new -type '!task' 'buy groceries'"

# Show haustoria status for the explore workspace.
[group('explore')]
explore-haustoria-status: build
  #!/usr/bin/env bash
  set -euo pipefail
  export PATH="{{dir_build}}/debug:$PATH"
  cd /tmp/dodder-haustoria-explore/workspace
  dodder status

# Debug a specific bats test file with --no-tempdir-cleanup for inspection.
[group('explore')]
explore-bats-debug *targets: build
  GOMEMLIMIT=512MiB DODDER_CEILING_DIRECTORIES="{{bats_ceiling}}" BATS_BIN_DIR="{{dir_build}}/debug" just zz-tests_bats/test-targets --no-tempdir-cleanup {{targets}}

# Run bats tests that need local network binding (e.g. haustoria CalDAV).
test-bats-network *targets="current_version/haustoria_caldav.bats current_version/sftp.bats": build
  GOMEMLIMIT=512MiB DODDER_CEILING_DIRECTORIES="{{bats_ceiling}}" BATS_BIN_DIR="{{dir_build}}/debug" just zz-tests_bats/test-targets --allow-local-binding --allow-unix-sockets {{targets}}

# Smart fixture generation: skip if fixtures exist for current store version.
[private]
_test-bats-ensure-fixtures $PATH=(dir_build / "debug" + ":" + env("PATH")):
  #!/usr/bin/env bash
  set -euo pipefail
  current_version="v$(dodder info store-version)"
  fixture_dir="zz-tests_bats/previous_versions/$current_version"

  if [[ -d "$fixture_dir/.dodder" ]] && [[ -s "$fixture_dir/.fixtures.env" ]]; then
    echo "==> Fixtures up-to-date (store version $current_version), skipping generation"
  else
    echo "==> Generating fixtures for store version $current_version..."
    just zz-tests_bats/test-generate_fixtures
  fi

# Run bats tests (no build, no fixture generation).
# GOMEMLIMIT caps each dodder process at 512 MiB to prevent OOM on leak (#68).
[private]
_test-bats-run:
  @echo "==> Running bats integration tests..."
  GOMEMLIMIT=512MiB DODDER_CEILING_DIRECTORIES="{{bats_ceiling}}" BATS_BIN_DIR="{{dir_build}}/debug" just zz-tests_bats/test

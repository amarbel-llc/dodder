dir_build := justfile_directory() / "go/build"

# Prevent dodder's findWorkspaceFile from walking above the BATS temp dir and
# discovering .dodder-workspace in the worktree. Without this, tests that
# expect "no workspace" would pass incorrectly when TMPDIR is inside the repo
# tree. The same ceiling also bounds dodder's CWD-XDG-override discovery walk
# (DODDER_CEILING_DIRECTORIES); without a stable ceiling, the walk can reach
# .dodder/ inside fixture dirs and contaminate them.
#
# justfile_directory() resolves at parse time to the worktree root regardless
# of the caller's CWD, so this stays correct even if the caller invokes just
# from outside the worktree (e.g. some spinclass / pre-merge-hook flows).
# absolute_path("") would resolve relative to just's invocation CWD, which is
# fragile.
bats_ceiling := justfile_directory()

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
  GOMEMLIMIT=512MiB DODDER_CEILING_DIRECTORIES="{{bats_ceiling}}" MADDER_CEILING_DIRECTORIES="{{bats_ceiling}}" BATS_BIN_DIR="{{dir_build}}/debug" just zz-tests_bats/test-targets {{targets}}

# Run bats tests filtered by tag.
test-bats-tags *tags: build
  GOMEMLIMIT=512MiB DODDER_CEILING_DIRECTORIES="{{bats_ceiling}}" MADDER_CEILING_DIRECTORIES="{{bats_ceiling}}" BATS_BIN_DIR="{{dir_build}}/debug" just zz-tests_bats/test-tags {{tags}}

# Run bats tests requiring Unix sockets (no sandbox).
test-bats-no-sandbox: build
  GOMEMLIMIT=512MiB DODDER_CEILING_DIRECTORIES="{{bats_ceiling}}" MADDER_CEILING_DIRECTORIES="{{bats_ceiling}}" BATS_BIN_DIR="{{dir_build}}/debug" just zz-tests_bats/test-tags-no-sandbox af_unix

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

live_workspace := env("HOME") / "workspaces/dodder-haustoria-caldav/workspace"

# Run a dodder command in the live CalDAV workspace (no build).
[group('explore')]
explore-live *args:
  #!/usr/bin/env bash
  set -euo pipefail
  source "$HOME/.secrets.env"
  export PATH="{{dir_build}}/debug:$PATH"
  cd "{{live_workspace}}"
  dodder {{args}}

# Debug a specific bats test file with --no-tempdir-cleanup for inspection.
[group('explore')]
explore-bats-debug *targets: build
  GOMEMLIMIT=512MiB DODDER_CEILING_DIRECTORIES="{{bats_ceiling}}" MADDER_CEILING_DIRECTORIES="{{bats_ceiling}}" BATS_BIN_DIR="{{dir_build}}/debug" just zz-tests_bats/test-targets --no-tempdir-cleanup {{targets}}

# Run bats tests that need local network binding (haustoria CalDAV, dodder serve).
test-bats-network *targets="current_version/haustoria_caldav.bats current_version/haustoria_orgmode.bats current_version/sftp.bats current_version/serve.bats current_version/clone_port.bats": build
  GOMEMLIMIT=512MiB DODDER_CEILING_DIRECTORIES="{{bats_ceiling}}" MADDER_CEILING_DIRECTORIES="{{bats_ceiling}}" BATS_BIN_DIR="{{dir_build}}/debug" just zz-tests_bats/test-targets --allow-local-binding --allow-unix-sockets {{targets}}

#   ____      _
#  |  _ \ ___| | ___  __ _ ___  ___
#  | |_) / _ \ |/ _ \/ _` / __|/ _ \
#  |  _ <  __/ |  __/ (_| \__ \  __/
#  |_| \_\___|_|\___|\__,_|___/\___|
#

# Tag a Go module release. The "go/v" prefix is added for you, so pass
# the semver without it. Usage: just tag 0.1.0 "feat: something"
[group('release')]
tag version message:
  #!/usr/bin/env bash
  set -euo pipefail
  tag="go/v{{version}}"
  prev=$(git tag --sort=-v:refname -l "go/v*" | head -1)
  if [[ -n "$prev" ]]; then
    gum log --level info "Previous: $prev"
    git log --oneline "$prev"..HEAD -- go/
  fi
  git tag -s -m "{{message}}" "$tag"
  gum log --level info "Created tag: $tag"
  git push origin "$tag"
  gum log --level info "Pushed $tag"
  git tag -v "$tag"

# Sed-rewrite dodderVersion in flake.nix to the given semver. The
# version string is burnt into the binary at build time via -ldflags
# (see go/cmd/*/main.go and go/internal/victor/commands_dodder/version.go),
# so flake.nix is the single source of truth. No-op if already at the
# target version. Usage: just bump-version 0.1.1
[group('release')]
bump-version new_version:
  #!/usr/bin/env bash
  set -euo pipefail
  current=$(grep 'dodderVersion = ' flake.nix | sed 's/.*"\(.*\)".*/\1/')
  if [[ "$current" == "{{new_version}}" ]]; then
    gum log --level info "already at {{new_version}}"
    exit 0
  fi
  sed -i.bak 's/dodderVersion = "'"$current"'"/dodderVersion = "{{new_version}}"/' flake.nix && rm flake.nix.bak
  gum log --level info "bumped dodderVersion: $current → {{new_version}}"

# Cut a release: must be run on master. Bumps dodderVersion in
# flake.nix, commits the bump with a changelog-style message built
# from commits since the last go/v* tag, pushes master, then signs
# and pushes the go/v{{version}} tag. The "go/v" prefix is added for
# you, so pass the semver without it. Usage: just release 0.1.1
#
# Use `just tag <version> <message>` directly if you want to
# control the commit message yourself without bumping.
[group('release')]
release version:
  #!/usr/bin/env bash
  set -euo pipefail
  current_branch=$(git rev-parse --abbrev-ref HEAD)
  if [[ "$current_branch" != "master" ]]; then
    gum log --level error "just release must be run on master (currently on $current_branch)"
    exit 1
  fi
  prev=$(git tag --sort=-v:refname -l "go/v*" | head -1)
  header="release v{{version}}"
  if [[ -n "$prev" ]]; then
    summary=$(git log --format='- %s' "$prev"..HEAD -- go/)
    if [[ -n "$summary" ]]; then
      msg="$header"$'\n\n'"$summary"
    else
      msg="$header"
    fi
  else
    msg="$header"
  fi
  just bump-version "{{version}}"
  if ! git diff --quiet flake.nix; then
    git add flake.nix
    git commit -m "chore: release go/v{{version}}"
    git push origin master
    gum log --level info "pushed flake.nix bump to master"
  fi
  just tag "{{version}}" "$msg"

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
  GOMEMLIMIT=512MiB DODDER_CEILING_DIRECTORIES="{{bats_ceiling}}" MADDER_CEILING_DIRECTORIES="{{bats_ceiling}}" BATS_BIN_DIR="{{dir_build}}/debug" just zz-tests_bats/test

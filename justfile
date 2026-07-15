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

default: lint build test

#   _     _       _
#  | |   (_)_ __ | |_
#  | |   | | '_ \| __|
#  | |___| | | | | |_
#  |_____|_|_| |_|\__|
#

# Read-only formatting gate (treelint check). Runs first in the default lane
# so a formatting drift fails fast, before the expensive build/test. This is
# what the spinclass pre-merge hook (bare `just`) enforces.
lint:
  just go/check-treelint
  just go/check-shellcheck

#   ____        _ _     _
#  | __ ) _   _(_) | __| |
#  |  _ \| | | | | |/ _` |
#  | |_) | |_| | | | (_| |
#  |____/ \__,_|_|_|\__,_|
#

build:
  just go/build-go

# Regenerate the dodder.net seed-set type files (FDR-0010 Phase 3) into
# zz-seed/types/ from the table in go/cmd/dodder-gen_seed_types/table.go.
# Deterministic and idempotent (stable ordering, no timestamps; stale files
# pruned) — review the diff and commit. Agent dev-loop: seed-set table edits.
generate-seed-types:
  cd go && go run ./cmd/dodder-gen_seed_types -dir ../zz-seed/types

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

# Run all tests: unit + bats integration + fixture-generator smoke. The
# bats lane builds dodder internally inside the nix sandbox, so no `build`
# dep is needed for this path.
test: test-go test-bats test-bats-generate

# Run unit tests only.
test-go *flags:
  just go/test-go-unit {{flags}}

# Run the full bats integration suite inside the nix sandbox.
# Wraps `pkgs.testers.batsLane` from amarbel-llc/bats; binaries are
# injected via the lane's binaries map (DODDER_BIN, MADDER_BIN, …).
# No fixture-generation step: fixtures are committed under
# zz-tests_bats/previous_versions/v*/ and the lane consumes them
# directly. To regenerate, run `just test-bats-update-fixtures`.
test-bats:
  nix build .#bats-default --no-link --print-build-logs

# Smoke-test the fixture generator: build the `fixtures-current`
# derivation (which runs previous_versions/generate_fixture.bats in the
# sandbox) without materializing it. The generator is excluded from the
# normal bats lanes, so this is the only thing that runs it in CI — a
# store/CLI change that breaks generation (e.g. FDR-0020 making config
# non-queryable) fails here instead of silently bitrotting until the next
# manual `test-bats-update-fixtures`. See #272.
test-bats-generate:
  nix build .#fixtures-current --no-link --print-build-logs

# Run a per-tag bats lane (e.g. `just test-bats-tags haustoria`).
# The tag is the file_tags value from `# bats file_tags=foo`
# directives. Tags with embedded `:` are quoted into the flake
# attribute path automatically.
test-bats-tags *tags:
  nix build '.#bats-{{tags}}' --no-link --print-build-logs

# Run a single bats test file (or list of files) via the legacy
# batman path. Kept because the nix lane builder operates at the
# tag level — there is no per-file nix lane to dispatch a single
# `show.bats` against. For tag-level filtering, use
# `test-bats-tags <tag>` (faster, hermetic) instead. The debug-tagged
# dodder binary is resolved through the flake so this recipe doesn't
# need `just build` first.
test-bats-targets *targets:
  #!/usr/bin/env bash
  set -euo pipefail
  bin=$(nix build --no-link --print-out-paths .#dodder-debug)
  export PATH="$bin/bin:$PATH"
  GOMEMLIMIT=512MiB \
    DODDER_CEILING_DIRECTORIES="{{bats_ceiling}}" \
    MADDER_CEILING_DIRECTORIES="{{bats_ceiling}}" \
    just zz-tests_bats/test-targets {{targets}}

# As test-bats-targets, but bypasses the bats sandbox (--no-sandbox). Use
# when the bats fork's per-run network bridge can't initialize on this host
# ("failed to initialize Linux bridge: timeout waiting for bridge sockets");
# the standard current_version files are hermetic via tmpdir/XDG pinning, so
# they run correctly without the sandbox. The merge-hook nix lane stays the
# authoritative gate.
test-bats-targets-no-sandbox *targets:
  #!/usr/bin/env bash
  set -euo pipefail
  bin=$(nix build --no-link --print-out-paths .#dodder-debug)
  export PATH="$bin/bin:$PATH"
  GOMEMLIMIT=512MiB \
    DODDER_CEILING_DIRECTORIES="{{bats_ceiling}}" \
    MADDER_CEILING_DIRECTORIES="{{bats_ceiling}}" \
    just zz-tests_bats/test-targets-no-sandbox {{targets}}

# As test-bats-targets-no-sandbox, but also builds and exports
# MADDER_TEST_SFTP_SERVER (amarbel-llc/madder#177) for bats files that
# use zz-tests_bats/lib/sftp.bash. Only the nix bats lane
# (go/bats.nix's netCapExtraBinaries) wires that env var automatically;
# this batman-path recipe needs the explicit build + export. Ad-hoc
# debug recipe for iterating on SFTP-backed blob store tests --
# amarbel-llc/dodder#118 tracks the sandbox networking restriction that
# blocks the loopback bind even with the binary present. Pass
# madder_path to build dodder-debug against a locally checked-out (and
# possibly hand-patched) madder source tree instead of the pinned
# flake.lock rev -- used for adding temporary diagnostic
# fmt.Fprintf(os.Stderr, ...) instrumentation directly into madder's
# blob store code without needing to file/push/re-bump anything first.
[group('debug')]
debug-test-bats-sftp madder_path="" *targets:
  #!/usr/bin/env bash
  set -euo pipefail
  if [ -n "{{madder_path}}" ]; then
    bin=$(nix build --no-link --print-out-paths .#dodder-debug --override-input madder "path:$(realpath '{{madder_path}}')")
    madder_bin=$(nix build --no-link --print-out-paths .#madder-bin)
    export PATH="$bin/bin:$madder_bin/bin:$PATH"
  else
    bin=$(nix build --no-link --print-out-paths .#dodder-debug)
    export PATH="$bin/bin:$PATH"
  fi
  sftp_bin=$(nix build --no-link --print-out-paths .#madder-test-sftp-server)
  GOMEMLIMIT=512MiB \
    MADDER_TEST_SFTP_SERVER="$sftp_bin/bin/madder-test-sftp-server" \
    DODDER_CEILING_DIRECTORIES="{{bats_ceiling}}" \
    MADDER_CEILING_DIRECTORIES="{{bats_ceiling}}" \
    just zz-tests_bats/test-targets-no-sandbox {{targets}}

# Run bats with race-instrumented binary to detect data races in pool reuse.
test-bats-race:
  just go/test-bats-race

# Force-regenerate fixtures inside the nix sandbox, then materialize
# the result into the worktree. Review the diff and commit.
test-bats-update-fixtures:
  #!/usr/bin/env bash
  set -euo pipefail
  out=$(nix build .#fixtures-current --no-link --print-out-paths)
  ver=$(basename "$out"/v*)
  dest="zz-tests_bats/previous_versions/$ver"
  echo "==> Materializing $ver from $out"
  rm -rf "$dest"
  mkdir -p "$dest"
  cp -r --no-preserve=mode "$out/$ver"/. "$dest"/
  bin/chflags.bash -R nouchg "$dest"
  echo ""
  echo "==> Fixture changes:"
  git diff --stat -- "$dest"
  echo ""
  echo "Review with: git diff -- $dest"
  echo "Then: git add $dest && git commit -m 'Update test fixtures'"

# Regenerate golden files for approval-testing assertions (assert_golden /
# assert_golden_unsorted from zz-tests_bats/lib/golden.bash). Runs the target
# files with DODDER_UPDATE_GOLDENS=1 so each assert_golden writes its
# normalized $output into zz-tests_bats/**/goldens/<file>/<name>.txt instead of
# asserting. Uses the NO-SANDBOX batman path because this host's bats Linux
# bridge can't initialize; the merge-hook nix lane remains the authoritative
# gate (it consumes the committed goldens read-only). Defaults to all
# current_version files; pass specific files (e.g.
# `just test-bats-update-goldens current_version/show.bats
# previous_versions/main.bats`) to scope the regen. Review the diff and commit
# the goldens alongside the converted assertions.
test-bats-update-goldens *targets="current_version/*.bats":
  #!/usr/bin/env bash
  set -euo pipefail
  bin=$(nix build --no-link --print-out-paths .#dodder-debug)
  export PATH="$bin/bin:$PATH"
  GOMEMLIMIT=512MiB \
    DODDER_UPDATE_GOLDENS=1 \
    DODDER_CEILING_DIRECTORIES="{{bats_ceiling}}" \
    MADDER_CEILING_DIRECTORIES="{{bats_ceiling}}" \
    just zz-tests_bats/test-targets-no-sandbox {{targets}}

# Snapshot current test suite for future reference.
# Run BEFORE bumping VCurrent in store_version/main.go.
test-bats-snapshot-version:
  #!/usr/bin/env bash
  set -euo pipefail
  bin=$(nix build --no-link --print-out-paths .#dodder-debug)
  export PATH="$bin/bin:$PATH"
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
explore-haustoria-init:
  #!/usr/bin/env bash
  set -euo pipefail
  bin=$(nix build --no-link --print-out-paths .#dodder-debug)
  export PATH="$bin/bin:$PATH"

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
explore-haustoria-status:
  #!/usr/bin/env bash
  set -euo pipefail
  bin=$(nix build --no-link --print-out-paths .#dodder-debug)
  export PATH="$bin/bin:$PATH"
  cd /tmp/dodder-haustoria-explore/workspace
  dodder status

# Pull a full copy of the real, already-initialized XDG-user "default" repo
# (source of truth for the operator's actual notes) into a fresh workspace
# under this worktree's .tmp/, to investigate whether -ephemeral workspaces
# (and pull in general) bring in every object from the source rather than a
# scoped subset (dodder#TBD). Does NOT touch the source repo.
#
# NOTE: `dodder clone -direct <path>` looked like the obvious tool for this
# but is NOT usable against a plain XDG-home repo: -direct's only blob type
# (TomlLocalOverridePathV0) resolves via
# env_dir.MakeWithXDGRootOverrideHomeAndInitialize, whose XDG template
# ("$XDG_OVERRIDE/.$XDG_UTILITY_NAME/local/share", dewey
# internal/delta/xdg_defaults/main.go) inserts an extra `.dodder/local/share`
# segment that a standard home-XDG repo (built via the DIFFERENT
# MakeWithHomeAndInitialize / "$HOME/.local/share/$XDG_UTILITY_NAME"
# template) never has on disk -- confirmed by direct `ls` against the real
# repo (.dodder, config-seed, etc. are siblings, not nested under
# .dodder/local/share). No override value collapses the two templates to
# the same path. `clone.go`'s Run() also never reaches
# MakeHomeRepoRemote/IsHomeRepoParent (that path is pull/push/set_parent
# only, via ResolveImplicitDirectPath when a workspace's recorded parent
# resolves to the home repo) -- clone has no flag equivalent for it.
#
# So instead: init-workspace with NO -parent (defaults to the home repo,
# same resolution `-repo_id default` uses elsewhere; refuses to
# auto-create if missing) followed by a plain `pull`, which DOES reach
# MakeHomeRepoRemote for a home-repo parent.
[group('debug')]
debug-clone-default-into-tmp:
  #!/usr/bin/env bash
  set -euo pipefail
  bin=$(nix build --no-link --print-out-paths .#dodder-debug)
  export PATH="$bin/bin:$PATH"

  dest="{{ justfile_directory() }}/.tmp/debug-clone-default/$(date +%s)"
  mkdir -p "$dest"
  cd "$dest"

  # Default (-experimental-repo=true) with no -parent: repo-backed
  # workspace whose parent resolves to the home repo (V1 config with empty
  # ParentPath), the same resolution `new -ephemeral` uses -- this is what
  # makes ResolveImplicitDirectPath -> IsHomeRepoParent() true for `pull`.
  # (-experimental-repo=false, the run_dodder_init_disable_age bats
  # pattern, creates a different lightweight V0 workspace with no parent
  # pointer at all -- confirmed by reading runLightweight vs
  # runExperimentalRepo in init_workspace.go; that flag does NOT apply
  # here.)
  dodder init-workspace debug-ws

  # TEMPORARY: dumps the exact bytes hashed for project-catapult_lag's
  # digest recomputation during pull's re-verification, via debug
  # instrumentation added to object_fmt_digest.WriteDigest (main.go).
  # Substring match (not exact) since the object-id string format at this
  # call site isn't confirmed yet. Redirected to a file (not stderr
  # inline) to avoid any stream-mixing/truncation in the tool wrapper.
  # Remove this env var (and the instrumentation) once root-caused.
  debug_log="{{ justfile_directory() }}/.tmp/debug-digest.log"
  DODDER_DEBUG_DIGEST_ALL=1 DODDER_DEBUG_DIGEST_OBJECT_ID=catapult dodder pull 2>"$debug_log" || true
  echo "=== debug digest log ($debug_log) ==="
  grep -i catapult "$debug_log" || echo "(no 'catapult' lines found in debug log)"
  wc -l "$debug_log"

  echo ""
  echo "==> Workspace: $dest"
  echo "==> Run:"
  echo "  cd $dest && dodder show :z | wc -l"

# Run `dodder fsck` directly against the REAL live default repo (no
# pull/clone involved -- this is the check that reportedly passes clean
# for project-catapult_lag), with the SAME DODDER_DEBUG_VERIFY /
# DODDER_DEBUG_DIGEST instrumentation active, so the exact pubkey/digest/
# sig bytes fsck sees for this object can be diffed byte-for-byte against
# what pull captured (.tmp/debug-digest.log) for the SAME object-id.
[group('debug')]
debug-fsck-catapult-lag-with-instrumentation:
  #!/usr/bin/env bash
  set -euo pipefail
  bin=$(nix build --no-link --print-out-paths .#dodder-debug)
  export PATH="$bin/bin:$PATH"

  debug_log="{{ justfile_directory() }}/.tmp/debug-fsck-digest.log"

  # -recompute: without this, fsck only verifies the SIGNATURE against the
  # digest ALREADY STORED on disk -- it never calls WriteDigest, so it
  # can't reveal whether recomputing the digest from the decoded object's
  # fields would reproduce that same stored value (confirmed by an
  # earlier run of this recipe with no -recompute: zero DODDER_DEBUG_DIGEST
  # lines appeared, only DODDER_DEBUG_VERIFY). With -recompute, fsck DOES
  # call WriteDigest, so its bytes= output can be diffed directly against
  # pull's bytes= for the same object/Tai (.tmp/debug-digest.log).
  DODDER_DEBUG_DIGEST_ALL=1 DODDER_DEBUG_DIGEST_OBJECT_ID=catapult \
    dodder fsck -recompute -repo_id default 'project-catapult_lag+:e' 2>"$debug_log" || true

  echo "=== fsck output ==="
  cat "$debug_log" | grep -v DODDER_DEBUG || true

  echo ""
  echo "=== debug digest/verify log ($debug_log) ==="
  grep -A6 'DODDER_DEBUG_VERIFY' "$debug_log" || echo "(no DODDER_DEBUG_VERIFY lines found -- fsck may not be calling pubKey.Verify for this object by default)"

# Sanity check BEFORE writing a new regression test: does `dodder new
# -object-id X -description '...'` with a literal embedded blank line
# (bash $'...\n\n...' construction) actually commit a Description with
# REAL 0x0A bytes preserved, or does something in CLI flag parsing / the
# commit path ALSO collapse/normalize newlines before the object is ever
# signed -- in which case earlier bats tests using this same construction
# never exercised the box_format/transacted.go newline-collapse bug at
# all (nothing to corrupt if the description was already newline-free).
# Uses `fsck -recompute` with the same DODDER_DEBUG_DIGEST instrumentation
# to inspect the freshly-committed object's digest bytes= directly.
[group('debug')]
debug-verify-cli-description-has-real-newlines:
  #!/usr/bin/env bash
  set -euo pipefail
  bin=$(nix build --no-link --print-out-paths .#dodder-debug)
  export PATH="$bin/bin:$PATH"

  dest="{{ justfile_directory() }}/.tmp/debug-cli-description-check/$(date +%s)"
  mkdir -p "$dest"
  cd "$dest"

  dodder init -encryption none .default

  dodder new -edit=false -object-id sanity-check-tag \
    -description "$(printf 'paragraph one text here.\n\nparagraph two text here.')"

  debug_log="{{ justfile_directory() }}/.tmp/debug-cli-description-check.log"
  DODDER_DEBUG_DIGEST_ALL=1 DODDER_DEBUG_DIGEST_OBJECT_ID=sanity-check-tag \
    dodder fsck -recompute 'sanity-check-tag:e' 2>"$debug_log" || true

  echo "=== fsck output ==="
  cat "$debug_log" | grep -v DODDER_DEBUG || true

  echo ""
  echo "=== bytes= line(s) BEFORE export/reimport (look for how many separate 'Description ' keys appear) ==="
  grep 'bytes=' "$debug_log" || echo "(no bytes= lines found)"

  # Now round-trip through the ACTUAL export/import path (the real
  # box_format-archive inventory_list encode/decode this bug lives in) and
  # check whether the digest still matches on the other side.
  list_path="{{ justfile_directory() }}/.tmp/debug-cli-description-check-list.inventory_list"
  dodder export -print-time=true 'sanity-check-tag:e' >"$list_path"

  dest2="{{ justfile_directory() }}/.tmp/debug-cli-description-check-reimport/$(date +%s)"
  mkdir -p "$dest2"
  cd "$dest2"
  dodder init -encryption none -exclude-default-type .default

  debug_log2="{{ justfile_directory() }}/.tmp/debug-cli-description-check-reimport.log"
  DODDER_DEBUG_DIGEST_ALL=1 DODDER_DEBUG_DIGEST_OBJECT_ID=sanity-check-tag \
    dodder import "$list_path" 2>"$debug_log2" || true

  echo ""
  echo "=== import output ==="
  cat "$debug_log2" | grep -v DODDER_DEBUG || true

  echo ""
  echo "=== bytes= line(s) AFTER export/reimport ==="
  grep 'bytes=' "$debug_log2" || echo "(no bytes= lines found)"

# Export the real, live "default" repo's full history (all genres) to a
# static inventory-list file under .tmp/, then bootstrap a FRESH standalone
# repo from that file via `dodder import` -- pointing -blob_store-id at the
# real repo's actual madder blob store (~/.local/share/madder/blob_stores/default,
# confirmed via direct `ls` + reading its blob_store-config) so the fresh
# repo gets real blob content copied in, not shared/live storage. This
# gives a portable, static fixture to retest `debug-clone-default-into-tmp`
# against without needing live repo access each time, and a starting point
# for bisecting the inventory list to isolate which object(s) trigger the
# pull ed25519 signature-verification bug (dodder#TBD).
[group('debug')]
debug-export-default-and-bootstrap:
  #!/usr/bin/env bash
  set -euo pipefail
  bin=$(nix build --no-link --print-out-paths .#dodder-debug)
  export PATH="$bin/bin:$PATH"

  export_dir="{{ justfile_directory() }}/.tmp/debug-export-default"
  mkdir -p "$export_dir"
  list_path="$export_dir/$(date +%s).inventory_list"

  dodder export -repo_id default -print-time=true +z,e,t,k >"$list_path"

  echo "==> Exported to: $list_path"
  wc -l "$list_path"

  dest="{{ justfile_directory() }}/.tmp/debug-bootstrap-from-export/$(date +%s)"
  mkdir -p "$dest"
  cd "$dest"

  # -exclude-default-type: plain `init` otherwise always genesis-creates
  # its OWN !md type + blob (clone/init-workspace already set this
  # programmatically; this session added a CLI flag for `init` too, since
  # importing the real repo's own !md object-id on top of a second,
  # independently-genesis'd !md orphaned a blob mid config-recompile:
  # "Blob ... does not exist locally").
  dodder init -encryption none -exclude-default-type .default

  dodder import \
    -blob_store-id default \
    "$list_path"

  # Stable symlink so debug-pull-from-bootstrapped-export can find the
  # most recent bootstrap without the caller having to pass a timestamp.
  ln -sfn "$dest" "{{ justfile_directory() }}/.tmp/debug-bootstrap-from-export/latest"

  echo ""
  echo "==> Fresh bootstrapped repo: $dest"
  echo "==> Exported list:          $list_path"
  echo "==> Run:"
  echo "  cd $dest && dodder show :z | wc -l"

# Pull from the static, exported-and-bootstrapped fixture repo (built by
# `just debug-export-default-and-bootstrap`) instead of the live default
# repo -- isolates whether the pull ed25519 signature-verification bug
# reproduces against a STANDALONE copy of the real data (no live repo
# access, no shared blob store with the operator's actual notes), or
# whether it needs something specific to the live repo/session that a
# static export+import round-trip does not carry over.
[group('debug')]
debug-pull-from-bootstrapped-export:
  #!/usr/bin/env bash
  set -euo pipefail
  bin=$(nix build --no-link --print-out-paths .#dodder-debug)
  export PATH="$bin/bin:$PATH"

  source="{{ justfile_directory() }}/.tmp/debug-bootstrap-from-export/latest"

  if [[ ! -d "$source/.dodder" ]]; then
    echo "no bootstrapped export repo found at $source -- run just debug-export-default-and-bootstrap first" >&2
    exit 1
  fi

  dest="{{ justfile_directory() }}/.tmp/debug-pull-from-export/$(date +%s)"
  mkdir -p "$dest"
  cd "$dest"

  dodder init -encryption none -exclude-default-type .default
  dodder pull -direct "$(realpath "$source")" +z,e,t,k

  echo ""
  echo "==> Source (bootstrapped export): $source"
  echo "==> Destination:                  $dest"

# Isolate whether the pull ed25519 signature-verification bug is triggered
# by the HOME-REPO PARENT-RESOLUTION MECHANISM itself (init-workspace's
# ParentPath -> ResolveImplicitDirectPath -> VerifyOrPinParent's pubkey
# pin/verify check, go/internal/tango/command_components_dodder/remote.go:120-159)
# rather than by anything data-dependent. debug-pull-from-bootstrapped-export
# uses plain `pull -direct`, which VerifyOrPinParent treats as a no-op
# (only a parent-RESOLVED remote is subject to the #287b pubkey check).
# This recipe instead runs `init-workspace -parent <bootstrapped-export>`
# so the workspace records a REAL ParentPath, then a plain `pull` (no
# -direct) against the SAME static, already-proven-clean data -- putting
# the parent-resolution/pin-verification code path under test while
# holding the data constant.
[group('debug')]
debug-pull-via-parent-resolution-from-bootstrapped-export:
  #!/usr/bin/env bash
  set -euo pipefail
  bin=$(nix build --no-link --print-out-paths .#dodder-debug)
  export PATH="$bin/bin:$PATH"

  source="{{ justfile_directory() }}/.tmp/debug-bootstrap-from-export/latest"

  if [[ ! -d "$source/.dodder" ]]; then
    echo "no bootstrapped export repo found at $source -- run just debug-export-default-and-bootstrap first" >&2
    exit 1
  fi

  dest="{{ justfile_directory() }}/.tmp/debug-pull-via-parent/$(date +%s)"
  mkdir -p "$dest"
  cd "$dest"

  dodder init-workspace -parent "$(realpath "$source")" debug-ws
  dodder pull +z,e,t,k

  echo ""
  echo "==> Source (bootstrapped export, as PARENT): $source"
  echo "==> Destination:                             $dest"

# Last isolation step: does the pull ed25519 signature-verification bug
# need the HOME-REPO resolution mechanism specifically (MakeHomeRepoRemote,
# go/internal/tango/command_components_dodder/remote.go:230-268), as
# opposed to -direct (ruled out: debug-pull-from-bootstrapped-export) or
# parent-path resolution with pubkey pin/verify (ruled out:
# debug-pull-via-parent-resolution-from-bootstrapped-export)?
# MakeHomeRepoRemote resolves via `os.UserHomeDir()` + the STANDARD (non-
# override) XDG template ($HOME/.local/share/$XDG_UTILITY_NAME/repos/<name>)
# -- NOT $XDG_DATA_HOME directly. So this recipe builds a FAKE $HOME
# containing a `default` repo (plain name, not CWD-scoped) bootstrapped
# from the SAME exported inventory list used by the other debug-pull-*
# recipes, then runs `init-workspace` with NO -parent (so parentIsHomeRepo
# resolves true) + `pull` with HOME overridden to the fake one --
# reproducing MakeHomeRepoRemote's exact resolution against clean,
# already-proven-clean-elsewhere static data. If this ALSO succeeds
# cleanly, the bug needs something about the LIVE repo's actual runtime
# state (not just its data, and not this resolution mechanism in the
# abstract); if it reproduces the failure, MakeHomeRepoRemote itself is
# implicated regardless of data.
[group('debug')]
debug-pull-via-home-repo-resolution-from-bootstrapped-export:
  #!/usr/bin/env bash
  set -euo pipefail
  bin=$(nix build --no-link --print-out-paths .#dodder-debug)
  export PATH="$bin/bin:$PATH"

  export_dir="{{ justfile_directory() }}/.tmp/debug-export-default"
  list_path="$(ls -t "$export_dir"/*.inventory_list 2>/dev/null | head -1)"

  if [[ -z "$list_path" ]]; then
    echo "no exported inventory list found in $export_dir -- run just debug-export-default-and-bootstrap first" >&2
    exit 1
  fi

  # A fake $HOME: init-workspace's home-repo branch resolves purely via
  # os.UserHomeDir() + the standard (non-override) XDG template
  # ($HOME/.local/share/$XDG_UTILITY_NAME/repos/<name>, confirmed earlier
  # this session by reading dewey's XDG template source directly). XDG_*
  # vars must ALSO be unset: per the XDG spec (and this session's own real
  # XDG_CONFIG_HOME etc.) an explicitly set XDG_CONFIG_HOME takes
  # precedence over $HOME-derived defaults, so leaving it set would
  # collide with this real session's actual ~/.config/dodder state.
  fake_home="{{ justfile_directory() }}/.tmp/debug-fake-home/$(date +%s)"
  mkdir -p "$fake_home/.local/share/dodder" "$fake_home/.local/share/madder"
  export HOME="$fake_home"
  unset XDG_DATA_HOME XDG_CONFIG_HOME XDG_STATE_HOME XDG_CACHE_HOME XDG_RUNTIME_HOME

  # `-blob_store-id default`'s flag type is a scoped_id, not a path -- it
  # rejects an arbitrary filesystem path outright, so it cannot reference
  # the real blob store directly once HOME is overridden. Instead,
  # symlink the fake home's expected blob-store location AT the real
  # one: madder's own repos/default resolution (independent of dodder's)
  # then finds the SAME real, read-only blob content under the fake HOME,
  # with no copying and no separate store.
  mkdir -p "$fake_home/.local/share/madder/blob_stores"
  ln -s "/Users/sfriedenberg/.local/share/madder/blob_stores/default" \
    "$fake_home/.local/share/madder/blob_stores/default"

  (
    mkdir -p "$fake_home/work"
    cd "$fake_home/work"
    dodder init -encryption none -exclude-default-type default
    dodder import -blob_store-id default "$list_path"
  )

  dest="{{ justfile_directory() }}/.tmp/debug-pull-via-home-repo/$(date +%s)"
  mkdir -p "$dest"
  cd "$dest"

  # No -parent: parentPath is empty, so ResolveImplicitDirectPath falls
  # through to the parentIsHomeRepo branch (remote.go:134-140) instead of
  # the explicit-path branch tested by the other two debug-pull-* recipes.
  dodder init-workspace debug-ws
  dodder pull +z,e,t,k

  echo ""
  echo "==> Fake HOME:      $fake_home"
  echo "==> Destination:    $dest"

# Verify Go's actual %q (strconv.Quote) escaping behavior for printable
# non-ASCII runes (em-dash, smart quotes) -- ground-truth check for the
# ed25519 pull-verification bug theory: box_format writes descriptions with
# %q (go/internal/alfa/string_format_writer/fields_writer.go:237) but
# doddish's scanner (go/internal/0/doddish/scanner.go
# consumeLiteralOrFieldValue) only unescapes a leading backslash before ANY
# character -- it has no strconv.Quote-compatible handling of \uXXXX
# sequences. If %q escapes em-dash/smart-quotes as \uXXXX (rather than
# printing them raw), doddish's scanner would misparse them, corrupting the
# description on any read path that goes through doddish (inventory_list
# decode) but not on paths that store raw bytes directly (stream_index
# binary). Uses `go run` directly (not `just build`) since this is
# stdlib-only and doesn't need dodder's own build.
[group('debug')]
debug-verify-go-percent-q-escaping:
  #!/usr/bin/env bash
  set -euo pipefail
  cd "{{ justfile_directory() }}/go"
  go run "{{ justfile_directory() }}/.tmp/debug-quote-roundtrip/main.go"

# Clean-room check of the exact pubkey/digest/sig bytes captured via
# DODDER_DEBUG_VERIFY instrumentation (object_finalizer/validation.go) for
# project-catapult_lag's FAILING verify during a live pull -- runs Go's
# stdlib crypto/ed25519.Verify directly against those bytes, completely
# outside dodder's own code, to determine whether the signature genuinely
# doesn't verify (real crypto mismatch) or whether dodder's own
# markl/piggy verify wrapper has a bug (would succeed here but fail in
# dodder). Also sanity-checks pubkey/sig byte lengths against
# ed25519.PublicKeySize/SignatureSize.
[group('debug')]
debug-verify-ed25519-bytes-standalone:
  #!/usr/bin/env bash
  set -euo pipefail
  cd "{{ justfile_directory() }}/go"
  go run "{{ justfile_directory() }}/.tmp/debug-quote-roundtrip/verify_check.go"

# Compare the inventory_list-format (full signature-bearing) representation
# of project-catapult_lag:e between the real default repo and der (debug aid
# for the pull-import ed25519 signature-verification failure -- fsck on the
# real repo passes clean, but pull's importer rejects this exact object's
# signature every time; comparing the two representations should show
# whether the import path re-serializes the object differently before
# re-verifying its signature).
[group('debug')]
debug-compare-catapult-lag-sig:
  #!/usr/bin/env bash
  set -euo pipefail
  bin=$(nix build --no-link --print-out-paths .#dodder-debug)
  export PATH="$bin/bin:$PATH"

  echo "=== dodder show -repo_id default -format inventory_list project-catapult_lag:e ==="
  dodder show -repo_id default -format inventory_list 'project-catapult_lag:e'

  echo ""
  echo "=== der show -repo_id default -format inventory_list project-catapult_lag:e ==="
  der show -repo_id default -format inventory_list 'project-catapult_lag:e'

  echo ""
  echo "=== dodder show -repo_id default -format text project-catapult_lag:e (cat -A, raw bytes) ==="
  dodder show -repo_id default -format text 'project-catapult_lag:e' | cat -A

live_workspace := env("HOME") / "workspaces/dodder-haustoria-caldav/workspace"

# Init a CWD-scoped dodder repo and wire `.claude/mcp.json` so this
# session's Claude Code picks up `dodder mcp` as an MCP server.
# Refuses to overwrite an existing .dodder/ in $PWD — clear it first
# if you want to re-init.
[group('explore')]
explore-mcp-init:
  #!/usr/bin/env bash
  set -euo pipefail
  # `.madder/` is the session-local blob store and is meant to persist
  # across re-inits; only `.dodder/` blocks a fresh init.
  if [[ -e .dodder ]]; then
    echo "refusing to re-init: .dodder/ already in $(pwd)"
    echo "  rm -rf .dodder .claude/mcp.json"
    echo "to start fresh."
    exit 1
  fi

  # Build into the `result` symlink (default for `nix build` without
  # --no-link) so `.mcp.json` can point at `./result/bin/dodder` and
  # pick up future rebuilds via the symlink refresh — no need to edit
  # .mcp.json each time.
  nix build .#dodder
  export PATH="$(pwd)/result/bin:$PATH"

  # Heredocs in justfile recipes inherit the recipe's leading indent,
  # which would land inside the file content. Use printf + explicit
  # newlines to keep wordlists and JSON clean.
  mkdir -p .tmp
  printf '%s\n' alpha bravo charlie delta echo foxtrot golf hotel india juliett > .tmp/yin
  printf '%s\n' kilo lima mike november oscar papa quebec romeo sierra tango > .tmp/yang

  # Clear any leftover session-local default store; an existing
  # blob_store-config there blocks init with "file exists".
  rm -rf .madder/local/share/blob_stores/default

  dodder init \
    -yin .tmp/yin \
    -yang .tmp/yang \
    -repo_id . \
    -encryption none \
    test-mcp

  # Claude Code looks for `.mcp.json` at the project root, not under
  # `.claude/`. Pointing at the `./result/bin/dodder` symlink means
  # rebuilds (`nix build .#dodder` or just re-running this recipe) are
  # picked up automatically on the next MCP reconnect — no .mcp.json
  # edit needed.
  printf '%s\n' \
    '{' \
    '  "mcpServers": {' \
    '    "dodder": {' \
    '      "command": "./result/bin/dodder",' \
    '      "args": ["mcp"]' \
    '    }' \
    '  }' \
    '}' \
    > .mcp.json

  echo ""
  echo "==> Repo:        $(pwd)/.dodder"
  echo "==> MCP config:  $(pwd)/.mcp.json"
  echo "==> Binary:      $(pwd)/result/bin/dodder"
  echo ""
  echo "Restart Claude Code (or reload MCP servers) to pick up the dodder MCP."
  echo "After a code change, re-run \`nix build .#dodder\` (refreshes the"
  echo "symlink) and then reconnect; .mcp.json doesn't need editing."
  echo "Cleanup: rm -rf .dodder .mcp.json result .tmp/yin .tmp/yang"

# Run a dodder command in the live CalDAV workspace.
[group('explore')]
explore-live *args:
  #!/usr/bin/env bash
  set -euo pipefail
  source "$HOME/.secrets.env"
  bin=$(nix build --no-link --print-out-paths .#dodder-debug)
  export PATH="$bin/bin:$PATH"
  cd "{{live_workspace}}"
  dodder {{args}}

nvim_explore_dir := "/tmp/dodder-nvim-explore"

# Shared prerequisite for the explore-nvim-* recipes below.
_require-nvim:
  #!/usr/bin/env bash
  command -v nvim >/dev/null || { echo "nvim not found on PATH"; exit 1; }

# Set up a throwaway dodder repo + workspace at /tmp/dodder-nvim-explore/
# with a couple of example zettels, for the other explore-nvim-* recipes to
# open in Neovim. Re-run any time to reset back to a clean slate.
[group('explore')]
explore-nvim-init:
  #!/usr/bin/env bash
  set -euo pipefail
  bin=$(nix build --no-link --print-out-paths .#dodder-debug)
  export PATH="$bin/bin:$PATH"

  rm -rf "{{nvim_explore_dir}}"
  mkdir -p "{{nvim_explore_dir}}"
  cd "{{nvim_explore_dir}}"

  dodder init \
    -yin <(echo -e "aleph\nbeth\ngimel") \
    -yang <(echo -e "one\ntwo\nthree") \
    -encryption none \
    .default
  # Lightweight (non-repo-backed) workspace: this is a single throwaway
  # repo with no parent to link, so the repo-backed path's extra wiring
  # isn't needed here.
  dodder init-workspace -experimental-repo=false

  printf '%s\n' \
    '---' \
    '# First example zettel for dodder.nvim' \
    '- project' \
    '- demo' \
    '! md' \
    '---' \
    '' \
    '# Hello' \
    '' \
    'This is a **markdown** body, injected via `vim-syntax-type`.' \
    | dodder new -edit=false -

  printf '%s\n' \
    '---' \
    '# Second example zettel, with a list' \
    '- reading-list' \
    '! md' \
    '---' \
    '' \
    'Body-language injection resolves per object, live, via the dodder CLI:' \
    '' \
    '- one' \
    '- two' \
    | dodder new -edit=false -

  dodder checkout :z

  echo ""
  echo "==> Explore dir: {{nvim_explore_dir}}"
  echo "==> Next, try:"
  echo "  just explore-nvim-hyphence     # hyphence highlighting + live body injection"
  echo "  just explore-nvim-workspace    # forced-toml injection on .dodder-workspace"
  echo "  just explore-nvim-organize     # organize-text highlighting"
  echo "  just explore-nvim-doddish      # doddish query language highlighting"
  echo "  just explore-nvim-checkhealth  # :checkhealth dodder"

# Open the example zettels from explore-nvim-init in Neovim with dodder.nvim
# loaded, demonstrating hyphence highlighting and live body-language
# injection (the dodder CLI is on PATH, so injection.lua's async resolver
# round-trips for real).
[group('explore')]
explore-nvim-hyphence: _require-nvim
  #!/usr/bin/env bash
  set -euo pipefail
  dodder_bin=$(nix build --no-link --print-out-paths .#dodder-debug)
  plugin=$(nix build --no-link --print-out-paths .#dodder-nvim)
  export PATH="$dodder_bin/bin:$PATH"

  cd "{{nvim_explore_dir}}" 2>/dev/null || { echo "run 'just explore-nvim-init' first"; exit 1; }
  files=$(find . -name '*.zettel' | sort)
  [[ -n "$files" ]] || { echo "no .zettel files found; run 'just explore-nvim-init' first"; exit 1; }

  # shellcheck disable=SC2086
  nvim --clean -p $files \
    --cmd "set rtp+=$plugin" \
    -c "lua require('dodder').setup()"

# Open the explore workspace's .dodder-workspace file in Neovim, showing the
# injection path that forces the body language to TOML unconditionally (no
# dodder CLI round-trip needed for this one).
[group('explore')]
explore-nvim-workspace: _require-nvim
  #!/usr/bin/env bash
  set -euo pipefail
  plugin=$(nix build --no-link --print-out-paths .#dodder-nvim)

  cd "{{nvim_explore_dir}}" 2>/dev/null || { echo "run 'just explore-nvim-init' first"; exit 1; }
  [[ -f .dodder-workspace ]] || { echo ".dodder-workspace not found; run 'just explore-nvim-init' first"; exit 1; }

  nvim --clean .dodder-workspace \
    --cmd "set rtp+=$plugin" \
    -c "lua require('dodder').setup()"

# Run `dodder organize` for real against the explore workspace and open the
# resulting organize-text buffer in Neovim.
[group('explore')]
explore-nvim-organize: _require-nvim
  #!/usr/bin/env bash
  set -euo pipefail
  dodder_bin=$(nix build --no-link --print-out-paths .#dodder-debug)
  plugin=$(nix build --no-link --print-out-paths .#dodder-nvim)
  export PATH="$dodder_bin/bin:$PATH"

  cd "{{nvim_explore_dir}}" 2>/dev/null || { echo "run 'just explore-nvim-init' first"; exit 1; }
  [[ -d .dodder ]] || { echo "no .dodder store found; run 'just explore-nvim-init' first"; exit 1; }

  dodder organize -mode output-only :z >example.organize

  nvim --clean example.organize \
    --cmd "set rtp+=$plugin" \
    -c "lua require('dodder').setup()" \
    -c "set filetype=dodder-organize"

# Open the standalone doddish query-language example in Neovim. No dodder
# repo or workspace needed -- doddish highlighting is purely syntactic.
[group('explore')]
explore-nvim-doddish: _require-nvim
  #!/usr/bin/env bash
  set -euo pipefail
  plugin=$(nix build --no-link --print-out-paths .#dodder-nvim)

  nvim --clean zz-nvim/examples/example.doddish \
    --cmd "set rtp+=$plugin" \
    -c "lua require('dodder').setup()" \
    -c "set filetype=doddish"

# Open :checkhealth dodder in a scratch Neovim session -- verifies the
# dodder binary, the three compiled parsers, and the shipped queries all
# resolve. Good first check before the other explore-nvim-* recipes.
[group('explore')]
explore-nvim-checkhealth: _require-nvim
  #!/usr/bin/env bash
  set -euo pipefail
  dodder_bin=$(nix build --no-link --print-out-paths .#dodder-debug)
  export PATH="$dodder_bin/bin:$PATH"
  plugin=$(nix build --no-link --print-out-paths .#dodder-nvim)

  nvim --clean \
    --cmd "set rtp+=$plugin" \
    -c "lua require('dodder').setup()" \
    -c "checkhealth dodder"

# Trace madder/dodder functions matching {{func_regexp}} while running
# `dodder show {{query}}`, dumping a {{stack}}-deep call stack at each
# hit. Builds the DWARF-retaining debug binary via nix (the standard
# debug binary is stripped, which dlv can't trace). Runs from the
# worktree root so CWD-store resolution matches an interactive
# `dodder show`. Investigation aid for the multi-store eager-init
# question; delete once root cause found.
#
# Examples:
#   just explore-dlv-trace 'remoteSftp.*\.initialize$'   # who dials SFTP
#   just explore-dlv-trace '\.HasBlob$' 0                # probe order/results
[group('explore')]
explore-dlv-trace func_regexp stack='80' query='ach/ab':
  #!/usr/bin/env bash
  set -euo pipefail
  bin=$(nix build --no-link --print-out-paths .#dodder-debug-dwarf)
  dlv trace --exec "$bin/bin/dodder" --stack {{stack}} '{{func_regexp}}' -- show {{query}}

# Debug a specific bats test file with --no-tempdir-cleanup for inspection.
[group('explore')]
explore-bats-debug *targets:
  #!/usr/bin/env bash
  set -euo pipefail
  bin=$(nix build --no-link --print-out-paths .#dodder-debug)
  export PATH="$bin/bin:$PATH"
  GOMEMLIMIT=512MiB \
    DODDER_CEILING_DIRECTORIES="{{bats_ceiling}}" \
    MADDER_CEILING_DIRECTORIES="{{bats_ceiling}}" \
    just zz-tests_bats/test-targets --no-tempdir-cleanup {{targets}}

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
  msg_file=$(mktemp)
  trap 'rm -f "$msg_file"' EXIT
  cat >"$msg_file" <<'TAGMSG'
  {{message}}
  TAGMSG
  git tag -s -F "$msg_file" "$tag"
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

# Duplicate-registration runtime proof for the madder#255 cutover. A fresh
# `dodder init` (and any other command) would panic at init() on any duplicate
# RegisterPurpose / RegisterPurposeIdAlias between dodder's, madder's, and
# piggy's registration packages; `dodder gen` additionally mints madder-*
# purposes (proving dodder inherits them from madder's now-madder-only
# registrations). Runs entirely in a throwaway temp dir with CWD-scoped repos.
[group('debug')]
debug-cutover-smoke:
  #!/usr/bin/env bash
  set -euo pipefail
  bin=$(nix build --no-link --print-out-paths .#dodder-debug)
  export PATH="$bin/bin:$PATH"
  base=$(mktemp -d)
  trap 'rm -rf "$base"' EXIT
  export DODDER_CEILING_DIRECTORIES="$base"
  export MADDER_CEILING_DIRECTORIES="$base"
  printf '%s\n' alpha bravo charlie delta echo foxtrot golf hotel india juliett > "$base/yin"
  printf '%s\n' kilo lima mike november oscar papa quebec romeo sierra tango > "$base/yang"

  echo "==> fresh init (encryption none)"
  mkdir "$base/plain"
  (cd "$base/plain" && dodder init -yin "$base/yin" -yang "$base/yang" -encryption none .default)

  echo "==> fresh init (default age encryption)"
  mkdir "$base/age"
  (cd "$base/age" && dodder init -yin "$base/yin" -yang "$base/yang" .default)

  echo "==> gen: madder-* (inherited) and dodder-* (own) private-key purposes"
  dodder gen madder-private_key-v0 madder-private_key-v1 dodder-repo-private_key-v1

  echo "==> cutover smoke OK"

# Import the live default repo's export into the disposable
# dodder-migration-staging repo (part of the dodder-index-test /
# resolve-tai-reassign investigation, see chat with krusty for context).
# The automated run of this same import hung silently for 18+ min against
# rsync_dot_net (2 !bookmark objects there are remote-only) with no visible
# progress from outside the process — run this interactively instead so the
# per-object commit lines and blob_store status lines stream live.
[group('debug')]
debug-import-staging-baseline:
  #!/usr/bin/env bash
  set -euo pipefail
  bin=$(nix build --no-link --print-out-paths .#dodder-debug)
  export PATH="$bin/bin:$PATH"
  sp=/home/sasha/workspaces/dodder-migration-staging
  dodder import -verbose -repo_id dodder-migration-staging -plan-format summary "$sp/default-repo.export.inventory_list"
  echo "==> done, staging repo object count:"
  dodder show -repo_id dodder-migration-staging '+?z,t,k,e' | wc -l

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

# As debug-test-bats-sftp, but builds dodder-debug against a locally
# checked-out (and possibly hand-patched) madder source tree instead of
# the pinned flake.lock rev. Used for adding temporary diagnostic
# fmt.Fprintf(os.Stderr, ...) instrumentation directly into madder's
# blob store code and observing it live against the exact same
# single-hash SFTP repro as blob_store_sftp_single_hash.bats, without
# needing to file/push/re-bump anything first. madder_path defaults to
# the scratch checkout used for this session's SFTP mover investigation
# (task #21) -- point it elsewhere for unrelated debugging.
[group('debug')]
debug-test-bats-sftp-madder-override madder_path=".tmp/madder-debug-checkout" *targets:
  #!/usr/bin/env bash
  set -euo pipefail
  bin=$(nix build --no-link --print-out-paths .#dodder-debug --override-input madder "path:$(realpath '{{madder_path}}')")
  madder_bin=$(nix build --no-link --print-out-paths .#madder-bin)
  sftp_bin=$(nix build --no-link --print-out-paths .#madder-test-sftp-server)
  export PATH="$bin/bin:$madder_bin/bin:$PATH"
  GOMEMLIMIT=512MiB \
    MADDER_TEST_SFTP_SERVER="$sftp_bin/bin/madder-test-sftp-server" \
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
# blocks the loopback bind even with the binary present.
[group('debug')]
debug-test-bats-sftp *targets:
  #!/usr/bin/env bash
  set -euo pipefail
  bin=$(nix build --no-link --print-out-paths .#dodder-debug)
  sftp_bin=$(nix build --no-link --print-out-paths .#madder-test-sftp-server)
  export PATH="$bin/bin:$PATH"
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

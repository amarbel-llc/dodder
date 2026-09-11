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

lint: lint-fmt lint-shell lint-grammar

# Read-only formatting gate (conformist check via the flake's
# checks.formatting). Runs first in the default lane so a formatting drift
# fails fast, before the expensive build/test. This is what the spinclass
# pre-merge hook (bare `just`) enforces.
#
# check formatting read-only via the flake's checks.formatting
lint-fmt:
  just go/check-conformist

# read-only shellcheck gate for tracked shell scripts (see go/check-shellcheck)
lint-shell:
  just go/check-shellcheck

# Read-only grammar-validation + codegen-drift gate for orgmode.peg
# (dodder#378, see go/check-grammar and go/check-grammar-drift). NOTE:
# `go/check` (vuln/vet/repool/seqerror/etc, the `check` recipe below)
# is a SEPARATE, non-gating aggregate that this pre-merge `lint` lane
# never calls -- these two recipes need their own explicit wiring here
# to actually be part of what `just`/the pre-merge hook enforces,
# same as lint-fmt/lint-shell above.
lint-grammar:
  just go/check-grammar
  just go/check-grammar-drift

#   ____        _ _     _
#  | __ ) _   _(_) | __| |
#  |  _ \| | | | | |/ _` |
#  | |_) | |_| | | | (_| |
#  |____/ \__,_|_|_|\__,_|
#

build: build-go

# Codegen + formatting via the go/ justfile — the pre-merge gate's build step
# (a debug binary for ad-hoc dev use comes from `just go/build-go-binary`; the
# release artifact is the flake's default package).
#
# run codegen and formatting via the go/ justfile
build-go:
  just go/build-go

# Regenerate the dodder.net seed-set type files (FDR-0010 Phase 3) into
# zz-seed/types/ from the table in go/cmd/dodder-gen_seed_types/table.go.
# Deterministic and idempotent (stable ordering, no timestamps; stale files
# pruned) — review the diff and commit. Agent dev-loop: seed-set table edits.
#
# regenerate the dodder.net seed-set type files into zz-seed/types/
generate-seed-types:
  cd go && go run ./cmd/dodder-gen_seed_types -dir ../zz-seed/types

#    ____ _               _
#   / ___| |__   ___  ___| | __
#  | |   | '_ \ / _ \/ __| |/ /
#  | |___| | | |  __/ (__|   <
#   \____|_| |_|\___|\___|_|\_\
#

# run all static checks: vuln, vet, repool, seqerror
check:
  just go/check

#   _____         _
#  |_   _|__  ___| |_
#    | |/ _ \/ __| __|
#    | |  __/\__ \ |_
#    |_|\___||___/\__|
#

# NB: the bats lane builds dodder internally inside the nix sandbox, so no
# `build` dep is needed for the test aggregate below. (Detached comment on
# purpose: aggregates carry no doc comment — conformist-justfile(7).)

test: test-go test-grammar test-bats test-bats-generate

# run unit tests only
test-go *flags:
  just go/test-go-unit {{flags}}

# Runs the grammar-vectors harness with a hermetically-resolved
# langlang (dodder#378, see go/test-grammar-vectors). Separate from
# test-go since it needs its own nix-resolved binary, not just the
# devshell's `go test`.
test-grammar:
  just go/test-grammar-vectors

# Run the full bats integration suite inside the nix sandbox.
# Wraps `pkgs.testers.batsLane` from amarbel-llc/bats; binaries are
# injected via the lane's binaries map (DODDER_BIN, MADDER_BIN, …).
# No fixture-generation step: fixtures are committed under
# zz-tests_bats/previous_versions/v*/ and the lane consumes them
# directly. To regenerate, run `just test-bats-update-fixtures`.
#
# run the full bats integration suite inside the nix sandbox
test-bats:
  nix build .#bats-default --no-link --print-build-logs

# Smoke-test the fixture generator: build the `fixtures-current`
# derivation (which runs previous_versions/generate_fixture.bats in the
# sandbox) without materializing it. The generator is excluded from the
# normal bats lanes, so this is the only thing that runs it in CI — a
# store/CLI change that breaks generation (e.g. FDR-0020 making config
# non-queryable) fails here instead of silently bitrotting until the next
# manual `test-bats-update-fixtures`. See #272.
#
# build the fixtures-current derivation to smoke-test the fixture generator
test-bats-generate:
  nix build .#fixtures-current --no-link --print-build-logs

# Run a per-tag bats lane (e.g. `just test-bats-tags haustoria`).
# The tag is the file_tags value from `# bats file_tags=foo`
# directives. Tags with embedded `:` are quoted into the flake
# attribute path automatically.
#
# run a per-tag bats lane
test-bats-tags *tags:
  nix build '.#bats-{{tags}}' --no-link --print-build-logs

# Run a single bats test file (or list of files) via the legacy
# batman path. Kept because the nix lane builder operates at the
# tag level — there is no per-file nix lane to dispatch a single
# `show.bats` against. For tag-level filtering, use
# `test-bats-tags <tag>` (faster, hermetic) instead. The debug-tagged
# dodder binary is resolved through the flake so this recipe doesn't
# need `just build` first.
#
# run a single bats test file (or list of files) via the legacy batman path
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
#
# run bats targets via the batman path with the bats sandbox disabled
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
#
# run bats targets with MADDER_TEST_SFTP_SERVER built and exported
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

# run bats with race-instrumented binary to detect data races in pool reuse
test-bats-race:
  just go/test-bats-race

# Force-regenerate fixtures inside the nix sandbox, then materialize
# the result into the worktree. Review the diff and commit.
#
# force-regenerate bats fixtures and materialize them into the worktree
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
#
# regenerate golden files for approval-testing assertions
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
#
# snapshot the current bats test suite as a previous_versions/ entry
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
#
# start a local Radicale CalDAV server for haustoria prototyping
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
#
# create a haustoria workspace pointing at local Radicale
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

# show haustoria status for the explore workspace
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
#
# init a CWD-scoped dodder repo and wire .mcp.json for `dodder mcp`
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

# run a dodder command in the live CalDAV workspace
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
#
# set up a throwaway dodder repo + workspace for the explore-nvim-* recipes
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
#
# open the example zettels in Neovim with dodder.nvim loaded
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
#
# open the explore workspace's .dodder-workspace file in Neovim
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
#
# open a real organize-text buffer from the explore workspace in Neovim
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
#
# open the standalone doddish query-language example in Neovim
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
#
# run `:checkhealth dodder` in a scratch Neovim session
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
#
# trace madder/dodder functions with dlv while running `dodder show`
[group('explore')]
explore-dlv-trace func_regexp stack='80' query='ach/ab':
  #!/usr/bin/env bash
  set -euo pipefail
  bin=$(nix build --no-link --print-out-paths .#dodder-debug-dwarf)
  dlv trace --exec "$bin/bin/dodder" --stack {{stack}} '{{func_regexp}}' -- show {{query}}

# debug a specific bats test file with --no-tempdir-cleanup for inspection
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
#
# tag a Go module release
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

# Sed-rewrite DODDER_VERSION in version.env to the given semver
# (eng-versioning(7)). The version string is burnt into the binary at
# build time via -ldflags (see go/cmd/*/main.go and
# go/internal/uniform/commands_dodder/version.go), flake.nix reads
# version.env as the single source of truth. No-op if already at the
# target version. Usage: just bump-version 0.1.1
#
# rewrite DODDER_VERSION in version.env to the given semver
[group('release')]
bump-version new_version:
  #!/usr/bin/env bash
  set -euo pipefail
  current=$(grep '^export DODDER_VERSION=' version.env | cut -d= -f2)
  if [[ "$current" == "{{new_version}}" ]]; then
    gum log --level info "already at {{new_version}}"
    exit 0
  fi
  sed -E -i "s/^(export DODDER_VERSION)=.*/\1={{new_version}}/" version.env
  gum log --level info "bumped DODDER_VERSION: $current → {{new_version}}"

# Cut a release: must be run on master. Bumps DODDER_VERSION in
# version.env, commits the bump with a changelog-style message built
# from commits since the last go/v* tag, pushes master, then signs
# and pushes the go/v{{version}} tag. The "go/v" prefix is added for
# you, so pass the semver without it. Usage: just release 0.1.1
#
# Use `just tag <version> <message>` directly if you want to
# control the commit message yourself without bumping.
#
# cut a release: bump version.env, commit, push master, then sign and push the tag
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
  if ! git diff --quiet version.env; then
    git add version.env
    git commit -m "chore: release go/v{{version}}"
    git push origin master
    gum log --level info "pushed version.env bump to master"
  fi
  just tag "{{version}}" "$msg"

# Duplicate-registration runtime proof for the madder#255 cutover. A fresh
# `dodder init` (and any other command) would panic at init() on any duplicate
# RegisterPurpose / RegisterPurposeIdAlias between dodder's, madder's, and
# piggy's registration packages; `dodder gen` additionally mints madder-*
# purposes (proving dodder inherits them from madder's now-madder-only
# registrations). Runs entirely in a throwaway temp dir with CWD-scoped repos.
#
# run the duplicate-registration runtime proof for the madder#255 cutover
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
#
# Phase 1 of the take4 consolidation (execution plan:
# ~/.claude/plans/bright-soaring-fiddle.md, top section): export the live
# default repo's full history via BOTH the index path (`export`) and the
# log path (`export-inventory_lists`, dodder board task #8) into the
# take4 work dir, then print per-file object-line counts. The two-path
# comparison is a free index-health check on the consolidation's primary
# source: the log path is authoritative, so an index-path shortfall means
# the index is silently incomplete. Read-only against the live repo.
[group('consolidate')]
consolidate-export-baseline:
  #!/usr/bin/env bash
  set -euo pipefail
  bin=$(nix build --no-link --print-out-paths .#dodder-debug)
  export PATH="$bin/bin:$PATH"
  work=/home/sasha/workspaces/take4/exports
  mkdir -p "$work"
  # -repo_id default on every invocation: the worktree contains a stray
  # explore-recipe .dodder/, and the cwd walk-up would silently resolve
  # to that toy repo instead of the live XDG-user default. An explicit
  # XDG-user bare name is scope-pinned (FDR-0019 / #294) and immune to
  # the override. The identity header makes the output self-identify
  # which repo was actually exported.
  echo "==> exporting from repo:"
  dodder info-repo -repo_id default pubkey instance-id store-version
  echo "==> index-path export: dodder export '+?z,t,k,e'"
  dodder export -repo_id default '+?z,t,k,e' > "$work/default.index-path.inventory_list"
  echo "==> log-path export (list objects only)"
  dodder export-inventory_lists -repo_id default > "$work/default.log-path.lists-only.inventory_list"
  echo "==> log-path export (contents; tolerant — the live repo has at least one missing list blob, take4 pathology P9)"
  dodder export-inventory_lists -repo_id default -contents -tolerate-missing-blobs > "$work/default.log-path.contents.inventory_list"
  echo "==> raw index dump (below the query layer; the P9 sole-carrier census)"
  dodder export-index -repo_id default > "$work/default.index-raw.inventory_list"
  echo "==> object-line counts ('[' lines; log-contents includes the :b list objects themselves, the index path does not)"
  for f in "$work"/default.*.inventory_list; do
    printf '%8d %s\n' "$(grep -c '^\[' "$f" || true)" "$f"
  done

# Checksum the migration2 workbench export files so the take4 union input
# manifest can exclude content-duplicate files and the known-empty one.
# CONTENT-normalized: sums only the object lines (leading '['), because
# the hyphence metadata header can differ between exports of identical
# content and would make a raw byte sum diverge spuriously. Read-only;
# consolidation Phase 1.
[group('consolidate')]
consolidate-checksum-workbench:
  #!/usr/bin/env bash
  set -euo pipefail
  cd /home/sasha/workspaces/dodder_migration2
  for f in ./*.inventory_list ./*.inventory_list-v2 dodder-val/*.inventory_list; do
    # (grep || true): an empty/object-less file is data, not an error.
    # Two sums: ordered (byte content of object lines) and sorted
    # (order-insensitive set identity) — same sorted sum with different
    # ordered sums means a reordered export of the same object set.
    ordered=$( (grep '^\[' "$f" || true) | b2sum | cut -d' ' -f1)
    sorted=$( (grep '^\[' "$f" || true) | sort | b2sum | cut -d' ' -f1)
    count=$( (grep -c '^\[' "$f" || true) )
    printf 'ordered=%.16s sorted=%.16s %8d  %s\n' "$ordered" "$sorted" "$count" "$f"
  done | sort -k2

# Characterize the differences between the same-count-different-set
# workbench export variants (nov10 base/abc/d; oct19 a/b/c/base) found by
# consolidate-checksum-workbench: per pair, how many object lines are
# unique to each side, with truncated samples. Distinguishes benign
# encoding drift (e.g. signature wire-form changes between export runs —
# would spuriously fork the take4 union) from genuinely different object
# sets. Read-only; consolidation Phase 1.
[group('consolidate')]
consolidate-diff-variants:
  #!/usr/bin/env bash
  set -euo pipefail
  cd /home/sasha/workspaces/dodder_migration2
  tmp=$(mktemp)
  trap 'rm -f "$tmp"' EXIT
  pair() {
    echo "=== $1 vs $2"
    comm -3 <(grep '^\[' "$1" | sort) <(grep '^\[' "$2" | sort) > "$tmp" || true
    left=$( (grep -c '^\[' "$tmp" || true) )
    right=$( (grep -c $'^\t' "$tmp" || true) )
    echo "unique-to-first: $left   unique-to-second: $right"
    echo "--- first-side samples:"
    (grep '^\[' "$tmp" || true) | head -2 | cut -c1-300
    echo "--- second-side samples:"
    (grep $'^\t' "$tmp" || true) | head -2 | cut -c2-300
  }
  pair inventory_lists-2025-11-10-a.inventory_list-v2 inventory_lists-2025-11-10.inventory_list-v2
  pair inventory_lists-2025-11-10-a.inventory_list-v2 inventory_lists-2025-11-10-d.inventory_list-v2
  pair objects-2025-10-19-a.inventory_list-v2 objects-2025-10-19.inventory_list-v2
  pair objects-2025-10-19-c.inventory_list-v2 objects-2025-10-19.inventory_list-v2

# Probe every LOCAL madder store for the take4 P9 missing list-blob
# digests via `madder cat` (real markl decoding + store routing — the
# on-disk hex filenames cannot be globbed from base32 ids, which
# invalidated the earlier take3 glob "check"). Starts with a POSITIVE
# CONTROL (a digest from a list the export successfully read must be
# FOUND, else the probe methodology itself is broken and the run
# aborts). Answers "are the 88 actually gone?" with per-store evidence.
# Read-only; consolidation P9 verification.
[group('consolidate')]
consolidate-probe-missing-list-blobs:
  #!/usr/bin/env bash
  set -euo pipefail
  command -v madder >/dev/null || { echo "madder not on PATH (devshell expected to provide it)"; exit 1; }
  work=/home/sasha/workspaces/take4
  digests="$work/missing-list-blobs.txt"
  stores=(dodder-v8-take3 default hardware-pihole-zz-inbox maneater moxy-async nebulous)
  # Control comes from the READABLE era, which is entirely @sha256-
  # digests (the missing 88 are ALL the @blake2b256- ones — a perfect
  # hash-format partition, itself the key P9 finding). (|| true):
  # head-of-pipeline SIGPIPE under pipefail would silently kill the
  # script; emptiness is checked explicitly.
  control=$(grep -o '@sha256-[a-z0-9]*' "$work/exports/default.log-path.lists-only.inventory_list" | tr -d '@' | head -1 || true)
  [[ -n $control ]] || { echo "no readable-list control digest derivable — aborting"; exit 1; }
  echo "==> positive control: $control"
  control_hits=""
  for s in "${stores[@]}"; do
    if madder cat "$s" "$control" >/dev/null 2>&1; then control_hits+=" $s"; fi
  done
  [[ -n $control_hits ]] || { echo "CONTROL NOT FOUND IN ANY STORE — probe methodology broken, aborting"; exit 1; }
  echo "==> control found in:$control_hits — probe is sound"
  echo "==> probing 88 missing digests across: ${stores[*]}"
  found=0; gone=0
  while read -r d; do
    hits=""
    for s in "${stores[@]}"; do
      if madder cat "$s" "$d" >/dev/null 2>&1; then hits+=" $s"; fi
    done
    if [[ -n $hits ]]; then
      echo "FOUND $d in:$hits"; found=$((found+1))
    else
      echo "missing-local $d"; gone=$((gone+1))
    fi
  done <"$digests"
  echo "==> summary: $found found locally, $gone missing from all local stores"

# Snapshot the live default repo's dodder XDG trees — ABOVE ALL the
# cache-category stream index, which is the ONLY remaining local carrier
# of the member states from the 88 P9-holed inventory lists (a stray
# `reindex` would rebuild it from the holed log and destroy them). Copies
# into a timestamped dir under the take4 work area, which the circus
# restic tier (provisioned 2026-08-10) picks up on its next snapshot —
# so this also promotes the at-risk states into off-host backup.
# Read-only w.r.t. the live trees; NOT crash-consistent if dodder writes
# concurrently (acceptable: rerun when in doubt).
[group('consolidate')]
consolidate-backup-live-index:
  #!/usr/bin/env bash
  set -euo pipefail
  stamp=$(date +%Y%m%d-%H%M%S)
  dest=/home/sasha/workspaces/take4/live-repo-backup/$stamp
  mkdir -p "$dest"
  cp -a /home/sasha/.cache/dodder "$dest/cache-dodder"
  cp -a /home/sasha/.local/share/dodder "$dest/share-dodder"
  cp -a /home/sasha/.local/state/dodder "$dest/state-dodder"
  cp -a /home/sasha/.config/dodder "$dest/config-dodder"
  {
    echo "created: $stamp"
    echo "host: $(hostname)"
    echo "purpose: take4 P9 — preserve the stream index (sole carrier of the 88 holed lists' member states) + repo metadata trees"
    echo
    du -sh "$dest"/*
    echo
    echo "file counts:"
    for d in "$dest"/*/; do printf '%8d %s\n' "$(find "$d" -type f | wc -l)" "$d"; done
  } | tee "$dest/MANIFEST"
  echo "==> backup complete: $dest"

# Raw READ-ONLY reconnaissance of the rsync.net account (user zh3447 per
# the rsync_dot_net blob_store-config): list the account root, the
# Library/Madder store tree (dodder's own miss there proves nothing —
# the store config says hash=sha256 so blake2b256 lookups miss by
# construction, madder routing issue), any blake2b256/ format dir, and
# the .zfs snapshot directory rsync.net exposes. Every step tolerant:
# a missing path is a finding, not an error. P9 remote probe.
[group('consolidate')]
consolidate-probe-rsyncnet:
  #!/usr/bin/env bash
  set -uo pipefail
  r() {
    echo "==> ssh zh3447@zh3447.rsync.net -- $*"
    # -F /dev/null: the SYSTEM ssh config pins IdentityFile to the
    # root-owned circus backup key (unreadable as user, and it blocks
    # agent auth); bypass all config and authenticate via the agent,
    # pointing known-hosts at the user file that already trusts the host.
    ssh -F /dev/null \
      -o UserKnownHostsFile=/home/sasha/.config/ssh/known_hosts \
      -o BatchMode=yes -o ConnectTimeout=15 \
      zh3447@zh3447.rsync.net "$@" 2>&1 || echo "(exit $?)"
    echo
  }
  r ls -la
  r ls -la Library
  r ls -la Library/Madder
  r ls Library/Madder/blake2b256
  r ls Library/Madder/sha256
  r ls .zfs/snapshot
  r df -h .

# Content-level sweep of rsync.net's Library/Madder for the 88 P9
# digests. The July-11 cross-hash sync re-keyed blake2b256 content under
# sha256 names with the aliases silently dropped (remoteSftp implements
# no BlobForeignDigestAdder), so presence is only decidable by CONTENT:
# list the remote tree, download small candidates (single-object list
# blobs are tiny), write raw + zstd-decompressed variants into a local
# throwaway blake2b256 store, then ask that store for each of the 88
# original ids — the store is the canonical hasher, no hand-rolled
# markl decoding. Writes only to take4 work area + the scratch store.
[group('consolidate')]
consolidate-rsyncnet-sweep:
  #!/usr/bin/env bash
  set -uo pipefail
  command -v madder >/dev/null || { echo "madder not on PATH"; exit 1; }
  work=/home/sasha/workspaces/take4/rsync-sweep
  digests=/home/sasha/workspaces/take4/missing-list-blobs.txt
  mkdir -p "$work/blobs"
  sshcmd="ssh -F /dev/null -o UserKnownHostsFile=/home/sasha/.config/ssh/known_hosts -o BatchMode=yes -o ConnectTimeout=15"
  echo "==> phase 1: listing remote tree"
  rsync -e "$sshcmd" --list-only -r zh3447@zh3447.rsync.net:Library/Madder/ > "$work/listing.txt"
  echo "listing entries: $(wc -l < "$work/listing.txt")"
  # Candidates: regular files in bucket dirs, 50..8192 bytes, no tmp_.
  awk '$1 ~ /^-/ { size=$2; gsub(",","",size); path=$NF; if (size+0 >= 50 && size+0 <= 8192 && path ~ /^[0-9a-f][0-9a-f]\// && path !~ /tmp_/) print path }' "$work/listing.txt" > "$work/candidates.txt"
  echo "candidates (50B..8KB): $(wc -l < "$work/candidates.txt")"
  [[ -s "$work/candidates.txt" ]] || { echo "no candidates — sweep conclusively empty"; exit 0; }
  echo "==> phase 2: downloading candidates"
  rsync -e "$sshcmd" -r --files-from="$work/candidates.txt" zh3447@zh3447.rsync.net:Library/Madder/ "$work/blobs/"
  echo "downloaded: $(find "$work/blobs" -type f | wc -l)"
  echo "==> phase 3: writing raw + decompressed variants into scratch store"
  madder init take4-sweep-scratch >/dev/null 2>&1 || echo "(scratch store init skipped — may already exist)"
  have_zstd=0; command -v zstd >/dev/null && have_zstd=1
  [[ $have_zstd -eq 1 ]] || echo "WARNING: zstd not on PATH — compressed variants not hashed"
  age_count=0; raw_count=0; dec_count=0
  while read -r f; do
    if head -c 20 "$f" 2>/dev/null | grep -q 'age-encryption.org'; then
      age_count=$((age_count+1)); continue
    fi
    madder write take4-sweep-scratch "$f" >/dev/null 2>&1 && raw_count=$((raw_count+1))
    if [[ $have_zstd -eq 1 ]] && [[ $(head -c 4 "$f" | od -An -tx1 | tr -d ' \n') == 28b52ffd ]]; then
      if zstd -d -q -c "$f" > "$f.dec" 2>/dev/null; then
        madder write take4-sweep-scratch "$f.dec" >/dev/null 2>&1 && dec_count=$((dec_count+1))
      fi
    fi
  done < <(find "$work/blobs" -type f ! -name '*.dec')
  echo "hashed: raw=$raw_count decompressed=$dec_count age-encrypted-skipped=$age_count"
  echo "==> phase 4: membership check — asking the scratch store for the 88 original ids"
  found=0; gone=0
  while read -r d; do
    if madder cat take4-sweep-scratch "$d" >/dev/null 2>&1; then
      echo "RECOVERED $d"; found=$((found+1))
    else
      gone=$((gone+1))
    fi
  done <"$digests"
  echo "==> sweep verdict: $found of 88 recoverable from rsync.net content; $gone not present among candidates"
  # Methodology control: the pandoc-defaults seed blob is small,
  # blake2b256-identified, and in every local store — any full
  # cross-hash sync carried its content, so a working sweep round-trip
  # (remote bytes → scratch store → blake2b256 identity) must find it.
  b2control=blake2b256-zcfmrghzp36r4r4qxtrh4t8xcd5g0f3mkpm8f3swac0vr5x503msyfsu3d
  if madder cat take4-sweep-scratch "$b2control" >/dev/null 2>&1; then
    echo "sweep control: FOUND — round-trip methodology validated; the 88-miss is real absence among candidates"
  else
    echo "sweep control: NOT FOUND — either the sync never carried this blob or the round-trip is broken; verdict NOT conclusive"
  fi

# Discriminate the sweep-control failure without any hashing: fetch the
# control blob's actual CONTENT from a local store and byte-compare it
# (cmp) against every same-sized downloaded candidate. Content found =>
# the round-trip (madder write id computation) is broken; content absent
# from candidates AND no same-size file in the full remote listing =>
# the sync never carried it and the sweep verdict stands. P9.
[group('consolidate')]
consolidate-sweep-verify-control:
  #!/usr/bin/env bash
  set -uo pipefail
  command -v madder >/dev/null || { echo "madder not on PATH"; exit 1; }
  work=/home/sasha/workspaces/take4/rsync-sweep
  b2control=blake2b256-zcfmrghzp36r4r4qxtrh4t8xcd5g0f3mkpm8f3swac0vr5x503msyfsu3d
  tmp=$(mktemp)
  trap 'rm -f "$tmp"' EXIT
  madder cat default "$b2control" > "$tmp" 2>/dev/null || madder cat dodder-v8-take3 "$b2control" > "$tmp"
  size=$(wc -c < "$tmp")
  echo "==> control content size: $size bytes"
  echo "==> same-size files in the FULL remote listing:"
  awk -v s="$size" '$1 ~ /^-/ { sz=$2; gsub(",","",sz); if (sz+0 == s) print }' "$work/listing.txt" | head -20
  echo "==> gathering same-size files (downloaded candidates + direct fetch of any outside the sweep band):"
  sshcmd="ssh -F /dev/null -o UserKnownHostsFile=/home/sasha/.config/ssh/known_hosts -o BatchMode=yes -o ConnectTimeout=15"
  fetch=$work/control-fetch
  mkdir -p "$fetch"
  awk -v s="$size" '$1 ~ /^-/ { sz=$2; gsub(",","",sz); if (sz+0 == s) print $NF }' "$work/listing.txt" > "$fetch/paths.txt"
  echo "same-size remote files: $(wc -l < "$fetch/paths.txt")"
  rsync -e "$sshcmd" -r --files-from="$fetch/paths.txt" zh3447@zh3447.rsync.net:Library/Madder/ "$fetch/blobs/" 2>/dev/null || true
  hit=""
  while read -r f; do
    if cmp -s "$tmp" "$f"; then hit="$f"; break; fi
  done < <(find "$fetch/blobs" "$work/blobs" -type f ! -name '*.dec' -size "${size}c" 2>/dev/null)
  if [[ -n $hit ]]; then
    echo "CONTENT FOUND remotely at $hit — sync DID carry blake2b256-store content"
    echo "==> completing round-trip validation: writing it to scratch and reading back by blake2b256 id"
    madder write take4-sweep-scratch "$hit" >/dev/null 2>&1 || true
    if madder cat take4-sweep-scratch "$b2control" >/dev/null 2>&1; then
      echo "round-trip VALID — scratch-store methodology sound; the 88-miss within the sweep band is real absence"
    else
      echo "round-trip BROKEN — madder write did not produce the expected blake2b256 identity; ALL sweep verdicts invalid, investigate id computation"
    fi
  else
    echo "content not on the remote at size $size — the sync never carried this blob; the sync's source did not include blake2b256-native content (weakens the premise that the 88 could be there at all)"
  fi

# READ-ONLY recon of hardware-pihole (tailnet): connectivity, then
# locate any madder blob-store trees so the P9 content sweep can be
# pointed at them. Tries the plain ssh config first, then the
# config-bypass variant if identity pinning interferes. P9 last probe.
[group('consolidate')]
consolidate-probe-pihole-recon:
  #!/usr/bin/env bash
  set -uo pipefail
  echo "==> network diagnostics:"
  echo "tailscale binary: $(command -v tailscale || echo MISSING)"
  tailscale status 2>&1 | head -15 || true
  echo "--- name resolution attempts:"
  getent hosts hardware-pihole.finch-carp.ts.net || echo "(ts.net name: no)"
  getent hosts hardware-pihole || echo "(bare name: no)"
  getent hosts hardware-pihole.local || echo "(.local: no)"
  ip=$(tailscale status 2>/dev/null | awk '/pihole/ {print $1; exit}')
  host=${ip:-hardware-pihole.finch-carp.ts.net}
  echo "==> target: $host"
  try() {
    echo "==> ssh $* -- probing"
    ssh -o BatchMode=yes -o ConnectTimeout=10 "$@" 'echo "connected as $(whoami)@$(hostname)"' 2>&1
  }
  ok=""
  if try "$host"; then ok="$host"; fi
  if [[ -z $ok ]]; then
    if try -F /dev/null -o UserKnownHostsFile=/home/sasha/.config/ssh/known_hosts "sasha@$host"; then ok="-F /dev/null -o UserKnownHostsFile=/home/sasha/.config/ssh/known_hosts sasha@$host"; fi
  fi
  if [[ -z $ok ]]; then
    if try -F /dev/null -o UserKnownHostsFile=/home/sasha/.config/ssh/known_hosts "pi@$host"; then ok="-F /dev/null -o UserKnownHostsFile=/home/sasha/.config/ssh/known_hosts pi@$host"; fi
  fi
  [[ -n $ok ]] || { echo "no ssh route to $host worked"; exit 1; }
  echo "==> working route: ssh $ok"
  # shellcheck disable=SC2086
  r() { echo "==> $*"; ssh -o BatchMode=yes -o ConnectTimeout=10 $ok "$@" 2>&1 || echo "(exit $?)"; echo; }
  r 'ls -la ~'
  r 'ls -la ~/.local/share/madder/blob_stores 2>/dev/null || echo no-user-madder'
  r 'ls -la /var/lib/madder 2>/dev/null || echo no-var-lib-madder'
  r 'find / -maxdepth 4 -name "blob_store-config" -o -maxdepth 4 -name "*.dodder-blob_store-config" 2>/dev/null | head -20'
  r 'command -v b2sum zstd madder; df -h ~'

# madder-cat probe of the rsync_dot_net store for the P9 digests:
# control (sha256-era, known-synced content) + a 10-digest sample of
# the missing 88. madder's READ path constructs the remote path from
# the requested id, so this tests presence under id-derived naming
# without needing a local bech32 decoder. Each call dials SFTP — sample
# first, full sweep only if the control behaves. P9 remote probe.
[group('consolidate')]
consolidate-probe-rsyncnet-madder:
  #!/usr/bin/env bash
  set -uo pipefail
  command -v madder >/dev/null || { echo "madder not on PATH"; exit 1; }
  work=/home/sasha/workspaces/take4
  control=$(grep -o '@sha256-[a-z0-9]*' "$work/exports/default.log-path.lists-only.inventory_list" | tr -d '@' | head -1 || true)
  echo "==> control (sha256 era): $control"
  if madder cat rsync_dot_net "$control" >/dev/null 2>&1; then
    echo "control: FOUND via rsync_dot_net — read path + sync coverage confirmed for the sha256 era"
  else
    echo "control: NOT FOUND via rsync_dot_net — either the era was never synced or the read path is format-confused; sample results below are NOT conclusive"
  fi
  # blake2b256 FORMAT control: the pandoc-defaults seed type blob exists
  # in every store; if blake2b256 reads work against this sha256-native
  # remote at all, this resolves — otherwise every blake2b256 miss below
  # is a format-routing artifact, not evidence of absence.
  b2control=blake2b256-zcfmrghzp36r4r4qxtrh4t8xcd5g0f3mkpm8f3swac0vr5x503msyfsu3d
  if madder cat rsync_dot_net "$b2control" >/dev/null 2>&1; then
    echo "blake2b256 control: FOUND — blake2b256 remote reads work; misses below are real absence"
  else
    echo "blake2b256 control: NOT FOUND — blake2b256 remote reads unproven (format-routing may mask presence); misses below are NOT conclusive"
  fi
  echo "==> sampling missing digests:"
  { head -5 "$work/missing-list-blobs.txt"; tail -5 "$work/missing-list-blobs.txt"; } | while read -r d; do
    if madder cat rsync_dot_net "$d" >/dev/null 2>&1; then
      echo "FOUND  $d"
    else
      echo "miss   $d"
    fi
  done

# Read-only full-repo signature audit via fsck -recompute. NOTE: this
# does NOT detect the description-newline-collapse bug class (dodder#TBD,
# fixed this session) -- fsck recomputes the digest directly from the
# live store's in-memory decoded fields (stream_index binary format), and
# never re-encodes through box_format's archive encoder or re-decodes
# through the doddish scanner, which is exactly where that bug lived. Use
# debug-export-import-audit instead to find objects whose signature
# breaks specifically under the export/import wire-format round-trip.
# This recipe is still useful for catching OTHER corruption classes
# (bit-rot, storage corruption, unrelated digest mismatches). No query
# means all genres/sigils (latest + history + hidden). Read-only.
#
# run a read-only full-repo signature audit via `fsck -recompute`
[group('debug')]
debug-fsck-full-repo-audit repo_id="default":
  #!/usr/bin/env bash
  set -euo pipefail
  bin=$(nix build --no-link --print-out-paths .#dodder-debug)
  export PATH="$bin/bin:$PATH"
  dodder fsck -recompute -repo_id {{ repo_id }}

# Real audit for the description-newline-collapse bug class: export the
# whole repo (all genres, full history) to a static inventory-list file,
# then import it into a fresh disposable repo. Any object whose signature
# was computed over already-corrupted wire-format text (collapsed
# newlines from before this session's fix) fails import's signature
# re-verification -- printed as an import error naming the object. Unlike
# fsck -recompute (see debug-fsck-full-repo-audit), this actually
# exercises the box_format archive encoder + doddish scanner decoder
# round-trip, i.e. the exact code path the bug lived in. Read-only
# against the source repo; writes only to a throwaway .tmp/ scratch repo.
#
# audit the repo by exporting every object and re-importing into a fresh repo
[group('debug')]
debug-export-import-audit repo_id="default":
  #!/usr/bin/env bash
  set -euo pipefail
  bin=$(nix build --no-link --print-out-paths .#dodder-debug)
  export PATH="$bin/bin:$PATH"

  export_dir="{{ justfile_directory() }}/.tmp/debug-export-import-audit"
  mkdir -p "$export_dir"
  list_path="$export_dir/$(date +%s).inventory_list"

  dodder export -repo_id {{ repo_id }} -print-time=true +z,e,t,k > "$list_path"
  echo "==> exported to: $list_path"
  wc -l "$list_path"

  dest="$export_dir/scratch-$(date +%s)"
  mkdir -p "$dest"
  cd "$dest"
  dodder init -encryption none -exclude-default-type .default
  dodder import -verbose -blob_store-id {{ repo_id }} "$list_path"

# Reproduce the ORIGINAL failing scenario exactly: a repo-backed
# workspace with NO -parent flag (resolves to the home repo, per
# ParentBackedWorkspace.ResolveParentPath -- the same resolution `new
# -ephemeral` uses internally), followed by a plain `pull` -- this is the
# code path (MakeHomeRepoRemote, remote.go) the earlier investigation
# actually failed against, distinct from a plain export/import round-trip
# which this session's audit found clean. Confirms whether the home-repo
# pull path still fails even though export/import doesn't. Assumes the
# ambient XDG-user home repo IS named "default" (the repo this session's
# investigation targeted) -- there is no flag to pick a different home
# repo name; it's whatever $XDG_DATA_HOME/dodder resolves to.
#
# reproduce the repo-backed-workspace + pull failure against the home repo
[group('debug')]
debug-pull-home-repo-repro:
  #!/usr/bin/env bash
  set -euo pipefail
  bin=$(nix build --no-link --print-out-paths .#dodder-debug)
  export PATH="$bin/bin:$PATH"

  dest="{{ justfile_directory() }}/.tmp/debug-pull-home-repo-repro/$(date +%s)"
  mkdir -p "$dest"
  cd "$dest"

  dodder init-workspace debug-ws
  dodder pull -verbose +z,e,t,k

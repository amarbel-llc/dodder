setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"
  export output
}

# bats file_tags=version

# Covers the version-burnin wiring end-to-end: flake.nix defines
# dodderVersion and dodderCommit, go/default.nix passes them via -ldflags,
# each cmd/*/main.go declares the receiving vars, and `dodder version`
# prints them via commands_dodder.SetVersion.
#
# Devshell `go build` (what `just build` does) leaves the defaults
# ("dev+unknown") because the fork's auto-ldflags only fire under
# buildGoApplication. nix-built binaries must match flake.nix's
# dodderVersion. Both states are valid; tests treat them distinctly.

function version_prints_format { # @test
  run "$DODDER_BIN" version
  assert_success

  # Format: <version>+<commit>. Both nonempty.
  assert_output --regexp '^[^+]+\+[^+]+$'
}

function version_matches_flake_version_or_dev { # @test
  run "$DODDER_BIN" version
  assert_success

  if [[ $output == "dev+unknown" ]]; then
    skip "devshell go-build (no ldflags); flake-version match enforced on nix builds"
  fi

  local got_version
  got_version="$(echo "$output" | head -n1 | cut -d+ -f1)"

  local flake_version
  flake_version="$(grep 'dodderVersion = ' "${BATS_TEST_DIRNAME}/../../flake.nix" | sed 's/.*"\(.*\)".*/\1/')"

  [[ $got_version == "$flake_version" ]] ||
    fail "dodder version prefix '$got_version' does not match flake.nix dodderVersion '$flake_version'"
}

function version_der_matches_dodder { # @test
  # Both binaries call commands_dodder.SetVersion with their own
  # ldflag-injected (or default) values, so their reported identity
  # must match byte-for-byte. Detects ldflag drift between subPackages.
  local der_bin="$DODDER_DER_BIN"

  run "$DODDER_BIN" version
  assert_success
  local dodder_version="$output"

  run timeout --preserve-status 2s "$der_bin" version
  assert_success
  local der_version="$output"

  [[ $dodder_version == "$der_version" ]] ||
    fail "dodder=\"$dodder_version\" der=\"$der_version\" disagree"
}

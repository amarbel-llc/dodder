setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"
  export output
}

# bats file_tags=version

# Covers the version-burnin wiring end-to-end: version.env at repo root
# declares DODDER_VERSION, flake.nix reads it (dodderVersion) alongside
# dodderCommit, go/default.nix passes them via -ldflags, each cmd/*/main.go
# declares the receiving vars, and `dodder version` prints them via
# commands_dodder.SetVersion.
#
# Devshell `go build` (what `just build` does) leaves the defaults
# ("dev+unknown") because the fork's auto-ldflags only fire under
# buildGoApplication. nix-built binaries must match version.env's
# DODDER_VERSION. Both states are valid; tests treat them distinctly.

function version_prints_format { # @test
  run "$DODDER_BIN" version
  assert_success

  # Format: <version>+<commit>. Both nonempty.
  assert_output --regexp '^[^+]+\+[^+]+$'
}

function version_matches_version_env_or_dev { # @test
  run "$DODDER_BIN" version
  assert_success

  if [[ $output == "dev+unknown" ]]; then
    skip "devshell go-build (no ldflags); version.env match enforced on nix builds"
  fi

  local got_version
  got_version="$(echo "$output" | head -n1 | cut -d+ -f1)"

  local env_version
  env_version="$(grep '^export DODDER_VERSION=' "${BATS_TEST_DIRNAME}/../../version.env" | cut -d= -f2)"

  [[ $got_version == "$env_version" ]] ||
    fail "dodder version prefix '$got_version' does not match version.env DODDER_VERSION '$env_version'"
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

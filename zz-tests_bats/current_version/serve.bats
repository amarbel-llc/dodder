#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"
  export output

  # dodder serve loads its blob stores from a madder store. Without
  # MADDER_XDG_UTILITY_OVERRIDE, madder defaults to $HOME/.madder
  # which is (a) blocked by sandcastle and (b) the user's real madder
  # store outside the sandbox. Pin both to the test tmpdir so init
  # writes there and serve reads from there.
  export DODDER_XDG_UTILITY_OVERRIDE="$BATS_TEST_TMPDIR"
  export MADDER_XDG_UTILITY_OVERRIDE="$BATS_TEST_TMPDIR"

  run_dodder_init test-serve

  start_server
}

teardown() {
  stop_server
  chflags_nouchg
}

# bats file_tags=serve

# Smoke test: confirm the handshake parser populated server_addr
# and port. start_server's read of the handshake line is itself the
# proof that the listener bound and the protocol contract held.
function serve_handshake_parses_addr { # @test
  [[ -n $server_addr ]] || fail "server_addr empty after start_server"
  [[ -n $port ]] || fail "port empty after start_server"
  [[ $server_addr == "127.0.0.1:$port" ]] || fail \
    "server_addr ($server_addr) != 127.0.0.1:port ($port)"
}

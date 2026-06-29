#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"
  export output

  # dodder serve loads its blob stores from a madder store. Without
  # MADDER_XDG_UTILITY_OVERRIDE, madder defaults to $HOME/.madder
  # which is (a) blocked by the bats sandbox and (b) the user's real madder
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

# curl_status hits a URL with the given method and body, returning
# just the HTTP status code on stdout. Used by the per-endpoint tests
# below to confirm sig-auth gating fires (400 without a nonce header)
# rather than the route being missing (which would return 404).
curl_status() {
  local method="$1"
  local path="$2"
  local data="${3:-}"

  local args=(-s -o /dev/null -w '%{http_code}' -X "$method")
  if [[ -n $data ]]; then
    args+=(-d "$data")
  fi
  args+=("http://${server_addr}${path}")

  curl "${args[@]}"
}

# All five HTTP endpoints share sigMiddleware as the outermost wrap.
# Without `X-Dodder-Challenge-Nonce`, the middleware short-circuits
# with 400 before ever touching the handler. A 404 here would mean
# the route isn't registered — failing back through the matcher to
# the mux's default not-found.
#
# These tests are intentionally negative-path-only: stage 4 (HTTP
# variants of clone/pull/push) covers the positive path through
# RoundTripperBufioWrappedSigner once #166 lands.

function serve_config_immutable_route_registered { # @test
  run curl_status GET /config-immutable
  assert_success
  assert_output 400
}

function serve_blobs_route_registered { # @test
  run curl_status GET /blobs/blake2b256-fakeplaceholder
  assert_success
  assert_output 400
}

function serve_query_route_registered { # @test
  run curl_status GET /query/inventory_list/some_query
  assert_success
  assert_output 400
}

# The /object-history route backs the parent negotiator's over-the-wire
# history fetch for merge resolution on pull (#299). Confirm it is registered
# (sig-auth 400, not a 404 from the not-found matcher).
function serve_object_history_route_registered { # @test
  run curl_status GET /object-history/some_object
  assert_success
  assert_output 400
}

function serve_mcp_route_registered { # @test
  run curl_status POST /mcp '{"jsonrpc":"2.0","method":"initialize","id":1}'
  assert_success
  assert_output 400
}

function serve_inventory_lists_route_registered { # @test
  run curl_status GET /inventory_lists
  assert_success
  assert_output 400
}

function serve_unknown_route_returns_404 { # @test
  run curl_status GET /no-such-endpoint
  assert_success
  assert_output 404
}

# /healthz is the one route that bypasses sigMiddleware. A bare GET
# (no nonce header) must return 200, not 400 — that's the whole
# point: a harness must be able to poll liveness without setting up
# a keypair.
function serve_healthz_returns_200_without_auth { # @test
  run curl_status GET /healthz
  assert_success
  assert_output 200
}

function serve_healthz_body_is_ok { # @test
  run curl -s "http://${server_addr}/healthz"
  assert_success
  assert_output "ok"
}

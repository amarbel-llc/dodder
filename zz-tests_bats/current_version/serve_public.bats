#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"
  export output

  # See serve.bats for the XDG override rationale: pin dodder + madder
  # stores to the test tmpdir so init writes there and serve reads from
  # there rather than the user's real ~/.madder.
  export DODDER_XDG_UTILITY_OVERRIDE="$BATS_TEST_TMPDIR"
  export MADDER_XDG_UTILITY_OVERRIDE="$BATS_TEST_TMPDIR"

  run_dodder_init test-serve-public

  # -public relaxes sigMiddleware for nonce-less read requests (see
  # remote_http.Server.Public). start_server forwards trailing args as
  # extra `serve` flags.
  start_server "" -public
}

teardown() {
  stop_server
  chflags_nouchg
}

# bats file_tags=serve

# Public read mode serves an existing object as a JSON array of
# sku_json_fmt.Transacted when the client sends `Accept:
# application/json`, with no challenge nonce. The `!md` type object is
# committed by `dodder init`, so it is a reliable read target. This is
# the content-negotiation contract the dodder-backed website API
# depends on.
function serve_public_serves_object_as_json { # @test
  local url="http://${server_addr}/objects/%21md"

  run curl -s -o /dev/null -w '%{http_code}' \
    -H 'Accept: application/json' "$url"
  assert_success
  assert_output 200

  run curl -s -o /dev/null -w '%{content_type}' \
    -H 'Accept: application/json' "$url"
  assert_success
  assert_output "application/json; charset=utf-8"

  # Body is a JSON array whose single element is the !md object.
  # Signatures/pubkeys in the payload are non-deterministic, so assert
  # only the stable structural markers via regexp.
  run curl -s -H 'Accept: application/json' "$url"
  assert_success
  assert_output --regexp '^\[\{'
  assert_output --regexp '"object-id":"!md"'
}

# The public-mode bypass is gated to read-only methods: a nonce-less
# POST (a mutating request) must still be rejected with 400 by
# sigMiddleware exactly as on a non-public server, so --public can never
# expose the write handlers (/blobs, /mcp, /inventory_lists)
# unauthenticated.
function serve_public_post_still_requires_nonce { # @test
  run curl -s -o /dev/null -w '%{http_code}' \
    -X POST -d '{"jsonrpc":"2.0","method":"initialize","id":1}' \
    "http://${server_addr}/mcp"
  assert_success
  assert_output 400
}

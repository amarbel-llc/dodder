#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"
  export output

  # Each repo gets its own XDG home so the transfer genuinely moves
  # objects and blobs over the wire rather than sharing a store.
  export them_home="$BATS_TEST_TMPDIR/them-home"
  export us_home="$BATS_TEST_TMPDIR/us-home"
}

teardown() {
  stop_proto_server
  chflags_nouchg
}

# bats file_tags=serve,user_story:pull,user_story:push,user_story:remote

# start_proto_server launches `dodder serve-proto -handshake` as a coproc,
# reads the OS-assigned port from the handshake line, and exports
# server_addr. Mirrors start_server (common.bash) but drives the drtp
# (sierra/remote_proto) backend instead of remote_http.
function start_proto_server {
  local dir="$1"
  shift || true
  local serve_args=("$@")

  coproc proto_server {
    export DODDER_XDG_UTILITY_OVERRIDE="$them_home"
    export MADDER_XDG_UTILITY_OVERRIDE="$them_home"
    if [[ -n $dir ]]; then
      cd "$dir" || exit 1
    fi
    # shellcheck disable=SC2068
    "$DODDER_BIN" serve-proto ${cmd_dodder_def[@]} -handshake ${serve_args[@]}
  }

  local line
  if ! IFS= read -r -t 5 -u "${proto_server[0]}" line; then
    fail <<-EOM
			no handshake from dodder serve-proto within 5s.
			server pid: ${proto_server_PID:-unknown}
		EOM
  fi

  # 1|1|tcp|127.0.0.1:PORT|dodder-drtp-v1
  local _core _app _net addr _proto
  IFS='|' read -r _core _app _net addr _proto <<<"$line"

  if [[ -z $addr ]]; then
    fail <<-EOM
			could not parse handshake line from dodder serve-proto.
			line: $line
		EOM
  fi

  # shellcheck disable=SC2154
  export server_addr="$addr"
}

function stop_proto_server {
  if [[ -n ${proto_server_PID:-} ]]; then
    kill "$proto_server_PID" 2>/dev/null || true
    wait "$proto_server_PID" 2>/dev/null || true
    unset proto_server_PID
  fi
  unset server_addr
}

function bootstrap_them {
  export DODDER_XDG_UTILITY_OVERRIDE="$them_home"
  export MADDER_XDG_UTILITY_OVERRIDE="$them_home"

  mkdir -p them
  (
    pushd them || exit 1
    run_dodder_init -repo_id . "test-repo-id-them"

    run_dodder new -edit=false - <<-EOM
			---
			# wow
			- tag
			! md
			---

			body
		EOM

    assert_success
    assert_output - <<-EOM
			[one/uno @blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc !md "wow" tag]
		EOM
  )
}

# pull_over_websocket exercises the drtp websocket transport end to end:
# serve-proto in `them`, then a fetch from a separate `us` repo over
# ws://. The server runs -public so the fetch needs no client attestation,
# but the client still verifies the server's per-session attestation.
function pull_over_websocket { # @test
  bootstrap_them
  start_proto_server them -public

  export DODDER_XDG_UTILITY_OVERRIDE="$us_home"
  export MADDER_XDG_UTILITY_OVERRIDE="$us_home"

  mkdir -p us
  pushd us || exit 1

  run_dodder_init -repo_id . "test-repo-id-us"

  run_dodder remote-add \
    -remote-connection-type url-websocket \
    toml-repo-uri-v0 \
    "http://${server_addr}" \
    them-ws

  assert_success

  run_dodder pull \
    -remote-connection-type url-websocket \
    /them-ws +zettel,typ,etikett

  assert_success

  # The transferred zettel is now present locally with its blob.
  run_dodder show one/uno
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc !md "wow" tag]
	EOM
}

# clone_over_websocket exercises a genesis clone over the drtp websocket
# transport: a fresh repo is created and populated from `them` in one shot.
function clone_over_websocket { # @test
  bootstrap_them
  start_proto_server them -public

  export DODDER_XDG_UTILITY_OVERRIDE="$us_home"
  export MADDER_XDG_UTILITY_OVERRIDE="$us_home"

  mkdir -p us
  pushd us || exit 1

  run_dodder clone \
    -encryption none \
    -yin <(cat_yin) \
    -yang <(cat_yang) \
    -repo_id . \
    -remote-connection-type url-websocket \
    test-repo-id-us \
    toml-repo-uri-v0 \
    "http://${server_addr}" \
    +zettel,typ,etikett

  assert_success

  run_dodder show one/uno
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc !md "wow" tag]
	EOM
}

# push_over_websocket exercises the inverse direction with mandatory client
# attestation (the server is NOT -public): a zettel created in `us` is
# pushed to `them` over ws://.
function push_over_websocket { # @test
  export DODDER_XDG_UTILITY_OVERRIDE="$them_home"
  export MADDER_XDG_UTILITY_OVERRIDE="$them_home"

  mkdir -p them
  (
    pushd them || exit 1
    run_dodder_init -repo_id . "test-repo-id-them"
  )

  start_proto_server them

  export DODDER_XDG_UTILITY_OVERRIDE="$us_home"
  export MADDER_XDG_UTILITY_OVERRIDE="$us_home"

  mkdir -p us
  pushd us || exit 1

  run_dodder_init -repo_id . "test-repo-id-us"

  run_dodder new -edit=false - <<-EOM
		---
		# wow
		- tag
		! md
		---

		body
	EOM

  assert_success

  run_dodder remote-add \
    -remote-connection-type url-websocket \
    toml-repo-uri-v0 \
    "http://${server_addr}" \
    them-ws

  assert_success

  run_dodder push \
    -remote-connection-type url-websocket \
    /them-ws +zettel,typ,etikett

  assert_success
}

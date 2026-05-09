#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"
  load "$(dirname "$BATS_TEST_FILE")/../lib/clone.bash"
  export output

  # dodder serve loads its blob stores from a madder store. Without
  # MADDER_XDG_UTILITY_OVERRIDE, madder defaults to $HOME/.madder
  # which is (a) blocked by sandcastle and (b) the user's real
  # madder store outside the sandbox. Pin both to the test tmpdir
  # so init writes there and serve reads from there.
  export DODDER_XDG_UTILITY_OVERRIDE="$BATS_TEST_TMPDIR"
  export MADDER_XDG_UTILITY_OVERRIDE="$BATS_TEST_TMPDIR"
}

teardown() {
  stop_server
  chflags_nouchg
}

# bats file_tags=serve,user_story:clone,user_story:repo,user_story:remote

# clone_history_zettel_type_tag_port exercises the TCP/HTTP transport
# path. Skipped pending #171: the URL-transport client returns nil for
# GetObjectStore(), which the pull path now requires. Sig-auth on the
# URL transport is fixed (#170); the next layer of work is a remote
# HTTP object-store proxy. Un-skip when #171 lands.
function clone_history_zettel_type_tag_port { # @test
  skip "blocked on #171: URL transport has no remote object store"

  them="them"
  bootstrap "$them"

  start_server them

  run_clone_default_with \
    test-repo-id-us \
    toml-repo-uri-v0 \
    "http://${server_addr}" \
    +zettel,typ,etikett

  assert_success
  assert_output_unsorted --regexp - <<-'EOM'
		\[!md @blake2b256-45v3c002j9xfjguu2a7ljxnf68tqglg8fa0csjgnn7d2n36ltp0snfjxgj !toml-type-v2]
		\[konfig @blake2b256-.+ !toml-config-v2]
		\[one/dos @blake2b256-fm7kce7793j3npevpm29spk04r6ycxv38dvx3hjxlzl8tcm5m3qq2mml86 !md "zettel with multiple etiketten" this_is_the_first this_is_the_second]
		\[one/uno @blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc !md "wow" tag]
		\[this_is_the_first]
		\[this_is_the_second]
		copied Blob blake2b256-fm7kce7793j3npevpm29spk04r6ycxv38dvx3hjxlzl8tcm5m3qq2mml86 \(36 B\)
		copied Blob blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc \(5 B\)
		copied Blob blake2b256-45v3c002j9xfjguu2a7ljxnf68tqglg8fa0csjgnn7d2n36ltp0snfjxgj \(51 B\)
	EOM

  try_add_new_after_clone
}

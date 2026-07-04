#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"
  load "$(dirname "$BATS_TEST_FILE")/../lib/clone.bash"

  # for shellcheck SC2154
  export output
}

teardown() {
  chflags_nouchg
}

# bats file_tags=user_story:clone,user_story:repo,user_store:xdg,user_story:remote

function clone_history_zettel_type_tag { # @test
  them="them"
  bootstrap "$them"

  run_clone_default_with \
    .default \
    toml-repo-local_override_path-v0 \
    "$(realpath ./them)" \
    +zettel,typ,etikett

  assert_success
  assert_output_unsorted --regexp - <<-'EOM'
		\[!md @blake2b256-45v3c002j9xfjguu2a7ljxnf68tqglg8fa0csjgnn7d2n36ltp0snfjxgj !toml-type-v2]
		\[one/dos @blake2b256-fm7kce7793j3npevpm29spk04r6ycxv38dvx3hjxlzl8tcm5m3qq2mml86 !md "zettel with multiple etiketten" this_is_the_first this_is_the_second]
		\[one/uno @blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc !md "wow" tag]
		copied Blob blake2b256-fm7kce7793j3npevpm29spk04r6ycxv38dvx3hjxlzl8tcm5m3qq2mml86 \(36 B)
		copied Blob blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc \(5 B)
		copied Blob blake2b256-45v3c002j9xfjguu2a7ljxnf68tqglg8fa0csjgnn7d2n36ltp0snfjxgj \(51 B)
	EOM

  try_add_new_after_clone
}

function clone_history_zettel_type_tag_stdio_local { # @test
  them="them"
  bootstrap "$them"

  run_clone_default_with \
    .default \
    toml-repo-local_override_path-v0 \
    "$(realpath them)" \
    +zettel,typ,etikett

  assert_success
  assert_output_unsorted --regexp - <<-EOM
		\[!md @blake2b256-45v3c002j9xfjguu2a7ljxnf68tqglg8fa0csjgnn7d2n36ltp0snfjxgj !toml-type-v2]
		\[one/dos @blake2b256-fm7kce7793j3npevpm29spk04r6ycxv38dvx3hjxlzl8tcm5m3qq2mml86 !md "zettel with multiple etiketten" this_is_the_first this_is_the_second]
		\[one/uno @blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc !md "wow" tag]
		copied Blob blake2b256-fm7kce7793j3npevpm29spk04r6ycxv38dvx3hjxlzl8tcm5m3qq2mml86 \(36 B)
		copied Blob blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc \(5 B)
		copied Blob blake2b256-45v3c002j9xfjguu2a7ljxnf68tqglg8fa0csjgnn7d2n36ltp0snfjxgj \(51 B)
	EOM

  try_add_new_after_clone
}

function clone_history_one_zettel_stdio_local { # @test
  # TODO figure out why stdio_local is not working at all
  skip
  them="them"
  bootstrap "$them"

  run_clone_default_with \
    .default \
    "$(realpath them)" \
    o/d+

  assert_success
  assert_output_unsorted --regexp - <<-'EOM'
		copied Blob blake2b256-fm7kce7793j3npevpm29spk04r6ycxv38dvx3hjxlzl8tcm5m3qq2mml86 \(36 B)
		\[konfig @blake2b256-.* !toml-config-v2]
		\[one/dos @blake2b256-fm7kce7793j3npevpm29spk04r6ycxv38dvx3hjxlzl8tcm5m3qq2mml86 !md "zettel with multiple etiketten" this_is_the_first this_is_the_second]
	EOM
}

function clone_history_zettel_type_tag_stdio_ssh { # @test
  them="them"
  bootstrap "$them"

  run_clone_default_with \
    .default \
    toml-repo-local_override_path-v0 \
    "$(realpath them)" \
    +zettel,typ,etikett

  assert_success
  assert_output_unsorted --regexp - <<-'EOM'
		\[!md @blake2b256-45v3c002j9xfjguu2a7ljxnf68tqglg8fa0csjgnn7d2n36ltp0snfjxgj !toml-type-v2]
		\[one/dos @blake2b256-fm7kce7793j3npevpm29spk04r6ycxv38dvx3hjxlzl8tcm5m3qq2mml86 !md "zettel with multiple etiketten" this_is_the_first this_is_the_second]
		\[one/uno @blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc !md "wow" tag]
		copied Blob blake2b256-fm7kce7793j3npevpm29spk04r6ycxv38dvx3hjxlzl8tcm5m3qq2mml86 \(36 B)
		copied Blob blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc \(5 B)
		copied Blob blake2b256-45v3c002j9xfjguu2a7ljxnf68tqglg8fa0csjgnn7d2n36ltp0snfjxgj \(51 B)
	EOM

  try_add_new_after_clone
}

function clone_history_default_allow_conflicts { # @test
  them="them"
  bootstrap "$them"

  run_clone_default_with \
    .default \
    toml-repo-local_override_path-v0 \
    "$(realpath ./them)"

  assert_success

  run_dodder show +?z,t,e
  assert_success
  assert_output_unsorted - <<-EOM
		[!md @blake2b256-45v3c002j9xfjguu2a7ljxnf68tqglg8fa0csjgnn7d2n36ltp0snfjxgj !toml-type-v2]
		[one/dos @blake2b256-fm7kce7793j3npevpm29spk04r6ycxv38dvx3hjxlzl8tcm5m3qq2mml86 !md "zettel with multiple etiketten" this_is_the_first this_is_the_second]
		[one/uno @blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc !md "wow" tag]
	EOM

  try_add_new_after_clone
}

# clone_history_zettel_type_tag_port lives in clone_port.bats — uses
# the -handshake harness (#150) which needs --allow-local-binding,
# so it runs in the test-bats-network lane.

function clone_direct_local_path { # @test
  them="them"
  bootstrap "$them"

  run_clone_default_with \
    -direct "$(realpath ./them)" \
    .default \
    +zettel,typ,etikett

  assert_success
  assert_output_unsorted --regexp - <<-'EOM'
		\[!md @blake2b256-45v3c002j9xfjguu2a7ljxnf68tqglg8fa0csjgnn7d2n36ltp0snfjxgj !toml-type-v2]
		\[one/dos @blake2b256-fm7kce7793j3npevpm29spk04r6ycxv38dvx3hjxlzl8tcm5m3qq2mml86 !md "zettel with multiple etiketten" this_is_the_first this_is_the_second]
		\[one/uno @blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc !md "wow" tag]
		copied Blob blake2b256-fm7kce7793j3npevpm29spk04r6ycxv38dvx3hjxlzl8tcm5m3qq2mml86 \(36 B)
		copied Blob blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc \(5 B)
		copied Blob blake2b256-45v3c002j9xfjguu2a7ljxnf68tqglg8fa0csjgnn7d2n36ltp0snfjxgj \(51 B)
	EOM

  try_add_new_after_clone
}

# A direct clone seeds the clone's config log from the source repo's
# current config (FDR 0020): config is repo-local and never pulled, so the
# clone adopts the source's config as a new log entry signed by the clone's
# own key. Edits the source config to a distinctive marker before cloning,
# then show-config in the clone streams that marker rather than the
# genesis default.
function clone_direct_seeds_config_from_source { # @test
  them="them"
  bootstrap "$them"

  (
    pushd "$them" || exit 1
    export EDITOR="bash -c 'echo \"# clone-seed-marker\" >> \"\$0\"'"
    run_dodder edit-config
    assert_success
    # edit-config prints the appended config entry as commit
    # confirmation (#266); the blob digest is content-addressed.
    assert_output '[konfig @blake2b256-0rc375uej7v4jjqv6xv3ywtc5nfc7cs5vmyh77wghcy3day0676q2ngalx !toml-config-v2]'

    run_dodder show-config
    assert_success
    assert_line '# clone-seed-marker'
  )

  run_clone_default_with \
    -direct "$(realpath ./them)" \
    .default \
    +zettel,typ,etikett

  assert_success

  run_dodder show-config
  assert_success
  assert_line '# clone-seed-marker'
}

function clone_direct_no_repo_at_path { # @test
  mkdir -p empty_dir

  run_clone_default_with \
    -direct "$(realpath empty_dir)" \
    .default

  assert_failure
  assert_output --partial 'not in a dodder directory'
}

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
  assert_golden_unsorted clone_history_zettel_type_tag_local

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
  assert_golden_unsorted clone_history_zettel_type_tag_stdio_local

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
  assert_golden_unsorted clone_history_zettel_type_tag_stdio_ssh

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
  assert_golden_unsorted clone_history_zettel_type_tag

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
  assert_golden_unsorted clone_direct_local_path

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

# -organize opens the matched objects (resolved against the source repo,
# pre-pull) in an editable outline; deleting an entry excludes it from the
# clone. EDITOR here deletes the "one/dos" heading, leaving only "one/uno"
# to survive into the narrowed query that is actually pulled.
function clone_direct_organize_filters_objects { # @test
  them="them"
  bootstrap "$them"

  function editor() {
    # shellcheck disable=SC2317
    grep -v '^- \[one/dos ' "$1" >"$1.filtered"
    mv "$1.filtered" "$1"
  }

  export -f editor
  # shellcheck disable=SC2016
  export EDITOR='bash -c "editor $0"'

  run_clone_default_with \
    -direct "$(realpath ./them)" \
    -organize \
    .default \
    +zettel,typ,etikett

  assert_success

  run_dodder show :z
  assert_success
  assert_output '[one/uno @blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc !md "wow" tag]'
}

# -organize resolves the pre-pull outline in-process against the remote;
# the websocket transport has no such intermediate step (client.Fetch is a
# single opaque streamed RPC), so -organize is rejected before any dial is
# attempted.
function clone_websocket_organize_rejected { # @test
  run_clone_default_with \
    -remote-connection-type url-websocket \
    -organize \
    .default \
    toml-repo-uri-v0 \
    "http://127.0.0.1:1" \
    +zettel,typ,etikett

  assert_failure
  assert_output --partial '-organize is not supported over the websocket protocol'
}

function clone_direct_no_repo_at_path { # @test
  mkdir -p empty_dir

  run_clone_default_with \
    -direct "$(realpath empty_dir)" \
    .default

  assert_failure
  assert_output --partial 'not in a dodder directory'
}

#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"
  load "$(dirname "$BATS_TEST_FILE")/../lib/clone.bash"
  export output

  # dodder serve loads its blob stores from a madder store. Without
  # MADDER_XDG_UTILITY_OVERRIDE, madder defaults to $HOME/.madder
  # which is (a) blocked by the bats sandbox and (b) the user's real
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
# path: dodder serve, then a clone over http://. Sig-auth fix in #170
# made signing happen at all; #171 added the GET /objects/{oid}
# endpoint and httpRemoteObjectStore so edge expansion can chase
# parents over the wire.
function clone_history_zettel_type_tag_port { # @test
  them="them"
  bootstrap "$them"

  start_server them

  run_clone_default_with \
    .default \
    toml-repo-uri-v0 \
    "http://${server_addr}" \
    +zettel,typ,etikett

  assert_success
  # Pandoc tools default-on (#208): the !md line carries the unquoted
  # two-token blob-ref trailer (per-run sigs -> .+), and the pandoc tool
  # types appear twice each (clone genesis + transferred history).
  assert_output_unsorted --regexp - <<-'EOM'
		\[!md @blake2b256-e3ew5ma0s399rmk3akms90ah2kdmr88l4jluckmdqylnlqtzu7dq60533j !toml-type-v2 .+]
		\[!pandoc-defaults @blake2b256-zcfmrghzp36r4r4qxtrh4t8xcd5g0f3mkpm8f3swac0vr5x503msyfsu3d !toml-type-v2]
		\[!pandoc-defaults @blake2b256-zcfmrghzp36r4r4qxtrh4t8xcd5g0f3mkpm8f3swac0vr5x503msyfsu3d !toml-type-v2]
		\[!pandoc-lua_filter @blake2b256-afnd989ttt3vmeunlj2asss5hjtkqe75vhupupuz2y9uv8wfx8hs6q8szw !toml-type-v2]
		\[!pandoc-lua_filter @blake2b256-afnd989ttt3vmeunlj2asss5hjtkqe75vhupupuz2y9uv8wfx8hs6q8szw !toml-type-v2]
		\[one/dos @blake2b256-fm7kce7793j3npevpm29spk04r6ycxv38dvx3hjxlzl8tcm5m3qq2mml86 !md "zettel with multiple etiketten" this_is_the_first this_is_the_second]
		\[one/uno @blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc !md "wow" tag]
		copied Blob blake2b256-fm7kce7793j3npevpm29spk04r6ycxv38dvx3hjxlzl8tcm5m3qq2mml86 \(36 B\)
		copied Blob blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc \(5 B\)
		copied Blob blake2b256-e3ew5ma0s399rmk3akms90ah2kdmr88l4jluckmdqylnlqtzu7dq60533j \(1\.7 kB\)
	EOM

  try_add_new_after_clone
}

# clone_over_http_seeds_config_from_source exercises RFC 0005 §HTTP Backend
# Transport: the source edits its config to a distinctive marker and serves
# it over HTTP; a clone over http:// fetches the source's config descriptor
# (GET /config) and the named config blob (GET /blobs/{id}), then adopts that
# config as a new config-log entry signed by the clone's own key. Config is
# repo-local and never carried by the object transfer, so without seeding the
# clone would keep its genesis default. Source and clone use separate XDG
# homes so the clone genuinely has its own config log.
function clone_over_http_seeds_config_from_source { # @test
  local them_home="$BATS_TEST_TMPDIR/them-home"
  local us_home="$BATS_TEST_TMPDIR/us-home"

  export DODDER_XDG_UTILITY_OVERRIDE="$them_home"
  export MADDER_XDG_UTILITY_OVERRIDE="$them_home"

  mkdir -p them
  (
    pushd them || exit 1
    run_dodder_init

    export EDITOR="bash -c 'echo \"# clone-seed-marker\" >> \"\$0\"'"
    run_dodder edit-config
    assert_success
    # edit-config prints the appended config entry as commit
    # confirmation; the blob digest is content-addressed.
    assert_output '[konfig @blake2b256-0rc375uej7v4jjqv6xv3ywtc5nfc7cs5vmyh77wghcy3day0676q2ngalx !toml-config-v2]'

    run_dodder show-config
    assert_success
    assert_line '# clone-seed-marker'
  )

  # start_server's coproc inherits them_home from the env at fork time, so
  # the server keeps serving them even after the parent switches to us_home.
  start_server them -public

  export DODDER_XDG_UTILITY_OVERRIDE="$us_home"
  export MADDER_XDG_UTILITY_OVERRIDE="$us_home"

  mkdir -p us
  pushd us || exit 1

  run_clone_default_with \
    .default \
    toml-repo-uri-v0 \
    "http://${server_addr}" \
    +zettel,typ,etikett

  assert_success

  run_dodder show-config
  assert_success
  assert_line '# clone-seed-marker'
}

# clone_over_http_unmodified_source confirms a clone from a source that never
# edited its config still succeeds and lands a working, marker-free config
# (RFC 0005 "MUST tolerate" the ordinary case). The source serves its genesis
# config; the clone fetches and seeds it, but it carries no edit marker.
function clone_over_http_unmodified_source { # @test
  local them_home="$BATS_TEST_TMPDIR/them-home"
  local us_home="$BATS_TEST_TMPDIR/us-home"

  export DODDER_XDG_UTILITY_OVERRIDE="$them_home"
  export MADDER_XDG_UTILITY_OVERRIDE="$them_home"

  bootstrap them

  start_server them -public

  export DODDER_XDG_UTILITY_OVERRIDE="$us_home"
  export MADDER_XDG_UTILITY_OVERRIDE="$us_home"

  mkdir -p us
  pushd us || exit 1

  run_clone_default_with \
    .default \
    toml-repo-uri-v0 \
    "http://${server_addr}" \
    +zettel,typ,etikett

  assert_success

  # The transferred zettel is present, so the clone is a usable repo.
  run_dodder show one/uno
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc !md "wow" tag]
	EOM

  # The config carries no edit marker: the source never edited its config,
  # so nothing spurious was seeded.
  run_dodder show-config
  assert_success
  refute_line '# clone-seed-marker'

  # Seeding nonetheless occurred: the config log holds exactly two entries,
  # the clone's genesis root plus the seeded source-config entry. The source
  # and clone init with distinct keys, so their genesis config blobs differ
  # and the equal-config skip never fires — seeding always appends one entry.
  run_dodder show-config -history
  assert_success
  assert_equal "${#lines[@]}" 2
}

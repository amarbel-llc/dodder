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

# bats file_tags=user_story:clone,user_story:transform

# dodder#392: clone -script pulls the source's objects, applies one Lua
# transform (here tagging every object "cloned"), and commits the result
# re-signed under the clone's OWN key — a clone born already rewritten rather
# than pulled verbatim and corrected after. Direct transfer only.
function clone_script_transforms_and_resigns { # @test
  them="them"
  bootstrap "$them"

  cat >s.lua <<-'EOM'
		local l = dodder.list()

		for object in l:each() do
		  object.Etiketten["cloned"] = true
		end

		return l
	EOM
  script="$(realpath s.lua)"

  run_clone_default_with \
    -direct "$(realpath ./them)" \
    -script "$script" \
    .default \
    +zettel,typ,etikett

  assert_success

  run_dodder show :z
  assert_success
  assert_output_unsorted - <<-EOM
		[one/dos @blake2b256-fm7kce7793j3npevpm29spk04r6ycxv38dvx3hjxlzl8tcm5m3qq2mml86 !md "zettel with multiple etiketten" cloned this_is_the_first this_is_the_second]
		[one/uno @blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc !md "wow" cloned tag]
	EOM

  # Self-containment (dodder#392): the clone does NOT reference the source repo's
  # blob store in its own config — those stores are overlaid read-only during
  # the run and every referenced blob is copied into the clone before commit. A
  # clean fsck (which reads only the clone's stores) proves the copy landed.
  run_dodder fsck
  assert_success
}

# dodder#392 / dodder#393: -script buffers the pull stream, which only the
# direct transport supports today. Over the websocket protocol it is rejected
# before any dial — validated before genesis, so no half-created repo is left.
function clone_script_websocket_rejected { # @test
  run_clone_default_with \
    -remote-connection-type url-websocket \
    -script /dev/null \
    .default \
    toml-repo-uri-v0 \
    "http://127.0.0.1:1" \
    +zettel,typ,etikett

  assert_failure
  assert_output --partial 'clone -script is only supported for direct'
}

#! /usr/bin/env bats

# Tests that exercise a FRESH dodder repo (run_dodder_init_disable_age) in
# the per-test sandbox. Crucially, this file's setup does NOT call
# copy_from_version — copying a fixture into $BATS_TEST_TMPDIR pollutes
# the parent directory of any sub-CWD a test pushd's into, and the
# git-matching XDG-override walk-up then climbs into that fixture and
# turns the fresh init into a split-brain repo. Tests that need a
# fixture base belong in files that explicitly call copy_from_version
# (e.g. checkin.bats).

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

  # for shellcheck SC2154
  export output
}

teardown() {
  chflags_nouchg
}

# https://github.com/amarbel-llc/dodder/issues/40
function checkin_type_file_creates_type_object { # @test
  run_dodder_init_disable_age

  cat >img.type <<-'TYPEFILE'
		---
		! toml-type-v2
		---

		file-extension = "png"
	TYPEFILE

  run_dodder checkin -delete img.type
  assert_success

  # The type object !img should exist after checkin
  run_dodder show '!img:t'
  assert_success
  assert_output --regexp '^\[!img @blake2b256-.+ !toml-type-v2\]$'
}

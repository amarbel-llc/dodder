#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/common.bash"

  # for shellcheck SC2154
  export output

  copy_from_version "$DIR"
}

teardown() {
  chflags_and_rm
}

# bats file_tags=user_story:workspace

function edit_and_change_workspace { # @test
  run_dodder init-workspace
  assert_success

  export EDITOR="/bin/bash -c 'echo \"this is the body 2\" > \"\$0\"'"
  run_dodder edit one/uno
  assert_success
  assert_output - <<-EOM
		      checked out [one/uno.zettel @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
		[one/uno @blake2b256-5sxcr2vpy33y4m72vfn9ya49jjrzrx0wulls880dv66jxfksjsfs5p6pg7]
	EOM

  run_dodder show -format blob one/uno
  assert_success
  assert_output - <<-EOM
		this is the body 2
	EOM
}

function edit_and_dont_change_workspace { # @test
  run_dodder init-workspace
  assert_success

  export EDITOR="true"
  run_dodder edit one/uno
  assert_success
  assert_output - <<-EOM
		      checked out [one/uno.zettel @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM

  run_dodder show -format blob one/uno
  assert_success
  assert_output - <<-EOM
		last time
	EOM
}

# bats file_tags=user_story:noworkspace
# TODO fix no-workspace edits, the files should always live in the temporary directory

function edit_and_change_no_workspace { # @test
  skip
  export EDITOR="/bin/bash -c 'echo \"this is the body 2\" > \"\$0\"'"
  run_dodder edit one/uno
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-5sxcr2vpy33y4m72vfn9ya49jjrzrx0wulls880dv66jxfksjsfs5p6pg7]
	EOM

  run_dodder show -format blob one/uno
  assert_success
  assert_output - <<-EOM
		this is the body 2
	EOM
}

function edit_and_dont_change_no_workspace { # @test
  skip
  export EDITOR="true"
  run_dodder edit one/uno
  assert_success
  assert_output - <<-EOM
	EOM

  run_dodder show -format blob one/uno
  assert_success
  assert_output - <<-EOM
		last time
	EOM
}

function edit_and_format_no_workspace { # @test
  skip

  # shellcheck disable=SC2329
  function editor() {
    out="$(mktemp)"
    "$DODDER_BIN" format-object "$0" >"$out"
    mv "$out" "$0"
  }

  export -f editor

  # shellcheck disable=SC2016
  export EDITOR='bash -c "editor $0"'

  run_dodder edit one/uno
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd]
	EOM

  run_dodder show -format blob one/uno
  assert_success
  assert_output - <<-EOM
		last time
	EOM
}

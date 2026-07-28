#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

  # for shellcheck SC2154
  export output

  copy_from_version "$DIR"
}

teardown() {
  chflags_nouchg
}

# bats file_tags=user_story:workspace

function workspace_show { # @test
  run_dodder init-workspace -experimental-repo=false -query tag-3
  assert_success

  run_dodder show
  assert_success
  assert_output_unsorted - <<-eom
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
	eom

  run_dodder show :e
  assert_success
  assert_output_unsorted - <<-eom
	eom

  run_dodder show one/uno
  assert_success
  assert_output - <<-eom
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	eom
}

function workspace_edit { # @test
  run_dodder init-workspace -experimental-repo=false -query tag-3
  assert_success

  export EDITOR="true"
  run_dodder edit
  assert_success
  assert_output_unsorted - <<-EOM
		      checked out [one/dos.zettel @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		      checked out [one/uno.zettel @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM

  run_dodder show -format blob one/uno
  assert_success
  assert_output - <<-EOM
		last time
	EOM
}

function workspace_checkout { # @test
  run_dodder init-workspace -experimental-repo=false -tags tag-3
  assert_success

  run_dodder checkout
  assert_success
  assert_output ''

  run_dodder checkout :
  assert_success
  assert_output_unsorted - <<-EOM
		      checked out [one/dos.zettel @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		      checked out [one/uno.zettel @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM

  run_dodder show -format blob one/uno.zettel
  assert_success
  assert_output - <<-EOM
		last time
	EOM
}

function workspace_organize { # @test
  run_dodder init-workspace -experimental-repo=false -tags tag-3 -query tag-3
  assert_success

  run_dodder organize -mode output-only
  assert_success
  assert_output - <<-EOM
		---
		- _base = @blake2b256-p3u45j3q3yfyerv48knu7k476uufanvnzgr3yxgehhnf3433kagskrrtjd
		- tag-3
		---
	EOM

  run_dodder organize -mode output-only :
  assert_success
  assert_output - <<-EOM
		---
		- _base = @blake2b256-y7gw8xlxm794s05c7dxyn78lv9e6rvqvc72uch7yc94q8sl45t3slualxm
		- tag-3
		---

		- [one/dos !md tag-4] wow ok again
		- [one/uno !md tag-4] wow the first
	EOM

  run_dodder organize -mode output-only one/uno
  assert_success
  assert_output - <<-EOM
		---
		- _base = @blake2b256-3le2pgux5ymzq3wl72qelxl5q7zaqzacqlhktvzm69pc0w6g95js7t26sj
		- tag-3
		---

		- [one/uno !md tag-4] wow the first
	EOM
}

function workspace_add_no_organize { # @test
  run_dodder init-workspace -experimental-repo=false -tags tag-3 -query tag-3
  assert_success

  echo "file to be added" >todo.wow.md

  run_dodder add -delete -tags new_tags -description "added file" todo.wow.md
  assert_success
  assert_output - <<-EOM
		[two/uno @blake2b256-qdflthfeky7ak3up8qgagd4qx2a8ua5lr4kvffynjl2k4063ja0qr65g5r !md "added file" new_tags tag-3]
		          deleted [todo.wow.md]
	EOM
}

function workspace_add_yes_organize { # @test
  run_dodder init-workspace -experimental-repo=false -tags tag-3 -query tag-3
  assert_success

  echo "file to be added1" >1.md
  echo "file to be added2" >2.md

  function editor() {
    # shellcheck disable=SC2317
    base_line="$(grep '_base = ' "$1")"
    cat - >"$1" <<-EOM
			---
			$base_line
			---

			# tag-two

			- [1.md]

			# tag-one

			- [2.md]
		EOM
  }

  export -f editor

  # shellcheck disable=SC2016
  export EDITOR='bash -c "editor $0"'

  run_dodder add -organize -delete ./*.md
  assert_success
  assert_output - <<-EOM
		[two/uno @blake2b256-5hwedpxxtvucp2wnhcwafgt6y0a93qca3x0522x2j6kmlw0zzp9qvmvt2s !md "2" tag-one]
		[one/tres @blake2b256-ax76uj5gxlkxj0za603p78t3fzyl23tzd977js8qkzv3j5lx8v9smrj5ch !md "1" tag-two]
		          deleted [1.md]
		          deleted [2.md]
	EOM
}

function workspace_add_yes_organize_omit_one { # @test
  run_dodder init-workspace -experimental-repo=false -tags tag-3 -query tag-3
  assert_success

  echo "file to be added1" >1.md
  echo "file to be added2" >2.md

  function editor() {
    # shellcheck disable=SC2317
    base_line="$(grep '_base = ' "$1")"
    cat - >"$1" <<-EOM
			---
			$base_line
			---

			# tag-two

			- [1.md]
		EOM
  }

  export -f editor

  # shellcheck disable=SC2016
  export EDITOR='bash -c "editor $0"'

  run_dodder add -organize -delete ./*.md
  assert_success
  assert_output - <<-EOM
		[two/uno @blake2b256-ax76uj5gxlkxj0za603p78t3fzyl23tzd977js8qkzv3j5lx8v9smrj5ch !md "1" tag-two]
		          deleted [1.md]
	EOM
}

function workspace_parent_directory { # @test
  run_dodder init-workspace -experimental-repo=false -tags tag-3 -query tag-3
  assert_success

  run_dodder info-repo xdg
  assert_success
  assert_output - <<-EOM
		XDG_DATA_HOME=$BATS_TEST_TMPDIR/.dodder/local/share/repos/default
		XDG_CONFIG_HOME=$BATS_TEST_TMPDIR/.dodder/config/repos/default
		XDG_STATE_HOME=$BATS_TEST_TMPDIR/.dodder/local/state/repos/default
		XDG_CACHE_HOME=$BATS_TEST_TMPDIR/.dodder/cache/repos/default
		XDG_RUNTIME_HOME=$BATS_TEST_TMPDIR/.dodder/local/runtime/repos/default
	EOM

  run_dodder info-workspace
  assert_success
  assert_output - <<-EOM
		---
		! toml-workspace_config-v0
		---

		query = "tag-3"

		[defaults]
		tags = ["tag-3"]
	EOM
  run test -f .dodder-workspace

  mkdir -p child
  pushd child || exit 1

  export BATS_TEST_BODY=true

  # Raise the ceiling above $BATS_TEST_TMPDIR so the walk-up from child/
  # can reach the workspace at $BATS_TEST_TMPDIR. common.bash defaults
  # the ceiling to $PWD; that would block parent-directory discovery,
  # which is exactly what this test exercises.
  export DODDER_TEST_CEILING="$(dirname "$BATS_TEST_TMPDIR")"
  export MADDER_TEST_CEILING="$(dirname "$BATS_TEST_TMPDIR")"

  run_dodder info-repo xdg
  assert_success
  assert_output - <<-EOM
		XDG_DATA_HOME=$BATS_TEST_TMPDIR/.dodder/local/share/repos/default
		XDG_CONFIG_HOME=$BATS_TEST_TMPDIR/.dodder/config/repos/default
		XDG_STATE_HOME=$BATS_TEST_TMPDIR/.dodder/local/state/repos/default
		XDG_CACHE_HOME=$BATS_TEST_TMPDIR/.dodder/cache/repos/default
		XDG_RUNTIME_HOME=$BATS_TEST_TMPDIR/.dodder/local/runtime/repos/default
	EOM

  run_dodder info-workspace
  assert_success
  assert_output - <<-EOM
		---
		! toml-workspace_config-v0
		---

		query = "tag-3"

		[defaults]
		tags = ["tag-3"]
	EOM
}

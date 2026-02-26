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

function mergetool_fails_outside_workspace { # @test
  run_dodder merge-tool .
  assert_failure
}

function mergetool_none { # @test
  run_dodder_init_workspace
  run_dodder merge-tool .
  assert_success
  assert_output "nothing to merge"
}

function mergetool_conflict_base {
  run_dodder_init_workspace
  run_dodder checkout one/dos
  assert_success
  assert_output_unsorted - <<-EOM
		      checked out [one/dos.zettel @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
	EOM

  cat - >one/dos.zettel <<-EOM
		---
		# wow ok again
		- get_this_shit_merged
		- tag-3
		- tag-4
		! txt
		---

		not another one, conflict time
	EOM

  run_dodder organize -mode commit-directly one/dos <<-EOM
		---
		! txt2
		---

		# new-etikett-for-all
		- [one/dos  tag-3 tag-4] wow ok again
	EOM
  assert_success
  assert_output_unsorted - <<-EOM
		[!txt2 !toml-type-v1]
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !txt2 "wow ok again" new-etikett-for-all tag-3 tag-4]
	EOM

  run_dodder show -format log new-etikett-for-all:z,e,t
  assert_success
  assert_output_unsorted - <<-EOM
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !txt2 "wow ok again" new-etikett-for-all tag-3 tag-4]
	EOM

  # TODO add better conflict printing output
  run_dodder status one/dos.zettel
  assert_success
  assert_output - <<-EOM
		       conflicted [one/dos.zettel]
	EOM
}

function mergetool_conflict_one_local { # @test
  #TODO-project-2022-zit-collapse_skus
  mergetool_conflict_base

  export BATS_TEST_BODY=true

  run cat one/dos.conflict
  assert_output --regexp - <<-'EOM'
		\[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd [0-9]+\.[0-9]+ dodder-repo-public_key-v1@.* dodder-object-mother-sig-v2@.* !txt2@.* "wow ok again" new-etikett-for-all tag-3 tag-4]
		\[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd [0-9]+\.[0-9]+ dodder-repo-public_key-v1@.* dodder-object-sig-v2@.* !md@.* "wow ok again" tag-3 tag-4]
		\[one/dos @blake2b256-k680x8tenq3j3ts88ydzga2ghjyg75czkfhzr4g3xxc8f33tykjs47n7xv [0-9]+\.[0-9]+ dodder-repo-public_key-v1@.* dodder-object-mother-sig-v2@.* dodder-object-sig-v2@.* !txt "wow ok again" get_this_shit_merged tag-3 tag-4]
	EOM

  # TODO add `-delete` option to `merge-tool`
  run_dodder merge-tool -merge-tool "/bin/bash -c 'cat \"\$0\" >\"\$3\"'" .
  assert_success
  assert_output - <<-EOM
		          deleted [one/dos.conflict]
		          deleted [one/dos.zettel]
		          deleted [one/]
	EOM

  run_dodder show -format blob one/dos
  assert_success
  assert_output - <<-EOM
		not another one
	EOM

  run_dodder status .
  assert_success
  assert_output ''

  run_dodder last
  assert_success
  assert_output_unsorted - <<-EOM
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !txt2 "wow ok again" new-etikett-for-all tag-3 tag-4]
		[!txt2 !toml-type-v1]
	EOM
}

function mergetool_conflict_one_remote { # @test
  #TODO-project-2022-zit-collapse_skus
  mergetool_conflict_base

  # TODO add `-delete` option to `merge-tool`
  run_dodder merge-tool -merge-tool "/bin/bash -c 'cat \"\$2\" >\"\$3\"'" .
  assert_success
  assert_output - <<-EOM
		[!txt !toml-type-v1]
		[one/dos @blake2b256-k680x8tenq3j3ts88ydzga2ghjyg75czkfhzr4g3xxc8f33tykjs47n7xv !txt "wow ok again" get_this_shit_merged tag-3 tag-4]
		          deleted [one/dos.conflict]
		          deleted [one/dos.zettel]
		          deleted [one/]
	EOM

  run_dodder show -format blob one/dos
  assert_success
  assert_output - <<-EOM
		not another one, conflict time
	EOM

  # run_dodder status .
  # assert_success
  # assert_output - <<-EOM
  # 	          changed [one/dos.zettel @blake2b256-k680x8tenq3j3ts88ydzga2ghjyg75czkfhzr4g3xxc8f33tykjs47n7xv !txt "wow ok again" get_this_shit_merged tag-3 tag-4]
  # EOM

  run_dodder last
  assert_success
  assert_output_unsorted - <<-EOM
		[!txt !toml-type-v1]
		[one/dos @blake2b256-k680x8tenq3j3ts88ydzga2ghjyg75czkfhzr4g3xxc8f33tykjs47n7xv !txt "wow ok again" get_this_shit_merged tag-3 tag-4]
	EOM
}

function mergetool_conflict_one_merged { # @test
  #TODO-project-2022-zit-collapse_skus
  mergetool_conflict_base

  cat - >merged <<-EOM
		---
		# wow ok again
		- get_this_shit_merged
		- new-etikett-for-all
		- tag-3
		- tag-4
		! txt2
		---

		not another one, conflict time
	EOM

  # TODO add `-delete` option to `merge-tool`
  run_dodder merge-tool -merge-tool "/bin/bash -c 'cat \"\$2\" >\"\$3\"'" .
  assert_success
  assert_output - <<-EOM
		[!txt !toml-type-v1]
		[one/dos @blake2b256-k680x8tenq3j3ts88ydzga2ghjyg75czkfhzr4g3xxc8f33tykjs47n7xv !txt "wow ok again" get_this_shit_merged tag-3 tag-4]
		          deleted [one/dos.conflict]
		          deleted [one/dos.zettel]
		          deleted [one/]
	EOM

  run_dodder show -format blob one/dos
  assert_success
  assert_output - <<-EOM
		not another one, conflict time
	EOM

  # run_dodder status .
  # assert_success
  # assert_output - <<-EOM
  # 	          changed [one/dos.zettel @blake2b256-k680x8tenq3j3ts88ydzga2ghjyg75czkfhzr4g3xxc8f33tykjs47n7xv !txt "wow ok again" get_this_shit_merged tag-3 tag-4]
  # EOM

  run_dodder last
  assert_success
  assert_output_unsorted - <<-EOM
		[!txt !toml-type-v1]
		[one/dos @blake2b256-k680x8tenq3j3ts88ydzga2ghjyg75czkfhzr4g3xxc8f33tykjs47n7xv !txt "wow ok again" get_this_shit_merged tag-3 tag-4]
	EOM
}

function mergetool_conflict_one_no_merge { # @test
  #TODO-project-2022-zit-collapse_skus
  mergetool_conflict_base

  run_dodder merge-tool -merge-tool "true" .
  assert_failure
}

#! /usr/bin/env bats

setup() {
	load "$(dirname "$BATS_TEST_FILE")/common.bash"

	# for shellcheck SC2154
	export output

	copy_from_version "$DIR"

  run_dodder_init_workspace
}

teardown() {
	chflags_and_rm
}

function recursive_tags_add_one { # @test
	cat - >tag-3.tag <<-EOM
		---
		- recurse
		---

	EOM

	run_dodder checkin -delete .e
	assert_success
	assert_output - <<-EOM
		[tag-3 recurse]
		          deleted [tag-3.tag]
	EOM

	run_dodder show recurse:e,z
	assert_success
	assert_output_unsorted - <<-EOM
		[tag-3 recurse]
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM
}

function recursive_tags_add_one_super_tags { # @test
	cat - >tag-3.tag <<-EOM
		---
		- recurse
		---

	EOM

	run_dodder checkin -delete .e
	assert_success
	assert_output - <<-EOM
		[tag-3 recurse]
		          deleted [tag-3.tag]
	EOM

	run_dodder organize -mode commit-directly <<-EOM
		- [tag-3-sub]
	EOM

	assert_success
	assert_output - <<-EOM
		[tag-3-sub]
	EOM

	run_dodder show -format tags-path tag-3-sub:e
	assert_success
	assert_output_unsorted - <<-EOM
		tag-3-sub [Paths: [TypeSuper:[tag-3 -> recurse] TypeSelf:[tag-3-sub]], All: [recurse:[TypeSuper:[tag-3 -> recurse]] tag-3:[TypeSuper:[tag-3 -> recurse]] tag-3-sub:[TypeSelf:[tag-3-sub]]]]
	EOM

	run_dodder show recurse:e,z
	assert_success
	assert_output_unsorted - <<-EOM
		[tag-3 recurse]
		[tag-3-sub]
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM

	run_dodder show -format tags-path recurse:z,e
	assert_success
	assert_output_unsorted - <<-EOM
		one/dos [Paths: [TypeDirect:[tag-3] TypeIndirect:[tag-3 -> recurse] TypeDirect:[tag-4]], All: [recurse:[TypeIndirect:[tag-3 -> recurse]] tag-3:[TypeDirect:[tag-3] TypeIndirect:[tag-3 -> recurse]] tag-4:[TypeDirect:[tag-4]]]]
		one/uno [Paths: [TypeDirect:[tag-3] TypeIndirect:[tag-3 -> recurse] TypeDirect:[tag-4]], All: [recurse:[TypeIndirect:[tag-3 -> recurse]] tag-3:[TypeDirect:[tag-3] TypeIndirect:[tag-3 -> recurse]] tag-4:[TypeDirect:[tag-4]]]]
		tag-3 [Paths: [TypeDirect:[recurse] TypeSelf:[tag-3]], All: [recurse:[TypeDirect:[recurse]] tag-3:[TypeSelf:[tag-3]]]]
		tag-3-sub [Paths: [TypeSuper:[tag-3 -> recurse] TypeSelf:[tag-3-sub]], All: [recurse:[TypeSuper:[tag-3 -> recurse]] tag-3:[TypeSuper:[tag-3 -> recurse]] tag-3-sub:[TypeSelf:[tag-3-sub]]]]
	EOM

	run_dodder show -format tags-path recurse:e,z
	assert_success
	assert_output_unsorted - <<-EOM
		one/dos [Paths: [TypeDirect:[tag-3] TypeIndirect:[tag-3 -> recurse] TypeDirect:[tag-4]], All: [recurse:[TypeIndirect:[tag-3 -> recurse]] tag-3:[TypeDirect:[tag-3] TypeIndirect:[tag-3 -> recurse]] tag-4:[TypeDirect:[tag-4]]]]
		one/uno [Paths: [TypeDirect:[tag-3] TypeIndirect:[tag-3 -> recurse] TypeDirect:[tag-4]], All: [recurse:[TypeIndirect:[tag-3 -> recurse]] tag-3:[TypeDirect:[tag-3] TypeIndirect:[tag-3 -> recurse]] tag-4:[TypeDirect:[tag-4]]]]
		tag-3 [Paths: [TypeDirect:[recurse] TypeSelf:[tag-3]], All: [recurse:[TypeDirect:[recurse]] tag-3:[TypeSelf:[tag-3]]]]
		tag-3-sub [Paths: [TypeSuper:[tag-3 -> recurse] TypeSelf:[tag-3-sub]], All: [recurse:[TypeSuper:[tag-3 -> recurse]] tag-3:[TypeSuper:[tag-3 -> recurse]] tag-3-sub:[TypeSelf:[tag-3-sub]]]]
	EOM
}

function recursive_tags_with_same_root { # @test
	run_dodder organize -mode commit-directly <<-EOM
		- [project-one-crit priority-0_must]
		- [project-one-general]
	EOM

	assert_success
	assert_output - <<-EOM
		[project-one-crit priority-0_must]
		[project-one-general]
	EOM

	run_dodder show priority-0_must:e
	assert_success
	assert_output_unsorted - <<-EOM
		[project-one-crit priority-0_must]
	EOM

	run_dodder organize -mode commit-directly one/uno <<-EOM
		# project-one-crit, project-one-general
		- [one/uno]
	EOM

	assert_success
	assert_output - <<-EOM
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md project-one-crit project-one-general]
	EOM

	run_dodder show priority-0_must:z
	assert_success
	assert_output_unsorted - <<-EOM
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md project-one-crit project-one-general]
	EOM
}

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

function basic { # @test
	run_dodder export +e,konfig,t,z
	assert_success
	assert_output_unsorted --regexp - <<-'EOM'
		---
		! inventory_list-v2
		---

		\[konfig @blake2b256-.+ .* !toml-config-v2]
		\[!md @blake2b256-.+ .* !toml-type-v1]
	EOM

	assert_output_unsorted --regexp - <<-'EOM'
		\[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd .* !md@.* "wow ok again" tag-3 tag-4]
	EOM
	assert_output_unsorted --regexp - <<-'EOM'
		\[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd .* !md@.* "wow the first" tag-3 tag-4]
	EOM
	assert_output_unsorted --regexp - <<-'EOM'
		\[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd .* !md@.* "wow ok" tag-1 tag-2]
	EOM
}

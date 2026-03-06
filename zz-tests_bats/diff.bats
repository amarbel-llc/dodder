#! /usr/bin/env bats

setup() {
	load "$(dirname "$BATS_TEST_FILE")/common.bash"

	# for shellcheck SC2154
	export output

	copy_from_version "$DIR"

	run_dodder init-workspace
	assert_success

	run_dodder checkout :z,t,e
	assert_success

	export BATS_TEST_BODY=true
}

teardown() {
	chflags_nouchg
}

function diff_all_same { # @test
	run_dodder diff .
	assert_success
	assert_output_unsorted - <<-EOM
	EOM
}

function diff_all_diff { # @test
	echo wowowow >>one/uno.zettel
	run_dodder diff one/uno.zettel
	assert_success
	assert_output ''
}

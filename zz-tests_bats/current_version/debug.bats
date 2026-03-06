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

function debug_options_all() { # @test
	run_dodder info -debug=all
	assert_success

  run test -f cpu.pprof
	assert_success

  run test -f heap.pprof
	assert_success

  run test -f trace
	assert_success
}

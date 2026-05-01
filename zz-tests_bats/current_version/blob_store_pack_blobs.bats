#! /usr/bin/env bats

setup() {
	load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

	# for shellcheck SC2154
	export output
}

function pack_blobs_no_args { # @test
	run_dodder_init_disable_age
	assert_success

	run_madder pack-blobs -format tap
	assert_success
	assert_output --partial 'TAP version 14'
	assert_output --partial '1..0'
}

function pack_blobs_file_into_archive { # @test
	run_dodder_init_disable_age
	assert_success

	run_madder init-inventory-archive .archive
	assert_success

	run_madder pack-blobs -format tap .archive <(echo pack-objects-test-content)
	assert_success
	assert_output --partial 'TAP version 14'
	assert_output --partial 'ok 1'
	assert_output --partial 'pack .archive'
	refute_output --partial 'not ok'
}

function pack_blobs_multiple_files { # @test
	run_dodder_init_disable_age
	assert_success

	run_madder init-inventory-archive .archive
	assert_success

	run_madder pack-blobs -format tap .archive <(echo content-alpha) <(echo content-beta)
	assert_success
	assert_output --partial 'ok 1'
	assert_output --partial 'ok 2'
	assert_output --partial 'pack .archive'
	refute_output --partial 'not ok'
}

function pack_blobs_not_packable_store { # @test
	run_dodder_init_disable_age
	assert_success

	run_madder pack-blobs -format tap <(echo some-content)
	assert_success
	assert_output --partial 'not ok'
	assert_output --partial 'not packable'
}

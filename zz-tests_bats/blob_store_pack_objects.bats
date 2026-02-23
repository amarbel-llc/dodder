#! /usr/bin/env bats

setup() {
	load "$(dirname "$BATS_TEST_FILE")/common.bash"

	# for shellcheck SC2154
	export output
}

function pack_objects_no_args { # @test
	run_dodder_init_disable_age
	assert_success

	run_dodder blob_store-pack-objects
	assert_success
	assert_output --partial 'TAP version 14'
	assert_output --partial '1..0'
}

function pack_objects_file_into_archive { # @test
	run_dodder_init_disable_age
	assert_success

	run_dodder blob_store-init-inventory-archive .archive
	assert_success

	run_dodder blob_store-pack-objects .archive <(echo pack-objects-test-content)
	assert_success
	assert_output --partial 'TAP version 14'
	assert_output --partial 'ok 1'
	assert_output --partial 'pack .archive'
	refute_output --partial 'not ok'
}

function pack_objects_multiple_files { # @test
	run_dodder_init_disable_age
	assert_success

	run_dodder blob_store-init-inventory-archive .archive
	assert_success

	run_dodder blob_store-pack-objects .archive <(echo content-alpha) <(echo content-beta)
	assert_success
	assert_output --partial 'ok 1'
	assert_output --partial 'ok 2'
	assert_output --partial 'pack .archive'
	refute_output --partial 'not ok'
}

function pack_objects_not_packable_store { # @test
	run_dodder_init_disable_age
	assert_success

	run_dodder blob_store-pack-objects <(echo some-content)
	assert_success
	assert_output --partial 'not ok'
	assert_output --partial 'not packable'
}

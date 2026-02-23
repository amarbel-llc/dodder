#! /usr/bin/env bats

setup() {
	load "$(dirname "$BATS_TEST_FILE")/common.bash"

	# for shellcheck SC2154
	export output
}

function pack_list_no_archives { # @test
	run_dodder_init_disable_age
	assert_success

	run_dodder blob_store-pack-list
	assert_success
	refute_output --partial 'entries'
}

function pack_list_shows_archive_after_pack { # @test
	run_dodder_init_disable_age
	assert_success

	run_dodder blob_store-init-inventory-archive .archive
	assert_success

	run_dodder blob_store-write .archive <(echo pack-list-test-content)
	assert_success

	run_dodder blob_store-pack .archive
	assert_success

	run_dodder blob_store-pack-list .archive
	assert_success
	assert_output --partial '1 entries'
}

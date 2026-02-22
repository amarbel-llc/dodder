#! /usr/bin/env bats

setup() {
	load "$(dirname "$BATS_TEST_FILE")/common.bash"

	# for shellcheck SC2154
	export output
}

function pack_no_packable_stores { # @test
	run_dodder_init_disable_age
	assert_success

	run_dodder blob_store-pack
	assert_success
	assert_output --partial 'TAP version 14'
	assert_output --partial '# SKIP not packable'
}

function pack_inventory_archive_with_blob { # @test
	run_dodder_init_disable_age
	assert_success

	run_dodder blob_store-init-inventory-archive .archive
	assert_success

	run_dodder blob_store-write .archive <(echo pack-test-content)
	assert_success

	run_dodder blob_store-pack .archive
	assert_success
	assert_output --partial 'TAP version 14'
	assert_output --partial 'ok'
	assert_output --partial 'pack .archive'
	refute_output --partial 'not ok'
}

function pack_with_blob_store_id_filters_other_stores { # @test
	run_dodder_init_disable_age
	assert_success

	run_dodder blob_store-init-inventory-archive .archive
	assert_success

	run_dodder blob_store-write .archive <(echo filter-test-content)
	assert_success

	run_dodder blob_store-pack .archive
	assert_success
	assert_output --partial 'pack .archive'
	refute_output --partial '.default'
}

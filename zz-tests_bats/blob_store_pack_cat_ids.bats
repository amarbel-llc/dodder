#! /usr/bin/env bats

setup() {
	load "$(dirname "$BATS_TEST_FILE")/common.bash"

	# for shellcheck SC2154
	export output
}

function pack_cat_ids_no_archives { # @test
	run_dodder_init_disable_age
	assert_success

	run_dodder blob_store-pack-cat-ids
	assert_success
	assert_output ''
}

function pack_cat_ids_shows_blob_ids_after_pack { # @test
	run_dodder_init_disable_age
	assert_success

	run_dodder blob_store-init-inventory-archive .archive
	assert_success

	run_dodder blob_store-write .archive <(echo pack-cat-ids-test-content)
	assert_success

	run_dodder blob_store-pack .archive
	assert_success

	run_dodder blob_store-pack-cat-ids
	assert_success
	# Should output exactly one blob ID (the packed blob)
	assert_output --regexp '^[a-z0-9]+-[a-z0-9]+$'
}

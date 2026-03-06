#! /usr/bin/env bats

setup() {
	load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

	# for shellcheck SC2154
	export output
}

function cat_ids { # @test
	run_dodder_init_disable_age
	assert_success

	run_dodder blob_store-cat-ids .default
	assert_success
	assert_output --partial "$(get_konfig_sha)"
}

function cat_with_explicit_store { # @test
	run_dodder_init_disable_age
	assert_success

	run_dodder blob_store-init-inventory-archive .archive
	assert_success

	run_dodder blob_store-write .archive <(echo cat-store-test)
	assert_success
	assert_output --partial 'ok 1'

	# extract the blake2b256 hash from TAP output
	sha="$(echo "$output" | grep -oP 'blake2b256-\S+')"

	run_dodder blob_store-cat .archive "$sha"
	assert_success
	assert_output --partial "cat-store-test"
}

function cat_default_then_archive { # @test
	run_dodder_init_disable_age
	assert_success

	run_dodder blob_store-init-inventory-archive .archive
	assert_success

	run_dodder blob_store-write .archive <(echo archive-content)
	assert_success
	archive_sha="$(echo "$output" | grep -oP 'blake2b256-\S+')"

	# cat from default store (konfig sha), then switch to archive
	run_dodder blob_store-cat "$(get_konfig_sha)" .archive "$archive_sha"
	assert_success
	assert_output --partial "archive-content"
}

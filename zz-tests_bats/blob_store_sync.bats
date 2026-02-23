#! /usr/bin/env bats

setup() {
	load "$(dirname "$BATS_TEST_FILE")/common.bash"

	# for shellcheck SC2154
	export output
}

teardown() {
	teardown_repo
}

# bats file_tags=user_story:blob_store

function blob_store_sync_twice { # @test
	# TODO once migrated to madder blob stores for bats tests, enable this test again
	skip
	setup_repo
	run_dodder blob_store-init test
	assert_success
	assert_output --regexp - <<-EOM
		Wrote config to .*/1-test.dodder-blob_store-config
	EOM

	run_dodder blob_store-sync
	assert_success
	assert_output --regexp - <<-EOM
		Successes: 14, Failures: 0, Ignored: 0, Total: 14
	EOM

	run_dodder blob_store-sync
	assert_success
	assert_output --regexp - <<-EOM
		Successes: 0, Failures: 0, Ignored: 14, Total: 14
	EOM
}

function blob_store_sync_cross_hash_multi_hash_destination { # @test
	run_dodder_init_disable_age
	assert_success

	# write a blob to the default (blake2b256) store
	run_dodder blob_store-write <(echo cross-hash-test)
	assert_success
	blake_sha="$(echo "$output" | grep -oP 'blake2b256-\S+')"

	# init a second store with sha256 (TomlV2 stores are multi-hash by default)
	run_dodder blob_store-init -hash_type-id sha256 -encryption none -lock-internal-files=false .sha256
	assert_success

	# sync from default to sha256 store
	run_dodder blob_store-sync .default .sha256
	assert_success

	# verify the blob exists in the sha256 store under both digests
	run_dodder blob_store-cat-ids .sha256
	assert_success
	assert_output --partial "$blake_sha"

	# verify the blob content is readable from the sha256 store
	run_dodder blob_store-cat .sha256 "$blake_sha"
	assert_success
	assert_line "cross-hash-test"
}

function blob_store_sync_cross_hash_second_sync_skips { # @test
	run_dodder_init_disable_age
	assert_success

	# write a blob to the default (blake2b256) store
	run_dodder blob_store-write <(echo idempotent-test)
	assert_success

	# init a second store with sha256
	run_dodder blob_store-init -hash_type-id sha256 -encryption none -lock-internal-files=false .sha256
	assert_success

	# first sync
	run_dodder blob_store-sync .default .sha256
	assert_success

	# second sync should skip already-synced blobs
	run_dodder blob_store-sync .default .sha256
	assert_success
}

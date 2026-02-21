#! /usr/bin/env bats

setup() {
	load "$(dirname "$BATS_TEST_FILE")/common.bash"

	# for shellcheck SC2154
	export output
}

teardown() {
	chflags_and_rm
}

# bats file_tags=user_story:blob_store,user_story:pack

# Helper: create an inventory archive v1 blob store config directory and file.
# Args: $1 = store directory name, $2 = delta enabled (true/false)
create_inventory_archive_v1_config() {
	local store_name="$1"
	local delta_enabled="$2"
	local config_dir=".madder/local/share/blob_stores/${store_name}"

	mkdir -p "$config_dir"

	cat >"${config_dir}/dodder-blob_store-config" <<-'HEADER'
		---
		! toml-blob_store_config-inventory_archive-v1
		---
	HEADER

	cat >>"${config_dir}/dodder-blob_store-config" <<-EOM

		hash_type-id = 'blake2b256'
		compression-type = 'zstd'
		loose-blob-store-id = '.default'
		encryption = ''

		[delta]
		enabled = ${delta_enabled}
		algorithm = 'bsdiff'
		min-blob-size = 0
		max-blob-size = 0
		size-ratio = 0.0
	EOM
}

# Helper: generate a long shared prefix for creating similar blobs.
shared_blob_prefix() {
	local i
	for i in $(seq 1 50); do
		echo "shared content line ${i} with some padding to make it large enough for delta compression"
	done
}

function pack_inventory_archive_v1_delta { # @test
	run_dodder_init_disable_age

	create_inventory_archive_v1_config "archive" "true"

	# Generate similar blobs with a shared prefix and unique suffixes.
	local prefix
	prefix="$(shared_blob_prefix)"

	# Write blob content to temp files (process substitution inside $()
	# subshells causes fd issues with BATS run + timeout).
	local blob1="$BATS_TEST_TMPDIR/blob1.txt"
	local blob2="$BATS_TEST_TMPDIR/blob2.txt"
	local blob3="$BATS_TEST_TMPDIR/blob3.txt"

	printf '%s\nunique suffix alpha for blob one\n' "$prefix" >"$blob1"
	printf '%s\nunique suffix beta for blob two\n' "$prefix" >"$blob2"
	printf '%s\nunique suffix gamma for blob three\n' "$prefix" >"$blob3"

	# Write blobs to the default loose store and capture their hashes.
	local hash1 hash2 hash3

	run_dodder blob_store-write "$blob1"
	assert_success
	hash1="$(echo "$output" | awk '{print $1}')"
	[[ -n "$hash1" ]] || fail "blob_store-write returned empty hash for blob one"

	run_dodder blob_store-write "$blob2"
	assert_success
	hash2="$(echo "$output" | awk '{print $1}')"
	[[ -n "$hash2" ]] || fail "blob_store-write returned empty hash for blob two"

	run_dodder blob_store-write "$blob3"
	assert_success
	hash3="$(echo "$output" | awk '{print $1}')"
	[[ -n "$hash3" ]] || fail "blob_store-write returned empty hash for blob three"

	# Verify blobs exist in the default loose store before packing.
	run_dodder blob_store-cat "$hash1"
	assert_success

	run_dodder blob_store-cat "$hash2"
	assert_success

	run_dodder blob_store-cat "$hash3"
	assert_success

	# Pack timeout is higher than default 2s due to bsdiff delta computation.
	run timeout --preserve-status "10s" madder pack
	assert_success

	# Verify blobs are still readable after packing.
	run_dodder blob_store-cat "$hash1"
	assert_success
	assert_output --partial "unique suffix alpha for blob one"

	run_dodder blob_store-cat "$hash2"
	assert_success
	assert_output --partial "unique suffix beta for blob two"

	run_dodder blob_store-cat "$hash3"
	assert_success
	assert_output --partial "unique suffix gamma for blob three"

	# Verify archive data files were created.
	run find .madder/local/share/blob_stores/archive -name '*.inventory_archive-v1' -type f
	assert_success
	assert_output # non-empty output means files exist
}

function pack_inventory_archive_v1_no_delta { # @test
	run_dodder_init_disable_age

	create_inventory_archive_v1_config "archive" "false"

	# Write blob content to temp files.
	local blob1="$BATS_TEST_TMPDIR/blob1.txt"
	local blob2="$BATS_TEST_TMPDIR/blob2.txt"
	local blob3="$BATS_TEST_TMPDIR/blob3.txt"

	echo "no delta blob content alpha" >"$blob1"
	echo "no delta blob content beta" >"$blob2"
	echo "no delta blob content gamma" >"$blob3"

	# Write blobs to the default loose store and capture their hashes.
	local hash1 hash2 hash3

	run_dodder blob_store-write "$blob1"
	assert_success
	hash1="$(echo "$output" | awk '{print $1}')"
	[[ -n "$hash1" ]] || fail "blob_store-write returned empty hash for blob one"

	run_dodder blob_store-write "$blob2"
	assert_success
	hash2="$(echo "$output" | awk '{print $1}')"
	[[ -n "$hash2" ]] || fail "blob_store-write returned empty hash for blob two"

	run_dodder blob_store-write "$blob3"
	assert_success
	hash3="$(echo "$output" | awk '{print $1}')"
	[[ -n "$hash3" ]] || fail "blob_store-write returned empty hash for blob three"

	# Verify blobs exist before packing.
	run_dodder blob_store-cat "$hash1"
	assert_success

	run_dodder blob_store-cat "$hash2"
	assert_success

	run_dodder blob_store-cat "$hash3"
	assert_success

	# Pack timeout is higher than default 2s due to archive write overhead.
	run timeout --preserve-status "10s" madder pack
	assert_success

	# Verify blobs are still readable after packing.
	run_dodder blob_store-cat "$hash1"
	assert_success
	assert_output "no delta blob content alpha"

	run_dodder blob_store-cat "$hash2"
	assert_success
	assert_output "no delta blob content beta"

	run_dodder blob_store-cat "$hash3"
	assert_success
	assert_output "no delta blob content gamma"

	# Verify archive data files were created.
	run find .madder/local/share/blob_stores/archive -name '*.inventory_archive-v1' -type f
	assert_success
	assert_output # non-empty output means files exist
}

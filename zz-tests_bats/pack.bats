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

# Helper: write a blob and capture its hash from the first field of output.
write_blob_and_capture_hash() {
	local content="$1"
	run_dodder blob_store-write <(echo "$content")
	assert_success
	echo "$output" | awk '{print $1}'
}

function pack_inventory_archive_v1_delta { # @test
	run_dodder_init_disable_age

	create_inventory_archive_v1_config "archive" "true"

	# Generate similar blobs with a shared prefix and unique suffixes.
	local prefix
	prefix="$(shared_blob_prefix)"

	local hash1 hash2 hash3

	hash1="$(write_blob_and_capture_hash "${prefix}
unique suffix alpha for blob one")"
	[[ -n "$hash1" ]] || fail "write_blob_and_capture_hash returned empty hash for blob one"

	hash2="$(write_blob_and_capture_hash "${prefix}
unique suffix beta for blob two")"
	[[ -n "$hash2" ]] || fail "write_blob_and_capture_hash returned empty hash for blob two"

	hash3="$(write_blob_and_capture_hash "${prefix}
unique suffix gamma for blob three")"
	[[ -n "$hash3" ]] || fail "write_blob_and_capture_hash returned empty hash for blob three"

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
	local expected1="${prefix}
unique suffix alpha for blob one"
	local expected2="${prefix}
unique suffix beta for blob two"
	local expected3="${prefix}
unique suffix gamma for blob three"

	run_dodder blob_store-cat "$hash1"
	assert_success
	assert_output "$expected1"

	run_dodder blob_store-cat "$hash2"
	assert_success
	assert_output "$expected2"

	run_dodder blob_store-cat "$hash3"
	assert_success
	assert_output "$expected3"

	# Verify archive data files were created.
	run find .madder/local/share/blob_stores/archive -name '*.inventory_archive-v1' -type f
	assert_success
	assert_output # non-empty output means files exist
}

function pack_inventory_archive_v1_no_delta { # @test
	run_dodder_init_disable_age

	create_inventory_archive_v1_config "archive" "false"

	# Generate blobs (no need for similar content since delta is disabled).
	local hash1 hash2 hash3

	hash1="$(write_blob_and_capture_hash "no delta blob content alpha")"
	[[ -n "$hash1" ]] || fail "write_blob_and_capture_hash returned empty hash for blob one"

	hash2="$(write_blob_and_capture_hash "no delta blob content beta")"
	[[ -n "$hash2" ]] || fail "write_blob_and_capture_hash returned empty hash for blob two"

	hash3="$(write_blob_and_capture_hash "no delta blob content gamma")"
	[[ -n "$hash3" ]] || fail "write_blob_and_capture_hash returned empty hash for blob three"

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

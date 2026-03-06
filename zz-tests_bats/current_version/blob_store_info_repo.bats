#! /usr/bin/env bats

setup() {
	load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

	# for shellcheck SC2154
	export output
}

teardown() {
	chflags_nouchg
}

# bats file_tags=user_story:blob_store,user_story:repo_info

function info_repo_encryption_default_store { # @test
	run_dodder_init_disable_age
	assert_success

	run_dodder blob_store-info-repo encryption
	assert_success
	assert_output ''
}

function info_repo_compression_type_default_store { # @test
	run_dodder_init_disable_age
	assert_success

	run_dodder blob_store-info-repo compression-type
	assert_success
	assert_output 'zstd'
}

function info_repo_hash_type_id_default_store { # @test
	run_dodder_init_disable_age
	assert_success

	run_dodder blob_store-info-repo hash_type-id
	assert_success
	assert_output 'blake2b256'
}

function info_repo_lock_internal_files_default_store { # @test
	run_dodder_init_disable_age
	assert_success

	run_dodder blob_store-info-repo lock-internal-files
	assert_success
	assert_output 'false'
}

function info_repo_archive_encryption { # @test
	run_dodder_init_disable_age
	assert_success

	run_dodder blob_store-init-inventory-archive -encryption generate .archive
	assert_success

	run_dodder blob_store-info-repo .archive encryption
	assert_success
	assert_output --regexp '.+'
}

function info_repo_archive_delta_enabled { # @test
	run_dodder_init_disable_age
	assert_success

	run_dodder blob_store-init-inventory-archive -delta .archive
	assert_success

	run_dodder blob_store-info-repo .archive delta.enabled
	assert_success
	assert_output 'true'
}

function info_repo_archive_max_pack_size { # @test
	run_dodder_init_disable_age
	assert_success

	run_dodder blob_store-init-inventory-archive .archive
	assert_success

	run_dodder blob_store-info-repo .archive max-pack-size
	assert_success
	assert_output '0'
}

function info_repo_unknown_key_fails { # @test
	run_dodder_init_disable_age
	assert_success

	run_dodder blob_store-info-repo nonexistent-key
	assert_failure
}

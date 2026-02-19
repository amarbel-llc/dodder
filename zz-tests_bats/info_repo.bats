#! /usr/bin/env bats

setup() {
	load "$(dirname "$BATS_TEST_FILE")/common.bash"

	# for shellcheck SC2154
	export output

	# copy_from_version "$DIR"
}

teardown() {
	chflags_and_rm
}

# bats file_tags=user_story:repo_info

# bats test_tags=user_story:config-immutable
function info_config_immutable { # @test
	run_dodder_init_disable_age
	run_dodder info store-version
	assert_success
	assert_output --regexp '[0-9]+'

	# shellcheck disable=SC2034
	storeVersionCurrent="$output"

	run_dodder info-repo config-immutable
	assert_success

	if [[ "$storeVersionCurrent" -le 10 ]]; then
		assert_output --regexp - <<-EOM
			---
			! toml-config-immutable-v1
			---

			public-key = 'dodder-repo-public_key-v1.*'
			store-version = $storeVersionCurrent
			repo-type = 'working-copy'
			id = 'test-repo-id'
			inventory_list-type = '!inventory_list-v2'

			\[blob-store]
			compression-type = 'zstd'
			lock-internal-files = false
		EOM
	else
		assert_output --regexp - <<-EOM
			---
			! toml-config-immutable-v2
			---

			public-key = 'dodder-repo-public_key-v1.*'
			store-version = $storeVersionCurrent
			id = 'test-repo-id'
			inventory_list-type = '!inventory_list-v2'
			object-sig-type = 'dodder-object-sig-v2'
		EOM
	fi
}

# bats test_tags=user_story:store_version
function info_store_version { # @test
	run_dodder info-repo
	assert_output
}

# bats test_tags=user_story:age_encryption
function info_age_none { # @test
	run_dodder_init_disable_age
	run_dodder info-repo blob_stores-0-encryption
	assert_output ''
}

# bats test_tags=user_story:age_encryption
function info_age_some { # @test
	run_dodder gen madder-private_key-v1
	assert_output --regexp 'madder-private_key-v1@age_x25519_sec-'
	key="$output"
	echo "$key" >age-key
	run_dodder_init -override-xdg-with-cwd -encryption age-key test-repo-id
	run_dodder info-repo blob_stores-0-encryption
	assert_output "$key"
}

# bats test_tags=user_story:compression
function info_compression_type { # @test
	run_dodder_init_disable_age
	run_dodder info-repo compression-type
	assert_output 'zstd'
}

# bats test_tags=user_story:xdg
function info_xdg { # @test
	set_xdg "$BATS_TEST_TMPDIR"
	run_dodder_init_disable_age_xdg
	run_dodder info-repo xdg
	local resolved
	resolved="$(realpath "$BATS_TEST_TMPDIR")"
	assert_output - <<-EOM
		XDG_DATA_HOME=$resolved/.xdg/data/dodder
		XDG_CONFIG_HOME=$resolved/.xdg/config/dodder
		XDG_STATE_HOME=$resolved/.xdg/state/dodder
		XDG_CACHE_HOME=$resolved/.xdg/cache/dodder
		XDG_RUNTIME_HOME=$resolved/.xdg/runtime/dodder
	EOM
}

function info_non_xdg { # @test
	run_dodder_init -override-xdg-with-cwd test-repo-id
	run_dodder info-repo xdg
	assert_output - <<-EOM
		XDG_DATA_HOME=$BATS_TEST_TMPDIR/.dodder/local/share
		XDG_CONFIG_HOME=$BATS_TEST_TMPDIR/.dodder/config
		XDG_STATE_HOME=$BATS_TEST_TMPDIR/.dodder/local/state
		XDG_CACHE_HOME=$BATS_TEST_TMPDIR/.dodder/cache
		XDG_RUNTIME_HOME=$BATS_TEST_TMPDIR/.dodder/local/runtime
	EOM
}

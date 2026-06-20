#! /usr/bin/env bats

setup() {
	load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

	# for shellcheck SC2154
	export output

}

teardown() {
	chflags_nouchg
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

	assert_output --regexp - <<-EOM
		---
		! toml-config-immutable-v2
		---

		public-key = "dodder-repo-public_key-v1.*"
		store-version = $storeVersionCurrent
		id = "test-repo-id"
		inventory_list-type = "!inventory_list-v2"
		object-sig-type = "dodder-object-sig-v2"
	EOM
}

# bats test_tags=user_story:store_version
function info_store_version { # @test
	run_dodder info-repo
	assert_output
}

# bats test_tags=user_story:age_encryption
function info_age_none { # @test
	run_dodder_init_disable_age
	run_dodder info-repo encryption
	assert_output ''
}

# bats test_tags=user_story:age_encryption
function info_age_some { # @test
	run_dodder gen madder-private_key-v1
	assert_output --regexp 'madder-private_key-v1@age_x25519_sec-'
	key="$output"
	echo "$key" >age-key
	run_dodder_init -repo_id .default -encryption age-key test-repo-id
	run_dodder info-repo encryption
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
		XDG_DATA_HOME=$resolved/.xdg/data/dodder/repos/default
		XDG_CONFIG_HOME=$resolved/.xdg/config/dodder/repos/default
		XDG_STATE_HOME=$resolved/.xdg/state/dodder/repos/default
		XDG_CACHE_HOME=$resolved/.xdg/cache/dodder/repos/default
		XDG_RUNTIME_HOME=$resolved/.xdg/runtime/dodder/repos/default
	EOM
}

function info_non_xdg { # @test
	run_dodder_init -repo_id .default test-repo-id
	run_dodder info-repo xdg
	assert_output - <<-EOM
		XDG_DATA_HOME=$BATS_TEST_TMPDIR/.dodder/local/share/repos/default
		XDG_CONFIG_HOME=$BATS_TEST_TMPDIR/.dodder/config/repos/default
		XDG_STATE_HOME=$BATS_TEST_TMPDIR/.dodder/local/state/repos/default
		XDG_CACHE_HOME=$BATS_TEST_TMPDIR/.dodder/cache/repos/default
		XDG_RUNTIME_HOME=$BATS_TEST_TMPDIR/.dodder/local/runtime/repos/default
	EOM
}

function info_repo_unknown_key_fails { # @test
	run_dodder_init_disable_age
	assert_success

	run_dodder info-repo nonexistent-key
	assert_failure
}

function info_repo_dynamic_config_key { # @test
	run_dodder_init_disable_age
	assert_success

	run_dodder info-repo blob-store-type
	assert_success
	assert_output --regexp '.+'
}

# bats test_tags=user_story:repos
function info_repo_repos_lists_repos { # @test
	# run_dodder_init creates a cwd repo (-repo_id .default), so the listing
	# shows its routable `.default` spelling, not the ambiguous bare
	# `default` (which would name a user-scope repo). FDR-0019 #276.
	run_dodder_init test-repo-id

	run_dodder info-repo repos
	assert_success
	assert_output '.default'
}

# bats test_tags=user_story:repos
function info_repo_repos_lists_both_scopes { # @test
	# A -repo_id can address both a user-scope repo (`name`) and a cwd-scope
	# repo (`.name`) regardless of cwd, so `info-repo repos` lists both
	# scopes together with their directly-usable spellings. FDR-0019 #276.
	run_dodder init \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-repo_id userrepo \
		test-repo-id
	assert_success

	run_dodder_init test-repo-id

	run_dodder info-repo repos
	assert_success
	assert_output - <<-EOM
		.default
		userrepo
	EOM
}

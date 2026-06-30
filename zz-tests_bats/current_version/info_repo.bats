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
		id = ""
		inventory_list-type = "!inventory_list-v2"
		object-sig-type = "dodder-object-sig-v2"
	EOM
}

# bats test_tags=user_story:store_version
function info_store_version { # @test
	run_dodder info-repo
	assert_output
}

# bats test_tags=user_story:repo_identity
function info_repo_id_shows_handle_at_pubkey { # @test
	# #294 / FDR-0021: `info-repo id` emits the repo's human-facing identity
	# <handle>@<pubkey> --- the addressable location handle joined to the
	# repo's public key --- not the deprecated config-seed id. The handle is
	# the FDR-0019 scoped repo id (`work` here); the pubkey is freshly
	# generated per repo, so only its format is matched with --regexp.
	run_dodder init -yin <(cat_yin) -yang <(cat_yang) work
	assert_success

	run_dodder info-repo -repo_id work id
	assert_success
	assert_output --regexp '^work@ed25519_pub-'
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
	run_dodder_init -encryption age-key .default
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
	run_dodder_init
	run_dodder info-repo xdg
	assert_output - <<-EOM
		XDG_DATA_HOME=$BATS_TEST_TMPDIR/.dodder/local/share/repos/default
		XDG_CONFIG_HOME=$BATS_TEST_TMPDIR/.dodder/config/repos/default
		XDG_STATE_HOME=$BATS_TEST_TMPDIR/.dodder/local/state/repos/default
		XDG_CACHE_HOME=$BATS_TEST_TMPDIR/.dodder/cache/repos/default
		XDG_RUNTIME_HOME=$BATS_TEST_TMPDIR/.dodder/local/runtime/repos/default
	EOM
}

# bats test_tags=repo_id
function info_repo_system_scope_roots_at_system_root { # @test
	# FDR-0019 #280: a system-scoped //name repo roots under the configured
	# system root (DODDER_SYSTEM_ROOT, sandboxed here) on BOTH the init and
	# operate paths, rather than silently falling back to the user tree.
	export DODDER_SYSTEM_ROOT="$BATS_TEST_TMPDIR/system"

	run_dodder init \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-encryption none \
		//backup
	assert_success

	# init rooted the repo's xdg under the system root, not $HOME
	run_dodder info-repo -repo_id //backup xdg
	assert_success
	assert_output --regexp "XDG_DATA_HOME=${DODDER_SYSTEM_ROOT}/"
	refute_output --regexp "XDG_DATA_HOME=$BATS_TEST_TMPDIR/.xdg/"

	# the operate path resolves the system repo too
	run_dodder show-config -repo_id //backup
	assert_success
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
	# run_dodder_init creates a cwd repo (the `.default` location positional),
	# so the listing shows its routable `.default` spelling, not the ambiguous
	# bare `default` (which would name a user-scope repo). FDR-0019 #276.
	run_dodder_init

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
		userrepo
	assert_success

	run_dodder_init

	run_dodder info-repo repos
	assert_success
	assert_output - <<-EOM
		.default
		userrepo
	EOM
}

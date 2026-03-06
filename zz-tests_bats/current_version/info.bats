#! /usr/bin/env bats

setup() {
	load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

	# for shellcheck SC2154
	export output
}

teardown() {
	chflags_nouchg
}

# bats file_tags=user_story:info

# bats test_tags=user_story:store_version
function info_store_version { # @test
	run_dodder info
	assert_output
}

# bats test_tags=user_story:compression
function info_compression_type { # @test
	run_dodder info compression-type
	assert_output 'zstd'
}

# bats test_tags=user_story:xdg
function info_xdg { # @test
	loc="$BATS_TEST_TMPDIR"
	export XDG_DATA_HOME="$loc/.xdg/data"
	export XDG_CONFIG_HOME="$loc/.xdg/config"
	export XDG_STATE_HOME="$loc/.xdg/state"
	export XDG_CACHE_HOME="$loc/.xdg/cache"
	export XDG_RUNTIME_HOME="$loc/.xdg/runtime"

	run_dodder_init_disable_age_xdg
	run_dodder info xdg
	assert_output - <<-EOM
		XDG_CACHE_HOME=$BATS_TEST_TMPDIR/.xdg/cache/dodder
		XDG_CONFIG_HOME=$BATS_TEST_TMPDIR/.xdg/config/dodder
		XDG_DATA_HOME=$BATS_TEST_TMPDIR/.xdg/data/dodder
		XDG_RUNTIME_HOME=$BATS_TEST_TMPDIR/.xdg/runtime/dodder
		XDG_STATE_HOME=$BATS_TEST_TMPDIR/.xdg/state/dodder
	EOM
}

function info_non_xdg { # @test
	run_dodder_init -override-xdg-with-cwd test-repo-id
	run_dodder info xdg
	assert_output - <<-EOM
		XDG_CACHE_HOME=$BATS_TEST_TMPDIR/.dodder/cache
		XDG_CONFIG_HOME=$BATS_TEST_TMPDIR/.dodder/config
		XDG_DATA_HOME=$BATS_TEST_TMPDIR/.dodder/local/share
		XDG_RUNTIME_HOME=$BATS_TEST_TMPDIR/.dodder/local/runtime
		XDG_STATE_HOME=$BATS_TEST_TMPDIR/.dodder/local/state
	EOM
}

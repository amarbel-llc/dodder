#! /usr/bin/env bats

setup() {
	load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

	# for shellcheck SC2154
	export output

	copy_from_version "$DIR"
}

teardown() {
	chflags_nouchg
}

# bats file_tags=user_story:workspace

function ceiling_blocks_workspace_discovery { # @test
	run_dodder_init_workspace
	run test -f .dodder-workspace
	assert_success

	mkdir -p child/grandchild
	pushd child/grandchild || exit 1

	# Without ceiling, workspace is discovered from child dir
	run_dodder info-workspace
	assert_success

	# With ceiling set to the test tmpdir, workspace discovery stops
	export DODDER_CEILING_DIRECTORIES="$BATS_TEST_TMPDIR"

	run_dodder info-workspace
	assert_failure
	assert_output --partial 'not in a workspace'
}

function ceiling_does_not_block_workspace_in_cwd { # @test
	run_dodder_init_workspace
	run test -f .dodder-workspace
	assert_success

	# Ceiling at the test dir itself should not block finding .dodder-workspace
	# in the current directory (only blocks walking *above* it)
	export DODDER_CEILING_DIRECTORIES="$(dirname "$BATS_TEST_TMPDIR")"

	run_dodder info-workspace
	assert_success
}

function ceiling_ignores_relative_paths { # @test
	run_dodder_init_workspace
	run test -f .dodder-workspace
	assert_success

	mkdir -p child
	pushd child || exit 1

	# Relative paths in ceiling list are silently ignored
	export DODDER_CEILING_DIRECTORIES="relative/path"

	run_dodder info-workspace
	assert_success
}

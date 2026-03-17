#! /usr/bin/env bats

setup() {
	load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

	# for shellcheck SC2154
	export output
}

teardown() {
	chflags_nouchg
}

# bats file_tags=user_story:workspace,user_story:repo

function bootstrap_workspace_repo {
	local parent="$1"
	local workspace="$2"

	(
		mkdir -p "$parent"
		pushd "$parent" || exit 1
		run_dodder_init -repo_id . "parent-repo-id"

		run_dodder new -edit=false - <<-EOM
			---
			# test zettel
			- project-alpha
			! md
			---

			test body
		EOM
		assert_success
	)

	local parent_path
	parent_path="$(realpath "$parent")"

	mkdir -p "$workspace"
	pushd "$workspace" || exit 1

	run_dodder init-workspace \
		-experimental-repo \
		-encryption none \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-repo_id . \
		-lock-internal-files=false \
		-direct "$parent_path" \
		workspace-repo-id \
		+zettel,typ,etikett

	assert_success
}

function check_workspace_dirty_clean_after_init { # @test
	bootstrap_workspace_repo parent workspace

	# Immediately after init, workspace should be clean
	run_dodder check-workspace dirty
	assert_failure  # exit 1 = clean
}

function check_workspace_dirty_after_local_change { # @test
	bootstrap_workspace_repo parent workspace

	# Create a new zettel in the workspace
	run_dodder new -edit=false - <<-EOM
		---
		# workspace zettel
		- project-alpha
		! md
		---

		workspace body
	EOM
	assert_success

	# Now workspace should be dirty
	run_dodder check-workspace dirty
	assert_success  # exit 0 = dirty
}

function check_workspace_dirty_clean_after_push { # @test
	bootstrap_workspace_repo parent workspace

	local parent_path
	parent_path="$(realpath ../parent)"

	# Create a new zettel in the workspace
	run_dodder new -edit=false - <<-EOM
		---
		# workspace zettel
		- project-alpha
		! md
		---

		workspace body
	EOM
	assert_success

	# Push to parent
	run_dodder push
	assert_success

	# After push, workspace should be clean again
	run_dodder check-workspace dirty
	assert_failure  # exit 1 = clean
}

function check_workspace_dirty_not_in_workspace { # @test
	# Run in a directory that is not a workspace
	copy_from_version "$DIR"

	run_dodder check-workspace dirty
	# exit 2 = not in a workspace-repo
	assert_failure
	[ "$status" -eq 2 ]
	assert_output --partial 'not in a workspace'
}

function check_workspace_dirty_quiet_output { # @test
	bootstrap_workspace_repo parent workspace

	# Clean workspace should produce no stdout
	run_dodder check-workspace dirty
	assert_output ''
}

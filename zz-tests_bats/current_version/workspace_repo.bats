#! /usr/bin/env bats

setup() {
	load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

	# for shellcheck SC2154
	export output
}

teardown() {
	chflags_nouchg
}

# bats file_tags=user_story:workspace,user_story:repo,user_story:direct

# Bootstrap a parent repo with content at the given directory
function bootstrap_parent {
	(
		mkdir -p "$1"
		pushd "$1" || exit 1
		run_dodder_init -repo_id . "parent-repo-id"

		run_dodder new -edit=false - <<-EOM
			---
			# first zettel
			- project-alpha
			! md
			---

			first zettel body
		EOM
		assert_success

		run_dodder new -edit=false - <<-EOM
			---
			# second zettel
			- project-alpha
			- priority-high
			! md
			---

			second zettel body
		EOM
		assert_success

		run_dodder new -edit=false - <<-EOM
			---
			# unrelated zettel
			- project-beta
			! md
			---

			unrelated zettel body
		EOM
		assert_success
	)
}

function workspace_repo_clone_pull_push { # @test
	parent="parent"
	bootstrap_parent "$parent"
	parent_path="$(realpath "$parent")"

	# --- Clone filtered subset into workspace-repo ---
	mkdir -p workspace
	pushd workspace || exit 1

	run_dodder clone \
		-encryption none \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-repo_id . \
		-lock-internal-files=false \
		-direct "$parent_path" \
		workspace-repo-id \
		+zettel,typ,etikett

	assert_success

	# Verify workspace has the cloned objects
	run_dodder init-workspace
	assert_success

	run_dodder show :z
	assert_success
	# Should have all 3 zettels (clone transfers matching objects)
	assert_output_unsorted --regexp - <<-'EOM'
		\[one/dos @blake2b256-.+ !md "second zettel" priority-high project-alpha]
		\[one/uno @blake2b256-.+ !md "first zettel" project-alpha]
		\[two/uno @blake2b256-.+ !md "unrelated zettel" project-beta]
	EOM

	# --- Create new content in workspace ---
	run_dodder new -edit=false - <<-EOM
		---
		# workspace-created zettel
		- project-alpha
		! md
		---

		created in workspace
	EOM
	assert_success

	# --- Push workspace changes back to parent ---
	run_dodder push -direct "$parent_path"
	assert_success

	# Verify parent received the new zettel
	pushd "$parent_path" || exit 1
	run_dodder show :z
	assert_success
	assert_output --partial 'workspace-created zettel'
	popd || exit 1

	# --- Add content in parent, pull into workspace ---
	(
		pushd "$parent_path" || exit 1
		run_dodder new -edit=false - <<-EOM
			---
			# parent-created zettel
			- project-alpha
			! md
			---

			created in parent after workspace clone
		EOM
		assert_success
	)

	run_dodder pull -direct "$parent_path"
	assert_success

	# Verify workspace received the new parent zettel
	run_dodder show :z
	assert_success
	assert_output --partial 'parent-created zettel'
}

function workspace_repo_clone_filtered_by_tag { # @test
	parent="parent"
	bootstrap_parent "$parent"
	parent_path="$(realpath "$parent")"

	# --- Clone only project-alpha zettels into workspace-repo ---
	mkdir -p workspace
	pushd workspace || exit 1

	run_dodder clone \
		-encryption none \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-repo_id . \
		-lock-internal-files=false \
		-direct "$parent_path" \
		workspace-repo-id \
		project-alpha:z

	assert_success

	run_dodder init-workspace
	assert_success

	# Should have only project-alpha zettels, not project-beta
	run_dodder show :z
	assert_success
	assert_output_unsorted --regexp - <<-'EOM'
		\[one/dos @blake2b256-.+ !md "second zettel" priority-high project-alpha]
		\[one/uno @blake2b256-.+ !md "first zettel" project-alpha]
	EOM

	# Verify project-beta zettel was NOT cloned
	refute_output --partial 'project-beta'

	# Verify that referenced types were also cloned (edge expansion)
	run_dodder show :t
	assert_success
	assert_output --partial '!md'
}

function workspace_repo_pull_filtered_by_tag { # @test
	parent="parent"
	bootstrap_parent "$parent"
	parent_path="$(realpath "$parent")"

	# --- Clone only project-alpha zettels into workspace-repo ---
	mkdir -p workspace
	pushd workspace || exit 1

	run_dodder clone \
		-encryption none \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-repo_id . \
		-lock-internal-files=false \
		-direct "$parent_path" \
		workspace-repo-id \
		project-alpha:z

	assert_success

	run_dodder init-workspace
	assert_success

	# --- Add both project-alpha and project-beta zettels in parent ---
	(
		pushd "$parent_path" || exit 1
		run_dodder new -edit=false - <<-EOM
			---
			# new alpha zettel
			- project-alpha
			! md
			---

			new alpha body
		EOM
		assert_success

		run_dodder new -edit=false - <<-EOM
			---
			# new beta zettel
			- project-beta
			! md
			---

			new beta body
		EOM
		assert_success
	)

	# --- Pull with the same tag filter ---
	run_dodder pull -direct "$parent_path" project-alpha:z

	assert_success

	# Should have the new project-alpha zettel
	run_dodder show :z
	assert_success
	assert_output --partial 'new alpha zettel'

	# Should NOT have the new project-beta zettel
	refute_output --partial 'new beta zettel'
}

function workspace_repo_init_experimental_repo { # @test
	parent="parent"
	bootstrap_parent "$parent"
	parent_path="$(realpath "$parent")"

	mkdir -p workspace
	pushd workspace || exit 1

	run_dodder init-workspace \
		-experimental-repo \
		-encryption none \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-repo_id . \
		-lock-internal-files=false \
		-direct "$parent_path" \
		workspace-repo-id \
		project-alpha:z

	assert_success

	# Verify workspace has filtered objects + edge-expanded types
	run_dodder show :z
	assert_success
	assert_output_unsorted --regexp - <<-'EOM'
		\[one/dos @blake2b256-.+ !md "second zettel" priority-high project-alpha]
		\[one/uno @blake2b256-.+ !md "first zettel" project-alpha]
	EOM
	refute_output --partial 'project-beta'

	# Verify that referenced types were also pulled (edge expansion)
	run_dodder show :t
	assert_success

	# Verify .dodder-workspace was created with parent path
	assert [ -f .dodder-workspace ]
}

function workspace_repo_linked_zettel_ids_from_parent { # @test
	parent="parent"
	bootstrap_parent "$parent"
	parent_path="$(realpath "$parent")"

	# --- Create workspace WITHOUT explicit -yin/-yang ---
	# The workspace should automatically discover and use the parent's word lists
	mkdir -p workspace
	pushd workspace || exit 1

	run_dodder init-workspace \
		-experimental-repo \
		-encryption none \
		-repo_id . \
		-lock-internal-files=false \
		-direct "$parent_path" \
		workspace-repo-id \
		+zettel,typ,etikett

	assert_success

	# Verify workspace has the parent's zettels
	run_dodder show :z
	assert_success
	assert_output_unsorted --regexp - <<-'EOM'
		\[one/dos @blake2b256-.+ !md "second zettel" priority-high project-alpha]
		\[one/uno @blake2b256-.+ !md "first zettel" project-alpha]
		\[two/uno @blake2b256-.+ !md "unrelated zettel" project-beta]
	EOM

	# --- Create a new zettel in the workspace (uses parent's ID space) ---
	run_dodder new -edit=false - <<-EOM
		---
		# workspace zettel via linked ids
		- project-alpha
		! md
		---

		created in workspace without explicit yin/yang
	EOM
	assert_success

	# Verify the new zettel exists and got a valid ID
	run_dodder show :z
	assert_success
	assert_output --partial 'workspace zettel via linked ids'

	# --- Push back to parent ---
	run_dodder push -direct "$parent_path"
	assert_success

	# Verify parent received the workspace-created zettel
	pushd "$parent_path" || exit 1
	run_dodder show :z
	assert_success
	assert_output --partial 'workspace zettel via linked ids'
}

function workspace_repo_implicit_parent_push_pull { # @test
	parent="parent"
	bootstrap_parent "$parent"
	parent_path="$(realpath "$parent")"

	mkdir -p workspace
	pushd workspace || exit 1

	# Create workspace-repo with V1 config storing parent path
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

	# --- Create content in workspace ---
	run_dodder new -edit=false - <<-EOM
		---
		# implicit-push zettel
		- project-alpha
		! md
		---

		created for implicit push test
	EOM
	assert_success

	# --- Push WITHOUT -direct (should use stored parent path) ---
	run_dodder push
	assert_success

	# Verify parent received the new zettel
	pushd "$parent_path" || exit 1
	run_dodder show :z
	assert_success
	assert_output --partial 'implicit-push zettel'
	popd || exit 1

	# --- Add content in parent ---
	(
		pushd "$parent_path" || exit 1
		run_dodder new -edit=false - <<-EOM
			---
			# implicit-pull zettel
			- project-alpha
			! md
			---

			created for implicit pull test
		EOM
		assert_success
	)

	# --- Pull WITHOUT -direct (should use stored parent path) ---
	run_dodder pull
	assert_success

	# Verify workspace received the new parent zettel
	run_dodder show :z
	assert_success
	assert_output --partial 'implicit-pull zettel'
}

function workspace_repo_init_experimental_repo_existing_repo { # @test
	parent="parent"
	bootstrap_parent "$parent"
	parent_path="$(realpath "$parent")"

	mkdir -p workspace
	pushd workspace || exit 1

	run_dodder init-workspace \
		-experimental-repo \
		-encryption none \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-repo_id . \
		-lock-internal-files=false \
		-direct "$parent_path" \
		workspace-repo-id \
		project-alpha:z

	assert_success

	# --- Second init should fail (repo already exists) ---
	run_dodder init-workspace \
		-experimental-repo \
		-encryption none \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-repo_id . \
		-lock-internal-files=false \
		-direct "$parent_path" \
		workspace-repo-id-2 \
		project-alpha:z

	assert_failure
}

function workspace_repo_stale_parent_path { # @test
	parent="parent"
	bootstrap_parent "$parent"
	parent_path="$(realpath "$parent")"

	mkdir -p workspace
	pushd workspace || exit 1

	run_dodder init-workspace \
		-experimental-repo \
		-encryption none \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-repo_id . \
		-lock-internal-files=false \
		-direct "$parent_path" \
		workspace-repo-id \
		project-alpha:z

	assert_success

	# --- Delete the parent repo ---
	rm -rf "$parent_path"

	# --- Push should fail with meaningful error ---
	run_dodder push
	assert_failure

	# --- Pull should fail with meaningful error ---
	run_dodder pull
	assert_failure
}

function workspace_repo_experimental_repo_implies_cwd { # @test
	# When -experimental-repo is used, the workspace repo should always be
	# CWD-based, even without an explicit -repo_id flag. This test sets HOME to
	# a temp dir with an existing XDG dodder repo, proving that without implicit
	# CWD routing the command collides with the home repo.
	parent="parent"
	bootstrap_parent "$parent"
	parent_path="$(realpath "$parent")"

	# Create an existing XDG dodder repo under a fake HOME so the home
	# directory already has an inventory_lists_log.
	fake_home="$BATS_TEST_TMPDIR/fake_home"
	mkdir -p "$fake_home"
	(
		export HOME="$fake_home"
		export XDG_DATA_HOME="$fake_home/.local/share"
		export XDG_CONFIG_HOME="$fake_home/.config"
		export XDG_STATE_HOME="$fake_home/.local/state"
		export XDG_CACHE_HOME="$fake_home/.cache"
		export XDG_RUNTIME_HOME="$fake_home/.local/runtime"
		run_dodder init \
			-yin <(cat_yin) \
			-yang <(cat_yang) \
			-encryption none \
			-lock-internal-files=false \
			home-repo-id
		assert_success
	)

	# Now try init-workspace WITHOUT -repo_id from a different directory.
	# The command should implicitly use CWD, not HOME.
	mkdir -p workspace_no_flag
	pushd workspace_no_flag || exit 1

	export HOME="$fake_home"
	export XDG_DATA_HOME="$fake_home/.local/share"
	export XDG_CONFIG_HOME="$fake_home/.config"
	export XDG_STATE_HOME="$fake_home/.local/state"
	export XDG_CACHE_HOME="$fake_home/.cache"
	export XDG_RUNTIME_HOME="$fake_home/.local/runtime"
	run_dodder init-workspace \
		-experimental-repo \
		-encryption none \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-lock-internal-files=false \
		-direct "$parent_path" \
		workspace-repo-id \
		project-alpha:z

	assert_success

	# Verify workspace has the expected objects
	run_dodder show :z
	assert_success
	assert_output_unsorted --regexp - <<-'EOM'
		\[one/dos @blake2b256-.+ !md "second zettel" priority-high project-alpha]
		\[one/uno @blake2b256-.+ !md "first zettel" project-alpha]
	EOM

	# Verify it created a CWD-local repo, not a home repo
	assert [ -d .dodder ]
	assert [ -f .dodder-workspace ]
}

function workspace_repo_init_experimental_repo_empty_query { # @test
	parent="parent"
	bootstrap_parent "$parent"
	parent_path="$(realpath "$parent")"

	mkdir -p workspace
	pushd workspace || exit 1

	# Query for a tag that doesn't exist — should succeed with empty workspace
	run_dodder init-workspace \
		-experimental-repo \
		-encryption none \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-repo_id . \
		-lock-internal-files=false \
		-direct "$parent_path" \
		workspace-repo-id \
		nonexistent-tag:z

	assert_success

	# Workspace should have no zettels
	run_dodder show :z
	assert_success
	assert_output ''

	# Workspace config should exist
	assert [ -f .dodder-workspace ]
}

function workspace_repo_push_unfiltered { # @test
	parent="parent"
	bootstrap_parent "$parent"
	parent_path="$(realpath "$parent")"

	# --- Clone all objects into workspace-repo ---
	mkdir -p workspace
	pushd workspace || exit 1

	run_dodder clone \
		-encryption none \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-repo_id . \
		-lock-internal-files=false \
		-direct "$parent_path" \
		workspace-repo-id \
		+zettel,typ,etikett

	assert_success

	run_dodder init-workspace
	assert_success

	# --- Create zettels with different tags in workspace ---
	run_dodder new -edit=false - <<-EOM
		---
		# workspace alpha zettel
		- project-alpha
		! md
		---

		workspace alpha body
	EOM
	assert_success

	run_dodder new -edit=false - <<-EOM
		---
		# workspace gamma zettel
		- project-gamma
		! md
		---

		workspace gamma body
	EOM
	assert_success

	# --- Push ALL workspace changes to parent (no filter) ---
	run_dodder push -direct "$parent_path"
	assert_success

	# Verify parent received BOTH zettels regardless of original clone filter
	pushd "$parent_path" || exit 1
	run_dodder show :z
	assert_success
	assert_output --partial 'workspace alpha zettel'
	assert_output --partial 'workspace gamma zettel'
}

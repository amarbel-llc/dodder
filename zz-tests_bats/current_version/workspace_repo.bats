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

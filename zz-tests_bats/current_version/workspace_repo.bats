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

# #290: in an ExcludeDefaultType workspace repo, `new` with no -type must NOT
# produce a typeless object (bare `!`). Such an object later breaks push/import
# with `unsupported seq: "!"`. The fix rejects the creation up front, telling
# the user to pass -type.
function workspace_repo_new_without_type_rejected { # @test
	parent="parent"
	bootstrap_parent "$parent"
	parent_path="$(realpath "$parent")"

	mkdir -p workspace
	pushd workspace || exit 1

	run_dodder init-workspace \
		-encryption none \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-parent "$parent_path" \
		workspace-repo-id \
		project-alpha:z
	assert_success

	# No -type, and the workspace repo has no default type -> must fail
	# rather than silently commit a typeless object.
	run_dodder new -edit=false -description x
	assert_failure
	assert_line --index 0 \
		'no type given and repo has no default type; pass -type'

	# Passing an explicit type still works.
	run_dodder new -edit=false -description x -type md
	assert_success
}

# Bootstrap a parent repo with content at the given directory
function bootstrap_parent {
	(
		mkdir -p "$1"
		pushd "$1" || exit 1
		run_dodder_init

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
		-direct "$parent_path" \
		.default \
		+zettel,typ,etikett

	assert_success

	# Verify workspace has the cloned objects
	run_dodder init-workspace -experimental-repo=false
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
		-direct "$parent_path" \
		.default \
		project-alpha:z

	assert_success

	run_dodder init-workspace -experimental-repo=false
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
		-direct "$parent_path" \
		.default \
		project-alpha:z

	assert_success

	run_dodder init-workspace -experimental-repo=false
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
		-encryption none \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-parent "$parent_path" \
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

# Covers the #200 pointer-blob-store path. The workspace's blob store
# must be a TomlPointerV1 pointing at the parent repo's default
# blob store, not a freshly-initialized local store. The on-disk
# layout uses the bare workspace-repo-id (no leading dot — the dot
# is a Stringer-rendered location prefix, not part of the dir name).
function workspace_repo_init_pointer_to_parent { # @test
	parent="parent"
	bootstrap_parent "$parent"
	parent_path="$(realpath "$parent")"

	mkdir -p workspace
	pushd workspace || exit 1

	run_dodder init-workspace \
		-encryption none \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-parent "$parent_path" \
		workspace-repo-id \
		project-alpha:z

	assert_success

	# Pointer config exists at the bare workspace-name directory.
	assert [ -f ".madder/local/share/blob_stores/workspace-repo-id/blob_store-config" ]

	# No locally-initialized default store in the workspace.
	assert [ ! -d ".madder/local/share/blob_stores/default" ]

	# The pointer config records the parent's default-store path.
	run cat .madder/local/share/blob_stores/workspace-repo-id/blob_store-config
	assert_success
	assert_output --partial "$parent_path/.madder/local/share/blob_stores/default"

	# Reads through the pointer resolve — the parent's konfig blob
	# was the original bug case from #200 (was failing with
	# "Blob with id ... does not exist locally"). Config is read via
	# show-config (FDR 0020), which fetches the blob through the pointer.
	run_dodder show-config
	assert_success
}

# Negative case for #200: when the parent repo has no default blob
# store, init-workspace must cancel cleanly rather than write a
# dangling pointer.
function workspace_repo_init_pointer_parent_missing_blob_store { # @test
	parent="parent"
	bootstrap_parent "$parent"
	parent_path="$(realpath "$parent")"

	# Remove the parent's default blob store after bootstrap.
	rm -rf "$parent_path/.madder/local/share/blob_stores/default"

	mkdir -p workspace
	pushd workspace || exit 1

	run_dodder init-workspace \
		-encryption none \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-parent "$parent_path" \
		workspace-repo-id \
		project-alpha:z

	assert_failure
	assert_output --partial "parent repo has no default blob store"
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
		-encryption none \
		-parent "$parent_path" \
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
		-encryption none \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-parent "$parent_path" \
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
		-encryption none \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-parent "$parent_path" \
		workspace-repo-id \
		project-alpha:z

	assert_success

	# --- Second init should fail (repo already exists) ---
	run_dodder init-workspace \
		-encryption none \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-parent "$parent_path" \
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
		-encryption none \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-parent "$parent_path" \
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

function workspace_repo_repo_id_rejected_with_experimental_repo { # @test
	parent="parent"
	bootstrap_parent "$parent"
	parent_path="$(realpath "$parent")"

	mkdir -p workspace
	pushd workspace || exit 1

	# -repo_id should be rejected with -experimental-repo (default)
	run_dodder init-workspace \
		-encryption none \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-repo_id .default \
		-parent "$parent_path" \
		workspace-repo-id \
		project-alpha:z

	assert_failure
	assert_output --partial 'cannot be used with'
}

function workspace_repo_init_experimental_repo_empty_query { # @test
	parent="parent"
	bootstrap_parent "$parent"
	parent_path="$(realpath "$parent")"

	mkdir -p workspace
	pushd workspace || exit 1

	# Query for a tag that doesn't exist — should succeed with empty workspace
	run_dodder init-workspace \
		-encryption none \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-parent "$parent_path" \
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
		-direct "$parent_path" \
		.default \
		+zettel,typ,etikett

	assert_success

	run_dodder init-workspace -experimental-repo=false
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

function workspace_repo_init_missing_parent_fails { # @test
	mkdir -p workspace
	pushd workspace || exit 1

	run_dodder init-workspace \
		-encryption none \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-parent /nonexistent/path \
		workspace-repo-id

	assert_failure
	assert_output --partial 'no dodder repo found'
}

# Covers the implicit-default-genre path. With a bare query like
# `project-alpha` (no `:z` suffix), init-workspace's default genres
# should select the tagged objects directly — NOT the inventory_list
# snapshots that contain them. Edge expansion should still pull
# referenced types. See dodder issue #133.
function workspace_repo_init_bare_query_excludes_unrelated { # @test
	parent="parent"
	bootstrap_parent "$parent"
	parent_path="$(realpath "$parent")"

	mkdir -p workspace
	pushd workspace || exit 1

	# Bare project-alpha — NO `:z` suffix. Relies on init-workspace's
	# default genres being something other than InventoryList.
	run_dodder init-workspace \
		-encryption none \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-parent "$parent_path" \
		workspace-repo-id \
		project-alpha

	assert_success

	# Should have ONLY project-alpha zettels.
	run_dodder show :z
	assert_success
	assert_output_unsorted --regexp - <<-'EOM'
		\[one/dos @blake2b256-.+ !md "second zettel" priority-high project-alpha]
		\[one/uno @blake2b256-.+ !md "first zettel" project-alpha]
	EOM

	# project-beta zettels must NOT be present (would be if the bare
	# query matched inventory_list snapshots whose blobs contained
	# both project-alpha and project-beta zettels committed together).
	refute_output --partial 'project-beta'

	# Edge expansion should have brought in the referenced !md type.
	run_dodder show :t
	assert_success
	assert_output --partial '!md'
}

# Bootstrap a HOME repo (XDG-scoped at $XDG_DATA_HOME/dodder, NOT
# CWD-scoped) so init-workspace's implicit-parent path resolves to it.
# A bare `default` location (XDG-user scope) is what makes init target the
# XDG home location at repos/default, where the implicit-parent path resolves.
function bootstrap_home_repo {
	run_dodder init \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-encryption none \
		default
	assert_success

	run_dodder new -edit=false - <<-EOM
		---
		# home zettel
		- project-alpha
		! md
		---

		home zettel body
	EOM
	assert_success
}

# Exercises the implicit-home-parent write path: `init-workspace` with
# NO -parent resolves the home repo as parent and writes a TomlPointerV1
# blob store pointing at the home repo's default store. The existing
# pointer-store coverage (workspace_repo_init_pointer_to_parent, #200)
# only verifies READS through the pointer; this verifies that
# workspace-ORIGINATED writes (new/edit) land in the resolved parent
# store and survive a reindex + checkout round-trip. Mirrors the
# real-world `der init-workspace today` invocation (bare repo id, no
# -parent, no explicit -yin/-yang — those are linked from the parent).
function workspace_repo_implicit_parent_write_roundtrip { # @test
	# The workspace discovers the home repo by walking up to
	# $XDG_DATA_HOME; raise the ceiling above $PWD so the walk-up
	# isn't blocked (see run_dodder ceiling rationale).
	export DODDER_TEST_CEILING="$BATS_TEST_TMPDIR"
	export MADDER_TEST_CEILING="$BATS_TEST_TMPDIR"

	bootstrap_home_repo

	mkdir -p workspace
	pushd workspace || exit 1

	# Implicit parent: no -parent flag → home repo is the parent.
	# Signature is `init-workspace [flags] <workspace-repo-id> [query]`.
	run_dodder init-workspace \
		-encryption none \
		workspace-repo-id \
		project-alpha:z
	assert_success

	# The pointer config is written at the bare workspace-repo-id dir
	# and resolves to the home repo's default store under $XDG_DATA_HOME
	# (bare madder layout, no local/share — the home-repo branch of
	# setupParentPointerBlobStore).
	assert [ -f ".madder/local/share/blob_stores/workspace-repo-id/blob_store-config" ]
	run cat .madder/local/share/blob_stores/workspace-repo-id/blob_store-config
	assert_success
	assert_output --partial "$XDG_DATA_HOME/madder/blob_stores/default"

	# The pulled home zettel is visible.
	run_dodder show :z
	assert_success
	assert_output --partial 'home zettel'

	# --- Create a NEW object in the workspace. Its blob must land in
	# the resolved parent store, not a workspace-local store.
	run_dodder new -edit=false - <<-EOM
		---
		# workspace zettel
		- project-alpha
		! md
		---

		workspace zettel body
	EOM
	assert_success

	# The workspace-authored blob resolves through the pointer: madder
	# can read it from the parent's default store by digest. Extract the
	# new zettel's blob digest from `show`, then cat it from the parent
	# store directly.
	run_dodder show -format object-id-blob-digest one/dos
	assert_success
	# Output is "<object-id> <blob-digest>"; take the digest field.
	blob_sha="${output##* }"
	run_madder cat default "$blob_sha"
	assert_success
	# madder cat prints a "switched to blob-store-id" status line first.
	assert_output - <<-EOM
		switched to blob-store-id: default
		workspace zettel body
	EOM

	# No workspace-local default store was created — the bare
	# workspace .madder only holds the pointer dir, never a `default`.
	assert [ ! -d ".madder/local/share/blob_stores/default" ]

	# --- edit a WORKSPACE-authored object (not the parent's, which
	# would create a push conflict). The edited blob must round-trip.
	export EDITOR="bash -c 'echo \"edited workspace body\" > \"\$0\"'"
	run_dodder edit one/dos
	assert_success

	run_dodder show -format blob one/dos
	assert_success
	assert_output 'edited workspace body'

	# --- reindex rebuilds the index FROM blobs; if any workspace-authored
	# blob failed to land in the resolved store it would surface here as
	# "does not exist locally".
	run_dodder reindex
	assert_success

	# --- checkout the workspace-authored object after reindex.
	run_dodder checkout one/dos
	assert_success

	run cat one/dos.zettel
	assert_success
	assert_output --partial 'edited workspace body'
}

# #287b: init-workspace -parent pins the parent repo's pubkey into
# .dodder-workspace. The stored value must equal the parent's
# `info-repo pubkey` (both are StringWithFormat()).
function workspace_repo_init_pins_parent_pubkey { # @test
	parent="parent"
	bootstrap_parent "$parent"
	parent_path="$(realpath "$parent")"

	# Capture the parent's pubkey.
	pushd "$parent_path" || exit 1
	run_dodder info-repo pubkey
	assert_success
	parent_pubkey="$output"
	popd || exit 1

	mkdir -p workspace
	pushd workspace || exit 1

	run_dodder init-workspace \
		-encryption none \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-parent "$parent_path" \
		workspace-repo-id \
		+zettel,typ,etikett
	assert_success

	# The pin is recorded in the workspace config.
	assert [ -f .dodder-workspace ]
	run grep -F "parent-pubkey" .dodder-workspace
	assert_success
	assert_output --partial "$parent_pubkey"
}

# #287b: a push whose resolved parent's pubkey differs from the pinned one
# must hard-fail rather than silently transfer to the wrong target. The pin is
# tampered (set to a different, valid pubkey) so the parent's blob store
# pointer stays intact and the workspace still loads — isolating the identity
# check from store-init concerns.
function workspace_repo_push_parent_pubkey_mismatch_fails { # @test
	parent="parent"
	bootstrap_parent "$parent"
	parent_path="$(realpath "$parent")"

	# A second, independent repo whose pubkey we will pin as a wrong value.
	other="other"
	bootstrap_parent "$other"
	pushd "$other" || exit 1
	run_dodder info-repo pubkey
	assert_success
	other_pubkey="$output"
	popd || exit 1

	mkdir -p workspace
	pushd workspace || exit 1

	run_dodder init-workspace \
		-encryption none \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-parent "$parent_path" \
		workspace-repo-id \
		+zettel,typ,etikett
	assert_success

	# Tamper the pin in place: rewrite the parent-pubkey line's value to the
	# other repo's pubkey. The parent path (and its blob-store pointer) is
	# untouched, so the workspace still loads; only the identity check fails.
	# Done in awk to replace just that top-level line, keeping TOML structure.
	run awk -v repl="parent-pubkey = \"$other_pubkey\"" \
		'/^parent-pubkey =/ {print repl; next} {print}' .dodder-workspace
	assert_success
	echo "$output" >.dodder-workspace

	# Sanity: the tampered value is present.
	run grep -F "$other_pubkey" .dodder-workspace
	assert_success

	run_dodder push
	assert_failure
	assert_output --partial "parent repo identity mismatch"
}

# #287b: a legacy workspace (parent-path but no pinned pubkey) must hard-fail
# a non-interactive push, directing the user to set-parent. bats has no TTY,
# so this exercises the non-TTY branch.
function workspace_repo_push_unpinned_non_tty_fails { # @test
	parent="parent"
	bootstrap_parent "$parent"
	parent_path="$(realpath "$parent")"

	mkdir -p workspace
	pushd workspace || exit 1

	run_dodder init-workspace \
		-encryption none \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-parent "$parent_path" \
		workspace-repo-id \
		+zettel,typ,etikett
	assert_success

	# Simulate a legacy (pre-pinning) workspace: strip the pinned pubkey line.
	run grep -v "parent-pubkey" .dodder-workspace
	assert_success
	echo "$output" >.dodder-workspace

	run_dodder push
	assert_failure
	assert_output --partial "set-parent"
}

# #287b: set-parent pins the current parent for a legacy workspace, after
# which a non-interactive push succeeds (the pin now matches).
function workspace_repo_set_parent_pins_then_push_succeeds { # @test
	parent="parent"
	bootstrap_parent "$parent"
	parent_path="$(realpath "$parent")"

	mkdir -p workspace
	pushd workspace || exit 1

	run_dodder init-workspace \
		-encryption none \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-parent "$parent_path" \
		workspace-repo-id \
		+zettel,typ,etikett
	assert_success

	# Strip the pin to simulate a legacy workspace.
	run grep -v "parent-pubkey" .dodder-workspace
	assert_success
	echo "$output" >.dodder-workspace

	# set-parent re-pins the current parent.
	run_dodder set-parent
	assert_success
	assert_output --partial "pinned parent:"

	# A new object + bare push now succeeds against the (re-pinned) parent.
	run_dodder new -edit=false - <<-EOM
		---
		# repinned push zettel
		- project-alpha
		! md
		---

		created after re-pin
	EOM
	assert_success

	run_dodder push
	assert_success

	pushd "$parent_path" || exit 1
	run_dodder show :z
	assert_success
	assert_output --partial 'repinned push zettel'
}

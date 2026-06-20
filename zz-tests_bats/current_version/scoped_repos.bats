#! /usr/bin/env bats

setup() {
	load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

	# for shellcheck SC2154
	export output
}

teardown() {
	chflags_nouchg
}

# bats file_tags=user_story:scoped_repos

# FDR-0019 prototype: several user-scoped named repos coexist under
# $XDG_DATA_HOME/dodder/repos/<name>/ and are addressed by the name
# portion of -repo_id. Each repo has an independent metadata tree, so
# an object committed to one is invisible to the other.

# bats test_tags=user_story:scoped_repos,isolation
function scoped_user_repos_are_isolated { # @test
	run_dodder init -yin <(cat_yin) -yang <(cat_yang) -repo_id work work-id
	assert_success

	run_dodder init -yin <(cat_yin) -yang <(cat_yang) -repo_id personal personal-id
	assert_success

	# a zettel committed to "work"
	run_dodder new -repo_id work -edit=false
	assert_success
	assert_output - <<-EOM
		[one/uno !md]
	EOM

	# ...is visible in "work"
	run_dodder show -repo_id work :z
	assert_success
	assert_output - <<-EOM
		[one/uno !md]
	EOM

	# ...and absent from "personal"
	run_dodder show -repo_id personal :z
	assert_success
	assert_output ''

	# each repo reports its own genesis id
	run_dodder info-repo -repo_id work id
	assert_success
	assert_output 'work-id'

	run_dodder info-repo -repo_id personal id
	assert_success
	assert_output 'personal-id'
}

# bats test_tags=user_story:scoped_repos,env_var
function scoped_repo_addressed_via_env_var { # @test
	run_dodder init -yin <(cat_yin) -yang <(cat_yang) -repo_id work work-id
	assert_success

	DODDER_REPO_ID=work run_dodder info-repo id
	assert_success
	assert_output 'work-id'
}

# bats test_tags=user_story:scoped_repos,repo_id
function scoped_repo_multi_dot_resolves_nth_ancestor { # @test
	# FDR-0019 #281: nested same-named cwd repos. The operate path resolves
	# `.notes` (depth 0) to the nearest `.dodder/` hosting a `notes` repo, and
	# `..notes` (depth 1) to the next such ancestor up — store-aware via
	# directory_layout.ResolveNthAncestorMatch. The walk-up must reach above
	# the nesting, so raise the ceiling past $BATS_TEST_TMPDIR.
	export DODDER_TEST_CEILING="$(dirname "$BATS_TEST_TMPDIR")"
	export MADDER_TEST_CEILING="$(dirname "$BATS_TEST_TMPDIR")"

	# outer `notes` repo at $BATS_TEST_TMPDIR, with one zettel as its marker.
	run_dodder init -yin <(cat_yin) -yang <(cat_yang) -repo_id .notes outer-id
	assert_success
	run_dodder new -repo_id .notes -edit=false
	assert_success

	# inner `notes` repo one level down, left empty — so the two repos are
	# distinguishable by content (inner empty, outer has the zettel) without
	# committing into the nested repo. (A nested-cwd-repo commit currently
	# trips a separate, pre-existing blob-store bug; see #283.)
	mkdir -p inner
	pushd inner || exit 1
	run_dodder init -yin <(cat_yin) -yang <(cat_yang) -repo_id .notes inner-id
	assert_success

	# from inner: `.notes` (nearest match) is the inner repo — empty.
	run_dodder show -repo_id .notes :z
	assert_success
	assert_output ''

	# `..notes` (one matching ancestor up) is the outer repo — its zettel.
	run_dodder show -repo_id ..notes :z
	assert_success
	assert_output - <<-EOM
		[one/uno !md]
	EOM

	# `...notes` overflows — only two matching ancestors exist.
	run_dodder show -repo_id ...notes :z
	assert_failure
}

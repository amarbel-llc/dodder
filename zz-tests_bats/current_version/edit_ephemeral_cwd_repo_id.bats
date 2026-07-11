#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

  # for shellcheck SC2154
  export output
}

teardown() {
  chflags_nouchg
}

# bats file_tags=user_story:workspace,user_story:noworkspace,user_story:scoped_repos

# A CWD-scoped parent id (.name / ..name) is NOT supported for -ephemeral and is
# rejected up front with a clear error (#343 step 5). The ephemeral flow chdirs
# into the temp dir and re-pins the ceiling to it before the post-chdir resolver
# calls (pointer blob store, MakeParentRemote), so the cwd walk-up that DEFINES a
# cwd-scoped id no longer finds the ancestor .dodder/ — it would silently fall
# back to the home repo. Pre-chdir absolute resolution of the ancestor is tracked
# as a follow-up. XDG-user ids (edit_ephemeral_repo_id.bats) and -parent paths
# (edit_ephemeral.bats) are the supported ephemeral-parent forms.
function edit_ephemeral_cwd_repo_id_is_rejected { # @test
  # Raise the ceiling above $BATS_TEST_TMPDIR so the cwd repo IS resolvable
  # (proving the rejection is by policy, not because the repo can't be found).
  export DODDER_TEST_CEILING="$(dirname "$BATS_TEST_TMPDIR")"
  export MADDER_TEST_CEILING="$(dirname "$BATS_TEST_TMPDIR")"

  # A cwd-scoped repo `.notes` at $BATS_TEST_TMPDIR, seeded with one zettel.
  run_dodder init -yin <(cat_yin) -yang <(cat_yang) -encryption none .notes
  assert_success
  run_dodder new -repo_id .notes -edit=false - <<-EOM
		---
		# notes zettel
		! md
		---

		original body
	EOM
  assert_success

  # A bare subdir with no .dodder / .dodder-workspace of its own.
  mkdir -p elsewhere
  pushd elsewhere || exit 1

  export EDITOR="bash -c 'echo \"should not run\" > \"\$0\"'"
  run_dodder edit -repo_id .notes -ephemeral one/uno
  assert_failure
  assert_line --index 0 \
    'cwd-scoped -repo_id ".notes" is not supported for -ephemeral; use an XDG-user repo id (e.g. `work`) or -parent <path>'

  # Nothing persisted in the invocation directory.
  assert [ ! -e .dodder-workspace ]
  assert [ ! -e .dodder ]
}

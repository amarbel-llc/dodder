#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

  # for shellcheck SC2154
  export output
}

teardown() {
  chflags_nouchg
}

# bats file_tags=user_story:workspace,user_story:noworkspace

# Bootstrap an XDG-user repo NAMED `work` (addressable as `-repo_id work`),
# distinct from the `default` repo the ephemeral home-fallback would resolve.
# Using a non-default name is deliberate: it forces true repo-id resolution —
# the home fallback (XDG_DATA_HOME/dodder's default repo) does not contain
# one/uno, so the test only passes if -repo_id actually targets `work`.
function bootstrap_named_repo {
  run_dodder init \
    -yin <(cat_yin) \
    -yang <(cat_yang) \
    -encryption none \
    work
  assert_success

  run_dodder new -edit=false -repo_id work - <<-EOM
		---
		# work zettel
		- project-alpha
		! md
		---

		original body
	EOM
  assert_success
}

# `edit -ephemeral -repo_id <id>` resolves its parent repo by repo-id (the
# FDR-0019 / MCP scope mechanism) rather than by -parent path or cwd: it spins
# a temp repo-backed workspace against the repo the id resolves to, edits,
# pushes back, and leaves nothing behind in the invocation directory.
function edit_ephemeral_repo_id_targets_resolved_repo { # @test
  # The ephemeral workspace + the repo-id resolution both walk up to
  # $XDG_DATA_HOME; raise the ceiling above $PWD so neither is blocked.
  export DODDER_TEST_CEILING="$BATS_TEST_TMPDIR"
  export MADDER_TEST_CEILING="$BATS_TEST_TMPDIR"

  bootstrap_named_repo

  # A bare working directory with no .dodder / .dodder-workspace.
  mkdir -p elsewhere
  pushd elsewhere || exit 1

  export EDITOR="bash -c 'echo \"edited via repo-id\" > \"\$0\"'"
  run_dodder edit -repo_id work -ephemeral one/uno
  assert_success

  # The edit landed in the `work` repo (addressed by repo-id).
  run_dodder show -repo_id work -format blob one/uno
  assert_success
  assert_output 'edited via repo-id'

  # Nothing persisted in the invocation directory.
  assert [ ! -e .dodder-workspace ]
  assert [ ! -e .dodder ]
}

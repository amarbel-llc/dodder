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

# Bootstrap a parent repo with one editable zettel (mirrors
# workspace_repo.bats' bootstrap_parent, trimmed to a single object).
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

			original body
		EOM
    assert_success
  )
}

# `edit -ephemeral` from a directory with NO .dodder-workspace edits an
# object against a resolved parent repo: it spins a temp repo-backed
# workspace, checks the object out, runs the editor, commits, and pushes
# the change back to the parent — leaving no workspace/repo behind in the
# invocation directory.
function edit_ephemeral_pushes_change_to_parent { # @test
  parent="parent"
  bootstrap_parent "$parent"
  parent_path="$(realpath "$parent")"

  # A bare working directory: no .dodder / .dodder-workspace here.
  mkdir -p elsewhere
  pushd elsewhere || exit 1

  export EDITOR="bash -c 'echo \"edited via ephemeral\" > \"\$0\"'"
  run_dodder edit -ephemeral -parent "$parent_path" one/uno
  assert_success

  # The edit landed in the parent repo.
  pushd "$parent_path" || exit 1
  run_dodder show -format blob one/uno
  assert_success
  assert_output 'edited via ephemeral'
  popd || exit 1

  # Nothing persisted in the invocation directory.
  assert [ ! -e .dodder-workspace ]
  assert [ ! -e .dodder ]
}

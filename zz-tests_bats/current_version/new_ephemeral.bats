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

# Bootstrap a parent repo (mirrors edit_ephemeral.bats' bootstrap_parent).
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

# `new -ephemeral` from a directory with NO .dodder-workspace creates a new
# object against a resolved parent repo: it spins a temp repo-backed
# workspace, creates the object, pushes it back to the parent, and leaves no
# workspace/repo behind in the invocation directory.
function new_ephemeral_creates_object_in_parent { # @test
  parent="parent"
  bootstrap_parent "$parent"
  parent_path="$(realpath "$parent")"

  # A bare working directory: no .dodder / .dodder-workspace here.
  mkdir -p elsewhere
  pushd elsewhere || exit 1

  run_dodder new -ephemeral -parent "$parent_path" \
    -edit=false -type md -description "ephemeral new zettel"
  assert_success

  # The new zettel landed in the parent repo.
  pushd "$parent_path" || exit 1
  run_dodder show :z
  assert_success
  assert_output --partial 'ephemeral new zettel'
  popd || exit 1

  # Nothing persisted in the invocation directory.
  assert [ ! -e .dodder-workspace ]
  assert [ ! -e .dodder ]
}

# `new -ephemeral` WITHOUT -type inherits the parent repo's default type
# (FDR-0023 / #15): the ephemeral workspace config is seeded with the parent's
# resolved default type (its mutable⊕workspace overlay), so creating a zettel
# needs no explicit -type — just like a normal workspace `new` against the
# parent would. The parent from run_dodder_init has !md as its default type.
function new_ephemeral_inherits_parent_default_type { # @test
  parent="parent"
  bootstrap_parent "$parent"
  parent_path="$(realpath "$parent")"

  # A bare working directory: no .dodder / .dodder-workspace here.
  mkdir -p elsewhere
  pushd elsewhere || exit 1

  # No -type given — must inherit the parent's default (!md) rather than
  # failing with "no type given and repo has no default type; pass -type".
  run_dodder new -ephemeral -parent "$parent_path" \
    -edit=false -description "inherits parent default type"
  assert_success

  # The new zettel landed in the parent with the inherited !md type.
  pushd "$parent_path" || exit 1
  run_dodder show :z
  assert_success
  assert_output --partial '!md "inherits parent default type"'
  popd || exit 1
}

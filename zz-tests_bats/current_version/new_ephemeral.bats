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

  # The new zettel landed in the parent repo, alongside the bootstrap zettel.
  # run_dodder_init makes a fresh key per run, but box format shows no
  # signature, so this is deterministic.
  pushd "$parent_path" || exit 1
  run_dodder show :z
  assert_success
  assert_output_unsorted - <<-EOM
		[one/dos !md "ephemeral new zettel"]
		[one/uno @blake2b256-2tn6x9p8k8gpz9tx52m556gxjw6d3zlfqd9w55mul4rhvma0jfzqwazeqr !md "first zettel" project-alpha]
	EOM
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

  # The new zettel landed in the parent with the inherited !md type, alongside
  # the bootstrap zettel.
  pushd "$parent_path" || exit 1
  run_dodder show :z
  assert_success
  assert_output_unsorted - <<-EOM
		[one/dos !md "inherits parent default type"]
		[one/uno @blake2b256-2tn6x9p8k8gpz9tx52m556gxjw6d3zlfqd9w55mul4rhvma0jfzqwazeqr !md "first zettel" project-alpha]
	EOM
  popd || exit 1
}

# `new -ephemeral -edit=true` (the Alfred `der new` / `zn` create actions):
# create an empty zettel, open EDITOR on it, commit the edited body, then push
# it back to the parent. The bare `new` path defaults -edit=true and opens the
# editor, so this exercises the same code path the Alfred `zn` snippet and
# `der new` keyword drive — only migrated to -ephemeral instead of `cd`.
function new_ephemeral_edit_true_pushes_edited_body_to_parent { # @test
  parent="parent"
  bootstrap_parent "$parent"
  parent_path="$(realpath "$parent")"

  # A bare working directory: no .dodder / .dodder-workspace here.
  mkdir -p elsewhere
  pushd elsewhere || exit 1

  # Faithful fake editor: APPEND the body to the checked-out file, preserving
  # the hyphence metadata header (type + description) that `new -edit` wrote.
  # A clobbering `> "$0"` fake would drop that header; appending mirrors a real
  # user editing the body beneath it, so type + description + body all survive.
  export EDITOR="bash -c 'echo \"created via ephemeral edit\" >> \"\$0\"'"
  run_dodder new -ephemeral -parent "$parent_path" \
    -edit=true -type md -description "ephemeral edit zettel"
  assert_success

  # The new zettel — full metadata + editor-written body — landed in the parent
  # alongside the bootstrap zettel. run_dodder_init generates a fresh key per
  # run, so the type signature is non-deterministic — match it with --regexp.
  pushd "$parent_path" || exit 1
  run_dodder show -format text :z
  assert_success
  assert_output --regexp - <<-EOM
		---
		# ephemeral edit zettel
		! md@.*
		---

		created via ephemeral edit
		---
		# first zettel
		- project-alpha
		! md@.*
		---

		original body
	EOM
  popd || exit 1

  # Nothing persisted in the invocation directory.
  assert [ ! -e .dodder-workspace ]
  assert [ ! -e .dodder ]
}

# `new -ephemeral -organize -delete <file>` (the Alfred Move-to-Dodder file
# action): import an existing file as a zettel blob, run organize on it, delete
# the source file on success, then push the imported object back to the parent.
# Exercises file-import + organize + -delete together under the ephemeral
# temp-workspace lifecycle.
function new_ephemeral_organize_delete_imports_file_to_parent { # @test
  parent="parent"
  bootstrap_parent "$parent"
  parent_path="$(realpath "$parent")"

  # A bare working directory: no .dodder / .dodder-workspace here.
  mkdir -p elsewhere
  pushd elsewhere || exit 1

  # The file to move into dodder.
  printf 'imported via move-to-dodder\n' >note.md

  # organize opens EDITOR on the plan buffer; `true` accepts it unchanged so
  # the imported object commits as-is.
  export EDITOR="true"
  run_dodder new -ephemeral -parent "$parent_path" \
    -organize -delete -type md note.md
  assert_success

  # The source file was deleted after a successful import.
  assert [ ! -e note.md ]

  # The imported blob landed in the parent, alongside the bootstrap zettel's
  # body. show -format blob :z concatenates both blob bodies in
  # non-deterministic order, so match unsorted.
  pushd "$parent_path" || exit 1
  run_dodder show -format blob :z
  assert_success
  assert_output_unsorted - <<-EOM
		imported via move-to-dodder
		original body
	EOM
  popd || exit 1

  # Nothing persisted in the invocation directory.
  assert [ ! -e .dodder-workspace ]
  assert [ ! -e .dodder ]
}

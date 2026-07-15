#! /usr/bin/env bats

# bats file_tags=debug

# Debug repro for a real-world failure seen with
# `dodder new -repo_id default -ephemeral -edit=true` where the checkin
# failed with:
#   failed to write type lock for type: "!task"
#     ... failed to validate object: "-61756905600.0 (0000-12-31T19:03:58-04:56) ..."
# The garbage Tai (-61756905600.0 == Go's zero time.Time{}, year 1) suggests
# WriteFSItemToExternal (go/internal/lima/store_fs/main.go) is stomping the
# Zettel's real Tai with a zero filesystem mtime. This test tries to
# reproduce that using a type ("task") that has never had an object
# committed in the parent repo before -- unlike the existing
# new_ephemeral_edit_true_pushes_edited_body_to_parent test in
# new_ephemeral.bats, which uses "md", whose type object already exists from
# bootstrap_parent's first zettel.
#
# NOTE: an earlier version of this fake editor appended straight onto the
# scaffold's closing `---` line with no blank-line separator, which tripped
# an UNRELATED hyphence "missing blank line after closing boundary" parse
# error. Verified via a debug dump (cat -A) of the checked-out scaffold: for
# a brand-new empty zettel, `new -edit=true`'s scaffold ends right at the
# closing `---` with EOF -- no body section yet. A real editor user types
# the blank line themselves before the body, so the fake editor below does
# the same (printf a leading blank line, then the body).
#
# RESULT (see task tracker): with this fixed, the commit SUCCEEDS -- a
# brand-new type alone does not reproduce the original "failed to write
# type lock for type: \"!task\"" / garbage-Tai failure. That failure needs
# something else not yet identified (real nvim vs fake editor, a large
# pre-existing repo, or workspace state left over from a prior command in
# the same session).

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

  # for shellcheck SC2154
  export output
}

teardown() {
  chflags_nouchg
}

# Bootstrap a parent repo with only an !md zettel -- no !task type object has
# ever been committed here, mirroring the real-world repo where !task was
# presumably first introduced via this exact ephemeral-edit flow.
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

function new_ephemeral_edit_true_new_type_reproduces_bad_tai { # @test
  parent="parent"
  bootstrap_parent "$parent"
  parent_path="$(realpath "$parent")"

  # A bare working directory: no .dodder / .dodder-workspace here.
  mkdir -p elsewhere
  pushd elsewhere || exit 1

  # Faithful fake editor: append a blank line THEN the body, mirroring what a
  # real editor user would type after the closing boundary.
  export EDITOR="bash -c 'printf \"\n created via ephemeral edit\n\" >> \"\$0\"'"
  run_dodder new -ephemeral -parent "$parent_path" \
    -edit=true -type task -description "schedule graph analysis"
  assert_success

  # The new zettel -- new type, full metadata, editor-written body -- landed
  # in the parent. Commit succeeded cleanly: no type-lock error, no garbage
  # Tai. This is the negative result: a brand-new type by itself does not
  # reproduce the original bug (see file header).
  pushd "$parent_path" || exit 1
  run_dodder show -format text one/dos
  assert_success
  assert_output --regexp - <<-EOM
		---
		# schedule graph analysis
		@ blake2b256-.*
		! task@.*
		---
	EOM

  run_dodder show -format blob one/dos
  assert_success
  assert_output ' created via ephemeral edit'
  popd || exit 1

  assert [ ! -e .dodder-workspace ]
  assert [ ! -e .dodder ]
}

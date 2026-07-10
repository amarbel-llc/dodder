#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

  # for shellcheck SC2154
  export output

  copy_from_version "$DIR"
}

teardown() {
  chflags_nouchg
}

function new_empty_no_edit { # @test
  run_dodder new -edit=false
  assert_success
  assert_output - <<-EOM
		[two/uno !md]
	EOM
}

function new_count_3 { # @test
  run_dodder new -edit=false -count 3
  assert_success
  assert_output - <<-EOM
		[two/uno !md]
		[one/tres !md]
		[two/dos !md]
	EOM
}

function new_empty_edit { # @test
  export EDITOR="bash -c 'echo \"this is the body\" > \"\$0\"'"
  run_dodder new
  assert_success
  assert_output - <<-EOM
		[two/uno !md]
		[two/uno @blake2b256-w2uv3ams8736hqllgvzgf7563m34ycem40nf8sg3mkefnrd9m75s083p85]
	EOM
}

function can_duplicate_zettel_content { # @test
  expected="$(mktemp)"
  {
    echo ---
    echo "# bez"
    echo - et1
    echo - et2
    echo ! md
    echo ---
    echo
    echo the body
  } >"$expected"

  run_dodder new -edit=false "$expected"
  assert_success
  assert_output - <<-EOM
		[two/uno @blake2b256-vl6ghtv2jsxppshflt86ardlx55ctn8jswx8j59tnv8r99uhs63syxsruy !md "bez" et1 et2]
	EOM

  run_dodder new -edit=false "$expected"
  assert_success
  assert_output - <<-EOM
		[one/tres @blake2b256-vl6ghtv2jsxppshflt86ardlx55ctn8jswx8j59tnv8r99uhs63syxsruy !md "bez" et1 et2]
	EOM

  # when
  run_dodder show -format text two/uno
  assert_success
  assert_output - <<-EOM
		---
		# bez
		- et1
		- et2
		! md@$(get_fixture_type_sig)
		---

		the body
	EOM

  run_dodder show -format text one/tres
  assert_success
  assert_output - <<-EOM
		---
		# bez
		- et1
		- et2
		! md@$(get_fixture_type_sig)
		---

		the body
	EOM
}

function use_blob_digests { # @test
  run_madder write -format tap - <<-EOM
		  the blob
	EOM
  assert_success
  assert_output --partial 'ok 1 - blake2b256-t9kaw07x3c89sft5axwjhe8z76p6d2642qr5xc62j5a4zq49pmvqypsla0 -'

  run_dodder new -edit=false -shas blake2b256-t9kaw07x3c89sft5axwjhe8z76p6d2642qr5xc62j5a4zq49pmvqypsla0
  assert_success
  assert_output - <<-EOM
		[two/uno @blake2b256-t9kaw07x3c89sft5axwjhe8z76p6d2642qr5xc62j5a4zq49pmvqypsla0 !md]
	EOM

  the_blob2_digest="blake2b256-65lys7dm4vfkag9y5j2hqhnah45qnc0kqvpdc46dw2cw63974a5q40q7xg"
  run_madder write -format tap - <<-EOM
		  the blob2
	EOM
  assert_success
  assert_output --partial "ok 1 - $the_blob2_digest -"

  run_dodder new -edit=false -shas -type txt "$the_blob2_digest"
  assert_success
  assert_output - <<-EOM
		[!txt !toml-type-v2]
		[one/tres @$the_blob2_digest !txt]
	EOM

  run_dodder_stderr_unified new -edit=false -shas "$the_blob2_digest"
  assert_success
  assert_output --partial - <<-EOM
		blake2b256-65lys7dm4vfkag9y5j2hqhnah45qnc0kqvpdc46dw2cw63974a5q40q7xg appears in object already checked in (["one/tres"]). Ignoring
	EOM
}

# bats file_tags=user_story:workspace

function new_empty_no_edit_workspace { # @test
  run_dodder init-workspace -experimental-repo=false -tags workspace-tags
  assert_success

  run_dodder new -edit=false
  assert_success
  assert_output - <<-EOM
		[two/uno !md workspace-tags]
	EOM
}

function new_empty_edit_workspace { # @test
  run_dodder init-workspace -experimental-repo=false -tags workspace-tags
  assert_success

  export EDITOR="bash -c 'echo \"this is the body\" > \"\$0\"'"
  run_dodder new
  assert_success
  assert_output - <<-EOM
		[two/uno !md workspace-tags]
		      checked out [two/uno.zettel !md workspace-tags]
		[two/uno @blake2b256-w2uv3ams8736hqllgvzgf7563m34ycem40nf8sg3mkefnrd9m75s083p85]
	EOM

  run_dodder status .
  assert_success
  assert_output - <<-EOM
		             same [two/uno.zettel @blake2b256-w2uv3ams8736hqllgvzgf7563m34ycem40nf8sg3mkefnrd9m75s083p85]
	EOM
}

function new_zettel_file { # @test
  to_add="$(mktemp)"
  {
    echo "---"
    echo "# wow"
    echo "- ok"
    echo "! md"
    echo "---"
  } >"$to_add"

  run_dodder new -edit=false "$to_add"
  assert_success
  assert_output - <<-EOM
		[two/uno !md "wow" ok]
	EOM

  run_dodder show -format text two/uno:z
  assert_success
  assert_output --regexp - <<EOM
---
# wow
- ok
! md@.*
---
EOM
}

function new_zettel_stdin { # @test
  to_add="$(mktemp)"
  {
    echo "---"
    echo "# wow"
    echo "- ok"
    echo "! md"
    echo "---"
  } >"$to_add"

  run_dodder new -edit=false - <"$to_add"
  assert_success
  assert_output - <<-EOM
		[two/uno !md "wow" ok]
	EOM

  run_dodder show -format text two/uno:z
  assert_success
  assert_output --regexp - <<EOM
---
# wow
- ok
! md@.*
---
EOM
}

function new_zettel { # @test
  expected="$(mktemp)"
  {
    echo "---"
    echo "# wow"
    echo "- ok"
    echo "! md"
    echo "---"
  } >"$expected"

  run_dodder new -edit=false -description wow -tags ok
  assert_success
  assert_output - <<-EOM
		[two/uno !md "wow" ok]
	EOM

  run_dodder show -format text two/uno:z
  assert_success
  assert_output --regexp - <<EOM
---
# wow
- ok
! md@.*
---
EOM
}

# -object-id with a type id authors the type object directly, with the
# meta-type (!toml-type-v2) set automatically from the id's genre, and the
# -blob written as the type's TOML body.
function new_object_id_type_with_blob { # @test
  run_dodder new -edit=false -object-id '!task' -blob 'file-extension = "toml"
'
  assert_success
  assert_output - <<-EOM
		[!task @blake2b256-agj380zh3wwj6n65chear8e4ednrwdxesrh6ccszc7xtu4707ujqucnjth !toml-type-v2]
	EOM

  run_dodder show -format blob '!task:t'
  assert_success
  assert_output - <<-EOM
		file-extension = "toml"
	EOM
}

# -object-id with a bare tag id authors a tag object; its meta-type is the
# tag genre default (!toml-tag-v1).
function new_object_id_tag { # @test
  run_dodder new -edit=false -object-id some-tag
  assert_success
  assert_output - <<-EOM
		[some-tag !toml-tag-v1]
	EOM
}

# -object-id with an explicit zettel id honors that id instead of
# auto-assigning one.
function new_object_id_explicit_zettel { # @test
  run_dodder new -edit=false -object-id left/right -description wow
  assert_success
  assert_output - <<-EOM
		[left/right !md "wow"]
	EOM
}

# -blob without -object-id writes the body onto an auto-assigned zettel.
function new_blob_only_auto_zettel { # @test
  run_dodder new -edit=false -blob 'the body
'
  assert_success
  assert_output - <<-EOM
		[two/uno @blake2b256-vl6ghtv2jsxppshflt86ardlx55ctn8jswx8j59tnv8r99uhs63syxsruy !md]
	EOM
}

# -object-id names exactly one object, so combining it with -count > 1 is a
# bad request.
function new_object_id_rejects_count { # @test
  run_dodder new -edit=false -object-id '!task' -count 2
  assert_failure
  assert_line --index 0 \
    '-object-id / -blob cannot be combined with -count > 1'
}

# -object-id / -blob only apply on the no-positional-args path.
function new_object_id_rejects_positional { # @test
  to_add="$(mktemp)"
  {
    echo "---"
    echo "# wow"
    echo "! md"
    echo "---"
  } >"$to_add"

  run_dodder new -edit=false -object-id left/right "$to_add"
  assert_failure
  assert_line --index 0 \
    '-object-id / -blob cannot be combined with positional arguments'
}

# A non-zettel -object-id sets the meta-type from the genre, so an explicit
# -type would be silently overridden — reject the combination.
function new_object_id_type_rejects_explicit_type { # @test
  run_dodder new -edit=false -object-id '!task' -type '!toml-type-v2'
  assert_failure
  assert_line --index 0 \
    '-type cannot be combined with a non-zettel -object-id (!task); the meta-type is set automatically from the id'"'"'s genre'
}

# `new -organize <path>` imports the file, then opens organize on the created
# object. Regression guard for #345: `new` used to build the organize op with a
# zero-value orgie.Metadata, whose uninitialized OptionCommentSet.prototype made
# GetOptions panic ("Metadata not initalized"). EDITOR=true accepts the organize
# plan unchanged, so the imported object commits as-is.
function new_organize_imports_and_organizes { # @test
  to_add="$(mktemp)"
  {
    echo "---"
    echo "# organized"
    echo "! md"
    echo "---"
  } >"$to_add"

  export EDITOR="true"
  run_dodder new -organize "$to_add"
  assert_success
  assert_output - <<-EOM
		[two/uno !md "organized"]
		[one/tres !md "organized"]
	EOM
}

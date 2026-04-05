#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

  # for shellcheck SC2154
  export output

  copy_from_version "$DIR"

  export BATS_TEST_BODY=true
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

function new_empty_edit { # @test
  export EDITOR="/bin/bash -c 'echo \"this is the body\" > \"\$0\"'"
  run_dodder new
  assert_success
  assert_output - <<-EOM
		[two/uno !md]
		[two/uno @blake2b256-w2uv3ams8736hqllgvzgf7563m34ycem40nf8sg3mkefnrd9m75s083p85]
	EOM

  run_dodder status .
  assert_failure
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
		@ blake2b256-vl6ghtv2jsxppshflt86ardlx55ctn8jswx8j59tnv8r99uhs63syxsruy
		! md@$(get_fixture_type_sig)
		---
	EOM

  run_dodder show -format text one/tres
  assert_success
  assert_output - <<-EOM
		---
		# bez
		- et1
		- et2
		@ blake2b256-vl6ghtv2jsxppshflt86ardlx55ctn8jswx8j59tnv8r99uhs63syxsruy
		! md@$(get_fixture_type_sig)
		---
	EOM
}

function use_blob_digest { # @test
  run_dodder blob_store-write - <<-EOM
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
  run_dodder blob_store-write - <<-EOM
		  the blob2
	EOM
  assert_success
  assert_output --partial "ok 1 - $the_blob2_digest -"

  run_dodder new -edit=false -shas -type txt "$the_blob2_digest"
  assert_success
  assert_output - <<-EOM
		[!txt !toml-type-v1]
		[one/tres @$the_blob2_digest !txt]
	EOM

  run_dodder_stderr_unified new -edit=false -shas "$the_blob2_digest"
  assert_success
  assert_output --partial - <<-EOM
		blake2b256-65lys7dm4vfkag9y5j2hqhnah45qnc0kqvpdc46dw2cw63974a5q40q7xg appears in object already checked in (["one/tres"]). Ignoring
	EOM
}

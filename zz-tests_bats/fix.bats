#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/common.bash"

  # for shellcheck SC2154
  export output
}

function can_update_blob { # @test
  # setup
  wd="$(mktemp -d)"
  cd "$wd" || exit 1

  run_dodder_init_disable_age

  assert_success
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
  assert_output_unsorted - <<-EOM
		[one/uno @blake2b256-vl6ghtv2jsxppshflt86ardlx55ctn8jswx8j59tnv8r99uhs63syxsruy !md "bez" et1 et2]
	EOM

  run_dodder show -format text one/uno
  assert_success
  assert_output --regexp - <<-EOM
		---
		# bez
		- et1
		- et2
		! md@.*
		---
		
		the body
EOM

  # when
  new_akte="$(mktemp)"
  {
    echo the body but new
  } >"$new_akte"

  run_dodder checkin-blob -new-tags et3 one/uno "$new_akte"
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-2qwngrkkpcptsnphu6jcyrwmtpyxux0hmsg4pjfpsn0tr7yt732sgk5lza !md "bez" et3]
	EOM

  # then
  {
    echo ---
    echo "# bez"
    echo - et3
    echo ! md
    echo ---
    echo
    echo the body but new
  } >"$expected"

  run_dodder show -format text one/uno
  assert_success
  assert_output --regexp - <<-EOM
		---
		# bez
		- et3
		! md@.*
		---
		
		the body but new
EOM
}

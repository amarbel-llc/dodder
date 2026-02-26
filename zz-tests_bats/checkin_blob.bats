#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/common.bash"

  run_dodder_init_disable_age
  assert_success

  # for shellcheck SC2154
  export output
}

function checkin_blob_filepath { # @test
  run_dodder new -edit=false - <<-EOM
---
# bez
- et1
- et2
! md
---

the body
EOM
  assert_success
  assert_output_unsorted - <<-EOM
		[alpha/golf @blake2b256-vl6ghtv2jsxppshflt86ardlx55ctn8jswx8j59tnv8r99uhs63syxsruy !md "bez" et1 et2]
	EOM

  run_dodder show -format text alpha/golf:z
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
  new_blob="$(mktemp)"
  {
    echo the body but new
  } >"$new_blob"

  run_dodder checkin-blob -new-tags et3 alpha/golf "$new_blob"
  assert_success
  assert_output - <<-EOM
		[alpha/golf @blake2b256-2qwngrkkpcptsnphu6jcyrwmtpyxux0hmsg4pjfpsn0tr7yt732sgk5lza !md "bez" et3]
	EOM

  run_dodder show -format text alpha/golf:z
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

function checkin_blob_digest { # @test
  run_dodder new -edit=false - <<-EOM
---
# bez
- et1
- et2
! md
---

the body
EOM
  assert_success
  assert_output_unsorted - <<-EOM
		[alpha/golf @blake2b256-vl6ghtv2jsxppshflt86ardlx55ctn8jswx8j59tnv8r99uhs63syxsruy !md "bez" et1 et2]
	EOM

  run_dodder show -format text alpha/golf:z
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
  run_dodder blob_store-write <(echo the body but new)
  assert_success
  assert_output --partial 'ok 1 - blake2b256-2qwngrkkpcptsnphu6jcyrwmtpyxux0hmsg4pjfpsn0tr7yt732sgk5lza'

  run_dodder checkin-blob -new-tags et3 alpha/golf blake2b256-2qwngrkkpcptsnphu6jcyrwmtpyxux0hmsg4pjfpsn0tr7yt732sgk5lza
  assert_success
  assert_output - <<-EOM
		[alpha/golf @blake2b256-2qwngrkkpcptsnphu6jcyrwmtpyxux0hmsg4pjfpsn0tr7yt732sgk5lza !md "bez" et3]
	EOM

  run_dodder show -format text alpha/golf:z
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

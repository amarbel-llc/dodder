#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/common.bash"

  # for shellcheck SC2154
  export output

  copy_from_version "$DIR"
}

teardown() {
  chflags_and_rm
}

function revert_one_zettel { # @test
  run_dodder revert one/uno
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 !md "wow ok" tag-1 tag-2]
	EOM
}

function revert_all_zettels { # @test
  run_dodder revert :z
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 !md "wow ok" tag-1 tag-2]
	EOM
}

function revert_last { # @test
  # TODO fix issue with output
  skip
  run_dodder revert -last
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 !md "wow ok" tag-1 tag-2]
	EOM

  run_dodder last
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 !md "wow ok" tag-1 tag-2]
	EOM

  run_dodder show one/uno
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 !md "wow ok" tag-1 tag-2]
	EOM
}

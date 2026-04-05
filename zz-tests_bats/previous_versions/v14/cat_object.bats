#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

  # for shellcheck SC2154
  export output

  copy_from_version "$DIR"

  run_dodder_init_workspace
}

function cat_object_by_type_sig { # @test
  run_dodder show -format sig '!md:t'
  assert_success
  sig="$output"

  run_dodder cat-object "$sig"
  assert_success
  assert_output --partial '!md'
  assert_output --partial '!toml-type-v1'
}

function cat_object_by_zettel_sig { # @test
  run_dodder show -format sig 'one/uno'
  assert_success
  sig="$output"

  run_dodder cat-object "$sig"
  assert_success
  assert_output --partial 'one/uno'
  assert_output --partial '!md'
}

function cat_object_not_found { # @test
  run_dodder cat-object blake2b256-0000000000000000000000000000000000000000000000000000000000000000
  assert_failure
}

function cat_object_no_args { # @test
  run_dodder cat-object
  assert_failure
}

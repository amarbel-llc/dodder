#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

  # for shellcheck SC2154
  export output
}

function pack_list_no_archives { # @test
  run_dodder_init_disable_age
  assert_success

  run_madder pack-list
  assert_success
  refute_output --partial 'entries'
}

function pack_list_shows_archive_after_pack { # @test
  run_dodder_init_disable_age
  assert_success

  run_madder init-inventory-archive .archive
  assert_success

  run_madder write -format tap .archive <(echo pack-list-test-content)
  assert_success

  run_madder pack .archive
  assert_success

  run_madder pack-list .archive
  assert_success
  assert_output --partial '1 entries'
}

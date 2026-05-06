#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

  # for shellcheck SC2154
  export output
}

teardown() {
  teardown_repo
}

# bats file_tags=user_story:blob_store

function blob_store_sync_twice { # @test
  setup_repo
  run_madder init test
  assert_success
  assert_line --regexp '^ok 1 - init .*/\.madder/local/share/blob_stores/test/blob_store-config$'
  assert_line "1..1"

  run_madder sync
  assert_success
  assert_line "Successes: 16, Failures: 0, Ignored: 0, Total: 16"
  # 1 header + 16 JSON lines, one per blob in the v15 fixture's
  # default store.
  [[ ${#lines[@]} -eq 17 ]] ||
    fail "sync 1: expected 17 output lines, got ${#lines[@]}: ${output}"

  run_madder sync
  assert_success
  assert_line "Successes: 0, Failures: 0, Ignored: 16, Total: 16"
  [[ ${#lines[@]} -eq 17 ]] ||
    fail "sync 2: expected 17 output lines, got ${#lines[@]}: ${output}"
}

function blob_store_sync_cross_hash_multi_hash_destination { # @test
  run_dodder_init_disable_age
  assert_success

  # write a blob to the default (blake2b256) store
  run_madder write -format tap <(echo cross-hash-test)
  assert_success
  blake_sha="$(echo "$output" | grep -oP 'blake2b256-\S+' | head -1)"

  # init a second store with sha256 (TomlV2 stores are multi-hash by default)
  run_madder init -hash_type-id sha256 -encryption none .sha256
  assert_success

  # sync from default to sha256 store
  run_madder sync .default .sha256
  assert_success

  # verify the blob exists in the sha256 store under both digests
  run_madder cat-ids .sha256
  assert_success
  assert_output --partial "$blake_sha"

  # verify the blob content is readable from the sha256 store
  run_madder cat .sha256 "$blake_sha"
  assert_success
  assert_line "cross-hash-test"
}

function blob_store_sync_cross_hash_second_sync_skips { # @test
  run_dodder_init_disable_age
  assert_success

  # write a blob to the default (blake2b256) store
  run_madder write -format tap <(echo idempotent-test)
  assert_success

  # init a second store with sha256
  run_madder init -hash_type-id sha256 -encryption none .sha256
  assert_success

  # first sync
  run_madder sync .default .sha256
  assert_success

  # second sync should skip already-synced blobs
  run_madder sync .default .sha256
  assert_success
}

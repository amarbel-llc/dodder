#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

  # for shellcheck SC2154
  export output
}

function cat_fallback_reads_from_non_default_store { # @test
  run_dodder_init_disable_age
  assert_success

  # create a second local store (.secondary sorts after .default)
  run_dodder blob_store-init -encryption none -lock-internal-files=false .secondary
  assert_success

  # write blob only to .secondary
  run_dodder blob_store-write .secondary <(echo fallback-content)
  assert_success
  sha="$(echo "$output" | grep -oP 'blake2b256-\S+' | head -1)"

  # cat without explicit store — should fallback from .default to .secondary
  run_dodder blob_store-cat "$sha"
  assert_success
  assert_output "fallback-content"
}

function cat_fallback_across_multiple_stores { # @test
  run_dodder_init_disable_age
  assert_success

  # create two additional stores (.secondary and .tertiary both sort after .default)
  run_dodder blob_store-init -encryption none -lock-internal-files=false .secondary
  assert_success

  run_dodder blob_store-init -encryption none -lock-internal-files=false .tertiary
  assert_success

  # write blob only to .tertiary (skips .default and .secondary)
  run_dodder blob_store-write .tertiary <(echo tertiary-content)
  assert_success
  sha="$(echo "$output" | grep -oP 'blake2b256-\S+' | head -1)"

  # cat without explicit store — should find it in .tertiary after skipping others
  run_dodder blob_store-cat "$sha"
  assert_success
  assert_output "tertiary-content"
}

function cat_fallback_blob_not_found_anywhere { # @test
  run_dodder_init_disable_age
  assert_success

  run_dodder blob_store-init -encryption none -lock-internal-files=false .secondary
  assert_success

  # use a hash that doesn't exist in any store
  run_dodder blob_store-cat "blake2b256-0000000000000000000000000000000000000000000000000000000000000000"
  assert_failure
}

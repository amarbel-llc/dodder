#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

  # for shellcheck SC2154
  export output
}

function cat_ids { # @test
  run_dodder_init_disable_age
  assert_success

  run_dodder blob_store-cat-ids .default
  assert_success
  assert_output --partial "$(get_konfig_sha)"
}

function cat_ids_format_hex { # @test
  run_dodder_init_disable_age
  assert_success

  run_dodder blob_store-cat-ids --format=hex .default
  assert_success
  # every ID line should be hash-type prefix + hex characters
  while IFS= read -r line; do
    [[ $line =~ ^blake2b256-[0-9a-f]+$ ]] || [[ $line =~ ^blobs\ with ]]
  done <<<"$output"
}

function cat_ids_format_blech32 { # @test
  run_dodder_init_disable_age
  assert_success

  # blech32 is the default, explicit flag should produce same output
  run_dodder blob_store-cat-ids .default
  assert_success
  default_output="$output"

  run_dodder blob_store-cat-ids --format=blech32 .default
  assert_success
  assert_output "$default_output"
}

function encode_ids { # @test
  run_dodder_init_disable_age
  assert_success

  # get native IDs (filter out summary line)
  run_dodder blob_store-cat-ids .default
  assert_success
  native_ids="$(echo "$output" | grep '^blake2b256-' | sort)"

  # get hex IDs, extract just the hex portion, pipe through encode-ids
  run_dodder blob_store-cat-ids --format=hex .default
  assert_success
  hex_only="$(echo "$output" | grep '^blake2b256-' | sed 's/^blake2b256-//')"

  encoded="$(echo "$hex_only" | dodder blob_store-encode-ids blake2b256 | sort)"
  [[ $encoded == "$native_ids" ]]
}

function encode_ids_missing_hash_type { # @test
  run bash -c "echo 'deadbeef' | $(command -v dodder) blob_store-encode-ids"
  assert_failure
}

function cat_with_explicit_store { # @test
  run_dodder_init_disable_age
  assert_success

  run_dodder blob_store-init-inventory-archive .archive
  assert_success

  run_dodder blob_store-write .archive <(echo cat-store-test)
  assert_success
  assert_output --partial 'ok 1'

  # extract the blake2b256 hash from TAP output
  sha="$(echo "$output" | grep -oP 'blake2b256-\S+')"

  run_dodder blob_store-cat .archive "$sha"
  assert_success
  assert_output --partial "cat-store-test"
}

function cat_default_then_archive { # @test
  run_dodder_init_disable_age
  assert_success

  run_dodder blob_store-init-inventory-archive .archive
  assert_success

  run_dodder blob_store-write .archive <(echo archive-content)
  assert_success
  archive_sha="$(echo "$output" | grep -oP 'blake2b256-\S+')"

  # cat from default store (konfig sha), then switch to archive
  run_dodder blob_store-cat "$(get_konfig_sha)" .archive "$archive_sha"
  assert_success
  assert_output --partial "archive-content"
}

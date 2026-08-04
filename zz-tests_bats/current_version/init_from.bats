#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"
  load "$(dirname "$BATS_TEST_FILE")/../lib/clone.bash"
  export output

  set_xdg "$BATS_TEST_TMPDIR"
}

teardown() {
  chflags_nouchg
}

# init-from is the repo half of the RFC-0007 copy-migration pattern:
# a NEW repo with the current config flavor (fresh uuidv7 instance
# identity) that KEEPS the source's keypair and full object history,
# leaving the source byte-untouched.
function init_from_migrates_keeping_keys_fresh_uuid { # @test
  them="source"
  bootstrap "$them"

  pushd "$them" || exit 1

  run_dodder info-repo pubkey
  assert_success
  source_pubkey="$output"

  run_dodder info-repo instance-id
  assert_success
  assert_output --regexp '^uuidv7-'
  source_instance_id="$output"

  run_dodder show :z
  assert_success
  source_zettels="$output"

  popd || exit 1

  mkdir dest
  pushd dest || exit 1

  run_dodder init-from \
    -from "$(realpath "../$them")" \
    .default \
    +zettel,typ,etikett
  assert_success

  # Same keypair: the uuid is the identity, the pubkey its attestor —
  # a migrated repo is the same logical repo under the same keys.
  run_dodder info-repo pubkey
  assert_success
  assert_output "$source_pubkey"

  # Fresh instance identity: the migration's whole point.
  run_dodder info-repo instance-id
  assert_success
  assert_output --regexp '^uuidv7-'
  [[ $output != "$source_instance_id" ]] ||
    fail "migrated repo kept the source's instance-id: $output"

  # Full object history migrated.
  run_dodder show :z
  assert_success
  assert_output_unsorted "$source_zettels"

  popd || exit 1

  # Source untouched: identity and objects unchanged after migration.
  pushd "$them" || exit 1

  run_dodder info-repo instance-id
  assert_success
  assert_output "$source_instance_id"

  run_dodder show :z
  assert_success
  assert_output_unsorted "$source_zettels"
}

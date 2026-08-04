#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

  # for shellcheck SC2154
  export output

  copy_from_version "$DIR"
}

teardown() {
  chflags_nouchg
}

# TODO add a preview of what would be deleted
function deinit_force() { # @test
  run_dodder deinit -force
  assert_success
  assert_output - <<-EOM
	EOM

  run_dodder status
  assert_failure
  assert_output --partial - <<-EOM
		not in a dodder directory
	EOM

  # Reuse the physical write store that survives deinit: under the
  # write_through multi default no store is named .default anymore;
  # .default-local is where the previous repo's blobs live.
  run_dodder_init -blob_store-id .default-local .default

  run_dodder last
  assert_success
  assert_golden deinit_force_last
}

function deinit() { # @test
  run_dodder deinit
  assert_success
  assert_output --regexp - <<-EOM
		stdin is not a tty, unable to get permission to continue
		permission denied and -force not specified, aborting
	EOM
}

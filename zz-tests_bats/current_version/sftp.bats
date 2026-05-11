#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"
  export output

  # https://github.com/amarbel-llc/dodder/issues/118
  # dodder-test-sftp-server never prints its READY line under the test sandbox.
  skip "SFTP test server setup blocked by sandbox, see #118"

  start_sftp_server
}

teardown() {
  chflags_nouchg
  stop_sftp_server
}

# bats file_tags=sftp

function start_sftp_server {
  local remote_dir="$BATS_TEST_TMPDIR/remote-blobs"
  mkdir -p "$remote_dir"

  local server_bin="$DODDER_TEST_SFTP_SERVER"
  if [[ ! -x $server_bin ]]; then
    skip "dodder-test-sftp-server not found at $server_bin"
  fi

  coproc SFTP_SERVER {
    "$server_bin" -root "$remote_dir" 2>"$BATS_TEST_TMPDIR/sftp-server.log"
  }

  local ready_line
  read -r ready_line <&"${SFTP_SERVER[0]}"

  if [[ $ready_line =~ ^READY\ port=([0-9]+)\ known_hosts=(.+)$ ]]; then
    SFTP_PORT="${BASH_REMATCH[1]}"
    local known_hosts_entry="${BASH_REMATCH[2]}"
    echo "$known_hosts_entry" >"$BATS_TEST_TMPDIR/known_hosts"
    SFTP_KNOWN_HOSTS="$BATS_TEST_TMPDIR/known_hosts"
  else
    fail "dodder-test-sftp-server did not print expected READY line: $ready_line"
  fi

  export SFTP_PORT SFTP_KNOWN_HOSTS
}

function stop_sftp_server {
  if [[ -n ${SFTP_SERVER_PID:-} ]]; then
    kill "$SFTP_SERVER_PID" 2>/dev/null || true
    wait "$SFTP_SERVER_PID" 2>/dev/null || true
    unset SFTP_SERVER_PID
  fi
}

# Write a blob store config file directly to the remote directory so that the
# SFTP store can read it during initialize(). This simulates a properly
# initialized remote.
function init_remote_blob_store_config {
  local remote_dir="$BATS_TEST_TMPDIR/remote-blobs"
  local config_path="$remote_dir/blob_store-config"

  cat >"$config_path" <<'EOM'
---
! toml-blob_store_config-v3
---

hash_buckets = [2]
hash_type-id = "blake2b256"
encryption = []
compression-type = "none"
EOM
}

function init_sftp_explicit_store {
  local store_id="${1:-.sftp-test}"

  run_madder init-sftp-explicit \
    -host 127.0.0.1 \
    -port "$SFTP_PORT" \
    -user test \
    -password test \
    -remote-path "$BATS_TEST_TMPDIR/remote-blobs" \
    -known-hosts-file "$SFTP_KNOWN_HOSTS" \
    "$store_id"
  assert_success
}

function sftp_explicit_init_and_fsck { # @test
  run_dodder_init_disable_age
  assert_success

  init_remote_blob_store_config
  init_sftp_explicit_store

  # Write a blob to the SFTP store
  run_madder write -format tap .sftp-test <(echo "hello sftp")
  assert_success

  # Verify blob exists via fsck
  run_madder fsck -format tap .sftp-test
  assert_success
}

function sftp_known_hosts_rejects_wrong_key { # @test
  run_dodder_init_disable_age
  assert_success

  init_remote_blob_store_config

  # Create a bogus known_hosts file with a wrong key
  echo "[127.0.0.1]:${SFTP_PORT} ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=" \
    >"$BATS_TEST_TMPDIR/bogus_known_hosts"

  run_madder init-sftp-explicit \
    -host 127.0.0.1 \
    -port "$SFTP_PORT" \
    -user test \
    -password test \
    -remote-path "$BATS_TEST_TMPDIR/remote-blobs" \
    -known-hosts-file "$BATS_TEST_TMPDIR/bogus_known_hosts" \
    .sftp-bogus
  assert_success

  # The init succeeds (just writes config), but using the store should fail
  # because the host key doesn't match
  run_madder write -format tap .sftp-bogus <(echo "should fail")
  assert_failure
}

function sftp_remote_config_missing_errors { # @test
  run_dodder_init_disable_age
  assert_success

  # Don't init a remote blob store config — the remote directory is empty
  init_sftp_explicit_store .sftp-no-config

  # Using the store should fail with a descriptive error about missing config
  run_madder write -format tap .sftp-no-config <(echo "should fail")
  assert_failure
  assert_output --partial "remote blob store config missing"
}

function sftp_discover_infers_config { # @test
  run_dodder_init_disable_age
  assert_success

  init_remote_blob_store_config

  # Write a blob directly to the remote directory using the local default store
  # to seed content for discover to find
  init_sftp_explicit_store .sftp-seed
  run_madder write -format tap .sftp-seed <(echo "discover me")
  assert_success

  # Remove the remote config file so discover has to infer it
  rm "$BATS_TEST_TMPDIR/remote-blobs/blob_store-config"

  # Run init with --discover
  run_madder init-sftp-explicit \
    -host 127.0.0.1 \
    -port "$SFTP_PORT" \
    -user test \
    -password test \
    -remote-path "$BATS_TEST_TMPDIR/remote-blobs" \
    -known-hosts-file "$SFTP_KNOWN_HOSTS" \
    -discover \
    .sftp-discovered
  assert_success
  assert_output --partial "discovered config"

  # The remote config should now exist again
  [[ -f "$BATS_TEST_TMPDIR/remote-blobs/blob_store-config" ]]

  # Verify the discovered store works via fsck
  run_madder fsck -format tap .sftp-discovered
  assert_success
}

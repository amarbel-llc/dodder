#! /bin/bash -e

# SFTP test harness, ported from amarbel-llc/madder's
# zz-tests_bats/lib/sftp.bash. Sourced from madder's flake
# (amarbel-llc/madder#177) via the MADDER_TEST_SFTP_SERVER env var
# wired in go/bats.nix -- see that file's netCapExtraBinaries comment.
#
# Not yet consumed by haustoria_orgmode.bats: that file predates
# madder#177 and pins an older, incompatible plain "READY port=N
# known_hosts=..." handshake against the (deleted) dodder-local
# dodder-test-sftp-server. The current madder-test-sftp-server speaks
# the cookie-based handshake below instead.

# start_sftp_server spawns madder-test-sftp-server as a coproc,
# validates the handshake line, then exports SFTP_PORT and
# SFTP_KNOWN_HOSTS for the test body to use.
#
# Caller contract: must call from a setup() that has already loaded
# common.bash (so $BATS_TEST_TMPDIR is set and require_bin is in
# scope). Pair with stop_sftp_server in teardown().
start_sftp_server() {
  require_bin MADDER_TEST_SFTP_SERVER madder-test-sftp-server
  local sftp_bin="${MADDER_TEST_SFTP_SERVER:-madder-test-sftp-server}"

  local cookie
  cookie="$(head -c 16 /dev/urandom | xxd -p)"

  local stderr_file="$BATS_TEST_TMPDIR/madder-test-sftp-server.stderr"

  # Named coproc: bash sets SFTP_PROC[0]=child-stdout-read-fd,
  # SFTP_PROC[1]=child-stdin-write-fd, and SFTP_PROC_PID=child pid.
  # Capture into stable scalar variables because array elements are
  # awkward inside `exec` redirections during teardown.
  coproc SFTP_PROC {
    MADDER_PLUGIN_COOKIE="$cookie" \
      "$sftp_bin" 2>"$stderr_file"
  }
  export SFTP_STDOUT_FD="${SFTP_PROC[0]}"
  export SFTP_STDIN_FD="${SFTP_PROC[1]}"
  export SFTP_PID="$SFTP_PROC_PID"

  local line
  if ! read -r -t 5 -u "$SFTP_STDOUT_FD" line; then
    local stderr_contents
    stderr_contents="$(cat "$stderr_file" 2>/dev/null || echo '<no stderr>')"
    fail "SFTP handshake timeout after 5s. stderr: $stderr_contents"
  fi

  local -a fields
  IFS='|' read -ra fields <<<"$line"
  if [[ ${#fields[@]} -ne 6 ]]; then
    fail "SFTP handshake malformed (want 6 fields, got ${#fields[@]}): $line"
  fi
  if [[ ${fields[0]} != "$cookie" ]]; then
    fail "SFTP handshake cookie mismatch: got ${fields[0]}, want $cookie"
  fi
  if [[ ${fields[1]} != "1" ]]; then
    fail "SFTP handshake version: got ${fields[1]}, want 1"
  fi

  export SFTP_PORT="${fields[3]##*:}"
  export SFTP_KNOWN_HOSTS="${fields[4]#known_hosts=}"
}

# init_sftp_test_store spins up a minimal SFTP-backed blob store
# under the running test SFTP server. Caller can override the remote
# directory and the local store id; both default to values that work
# for a single-store test scenario. A fresh remote gets a modern
# multi-hash blake2b256 config -- to reproduce a legacy single-hash
# remote (e.g. an sha256, non-multi-hash store), pre-seed remote_root
# with a harvested blob_store-config file before calling this, and
# init-sftp-explicit will detect and adopt the existing config instead
# of writing a fresh one.
init_sftp_test_store() {
  local remote_root="${1:-$BATS_TEST_TMPDIR/sftp-remote}"
  local store_id="${2:-.sftp-test}"

  run_madder init-sftp-explicit \
    -host 127.0.0.1 \
    -port "$SFTP_PORT" \
    -user testuser \
    -password anything \
    -remote-path "$remote_root" \
    -known-hosts-file "$SFTP_KNOWN_HOSTS" \
    "$store_id"
  assert_success
}

# stop_sftp_server closes the child's stdin (graceful shutdown
# signal), then reaps the child. Safe to call when start failed or
# when a previous teardown already ran.
stop_sftp_server() {
  if [[ -n ${SFTP_STDIN_FD:-} ]]; then
    eval "exec ${SFTP_STDIN_FD}>&-"
    unset SFTP_STDIN_FD
  fi
  if [[ -n ${SFTP_STDOUT_FD:-} ]]; then
    eval "exec ${SFTP_STDOUT_FD}<&-"
    unset SFTP_STDOUT_FD
  fi
  if [[ -n ${SFTP_PID:-} ]]; then
    wait "$SFTP_PID" 2>/dev/null || true
    unset SFTP_PID
  fi
  unset SFTP_PORT SFTP_KNOWN_HOSTS
}

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

# bats file_tags=haustoria,orgmode

function start_sftp_server {
  local remote_dir="$BATS_TEST_TMPDIR/remote-org"
  mkdir -p "$remote_dir/notes"

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

# Write a .org file to the remote SFTP directory.
function put_org_file {
  local folder="$1"
  local name="$2"
  local title="$3"
  local body="${4:-}"
  local tags="${5:-}"

  local remote_dir="$BATS_TEST_TMPDIR/remote-org"
  mkdir -p "$remote_dir/$folder"

  local tag_suffix=""
  if [[ -n $tags ]]; then
    tag_suffix=" :${tags}:"
  fi

  local content="* ${title}${tag_suffix}"

  if [[ -n $body ]]; then
    content="${content}
${body}"
  fi

  printf '%s\n' "$content" >"$remote_dir/$folder/$name.org"
}

# Write a .org file with a :PROPERTIES: drawer.
function put_org_file_with_properties {
  local folder="$1"
  local name="$2"
  local title="$3"
  local prop_key="$4"
  local prop_value="$5"
  local body="${6:-}"

  local remote_dir="$BATS_TEST_TMPDIR/remote-org"
  mkdir -p "$remote_dir/$folder"

  local content="* ${title}
:PROPERTIES:
:${prop_key}: ${prop_value}
:END:"

  if [[ -n $body ]]; then
    content="${content}
${body}"
  fi

  printf '%s\n' "$content" >"$remote_dir/$folder/$name.org"
}

function bootstrap_orgmode_workspace {
  local parent_dir="$BATS_TEST_TMPDIR/parent"
  local workspace_dir="$BATS_TEST_TMPDIR/workspace"
  mkdir -p "$parent_dir" "$workspace_dir"

  pushd "$parent_dir" || return 1
  run_dodder_init_disable_age "test-parent"
  popd || return 1

  pushd "$workspace_dir" || return 1

  # init-workspace -haustoria orgmode defaults to webdav transport,
  # so provide a dummy URL to pass validation. We overwrite the config
  # with SFTP settings immediately after.
  export ORGMODE_WEBDAV_URL="http://dummy"

  run_dodder init-workspace \
    -haustoria orgmode \
    -parent "$parent_dir" \
    ${cmd_dodder_def[@]} \
    haustoria-ws
  assert_success

  unset ORGMODE_WEBDAV_URL

  # Password is resolved from env var, not stored in config.
  export ORGMODE_SFTP_PASSWORD="test"

  # Prevent SSH agent fallback inside sandbox.
  unset SSH_AUTH_SOCK

  # Overwrite config with SFTP transport pointing at the test server.
  cat >.dodder-workspace <<EOF
---
! toml-workspace_config-v2
---

parent-path = "$parent_dir"

[defaults]
tags = []

[haustoria]
type = "orgmode"

[haustoria.orgmode]
transport = "sftp"

[haustoria.orgmode.sftp]
host = "127.0.0.1"
port = $SFTP_PORT
user = "test"
known-hosts-file = "$SFTP_KNOWN_HOSTS"

[haustoria.folders.notes]
path = "$BATS_TEST_TMPDIR/remote-org/notes"
type = "!md"
EOF

  popd || return 1
}

function status_shows_orgmode_files { # @test
  bootstrap_orgmode_workspace

  put_org_file "notes" "groceries" "Buy groceries" "milk eggs bread" "errands:shopping"
  put_org_file "notes" "dentist" "Call dentist" "schedule cleaning" "health"

  pushd "$BATS_TEST_TMPDIR/workspace" || return 1
  run_dodder status
  assert_success
  assert_output --partial "Buy groceries"
  assert_output --partial "Call dentist"
  assert_output --partial "untracked"
  popd || return 1
}

function checkin_creates_zettels_from_orgmode { # @test
  bootstrap_orgmode_workspace

  put_org_file "notes" "report" "Write report" "" "work"
  put_org_file "notes" "bike" "Fix bike" "" "errands"

  pushd "$BATS_TEST_TMPDIR/workspace" || return 1

  run_dodder checkin :
  assert_success

  run_dodder show :
  assert_success
  assert_output_unsorted --regexp - <<-EOM
		\[one/... !md "Write report" work\]
		\[one/... !md "Fix bike" errands\]
	EOM
  popd || return 1
}

function checkin_preserves_blob_from_body { # @test
  bootstrap_orgmode_workspace

  put_org_file "notes" "meeting" "Team standup" "discussed blockers and next steps"

  pushd "$BATS_TEST_TMPDIR/workspace" || return 1

  run_dodder checkin :
  assert_success

  # Verify the zettel was created with a blob (has a @blake2b256-... digest).
  run_dodder show :
  assert_success
  assert_output --partial "Team standup"
  assert_output --partial "@blake2b256-"
  popd || return 1
}

function checkin_idempotent_no_duplicates { # @test
  bootstrap_orgmode_workspace

  put_org_file "notes" "idempotent" "Idempotent task" "" "test"

  pushd "$BATS_TEST_TMPDIR/workspace" || return 1

  # First checkin creates the zettel.
  run_dodder checkin :
  assert_success

  run_dodder show :
  assert_success
  assert_output --partial "Idempotent task"

  # Second checkin should NOT create a duplicate.
  run_dodder checkin :
  assert_success

  run_dodder show :
  assert_success
  assert_output - <<-EOM
		[one/uno !md "Idempotent task" test]
	EOM
  popd || return 1
}

function status_shows_checked_out_after_checkin { # @test
  bootstrap_orgmode_workspace

  put_org_file "notes" "bound" "Bound task" "" "test"

  pushd "$BATS_TEST_TMPDIR/workspace" || return 1

  # Before checkin: Untracked.
  run_dodder status
  assert_success
  assert_output --partial "untracked"

  # Checkin binds the org file to a dodder object.
  run_dodder checkin :
  assert_success

  # After checkin: should show as checked out, not untracked.
  run_dodder status
  assert_success
  assert_output --partial "changed"
  refute_output --partial "untracked"
  popd || return 1
}

function checkin_empty_folder_no_error { # @test
  bootstrap_orgmode_workspace

  pushd "$BATS_TEST_TMPDIR/workspace" || return 1

  run_dodder checkin :
  assert_success

  run_dodder show :
  assert_success
  assert_output ""
  popd || return 1
}

function orgmode_properties_round_trip { # @test
  bootstrap_orgmode_workspace

  put_org_file_with_properties "notes" "props" "Task with props" "DODDER_ID" "one/uno" "the body"

  pushd "$BATS_TEST_TMPDIR/workspace" || return 1

  run_dodder status
  assert_success
  assert_output --partial "Task with props"
  popd || return 1
}

function orgmode_file_without_heading_uses_preamble { # @test
  bootstrap_orgmode_workspace

  # Write a plain text file without any org headings.
  local remote_dir="$BATS_TEST_TMPDIR/remote-org"
  printf 'Just a plain note\nwith two lines\n' >"$remote_dir/notes/plain.org"

  pushd "$BATS_TEST_TMPDIR/workspace" || return 1

  run_dodder status
  assert_success
  assert_output - <<-'EOM'
		        untracked [plain @blake2b256-k5jwcnwfnc2tadu5k32tfxywcn5nq3eyvqsehkncx8usj8jzacesr3gvq3 !md "Just a plain note"]
	EOM
  popd || return 1
}

function multiple_tags_from_orgmode { # @test
  bootstrap_orgmode_workspace

  put_org_file "notes" "multi" "Multi-tag task" "" "project:urgent:area-home"

  pushd "$BATS_TEST_TMPDIR/workspace" || return 1

  run_dodder checkin :
  assert_success

  run_dodder show :
  assert_success
  assert_output --partial "area-home"
  assert_output --partial "project"
  assert_output --partial "urgent"
  popd || return 1
}

function sftp_known_hosts_rejects_wrong_key { # @test
  local parent_dir="$BATS_TEST_TMPDIR/parent"
  local workspace_dir="$BATS_TEST_TMPDIR/workspace"
  mkdir -p "$parent_dir" "$workspace_dir"

  pushd "$parent_dir" || return 1
  run_dodder_init_disable_age "test-parent"
  popd || return 1

  pushd "$workspace_dir" || return 1

  # Create a bogus known_hosts file with a wrong key.
  echo "[127.0.0.1]:${SFTP_PORT} ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=" \
    >"$BATS_TEST_TMPDIR/bogus_known_hosts"

  export ORGMODE_WEBDAV_URL="http://dummy"
  run_dodder init-workspace \
    -haustoria orgmode \
    -parent "$parent_dir" \
    ${cmd_dodder_def[@]} \
    haustoria-ws
  unset ORGMODE_WEBDAV_URL

  export ORGMODE_SFTP_PASSWORD="test"
  unset SSH_AUTH_SOCK

  # Overwrite config to use SFTP with bogus known_hosts.
  cat >.dodder-workspace <<EOF
---
! toml-workspace_config-v2
---

parent-path = "$parent_dir"

[defaults]
tags = []

[haustoria]
type = "orgmode"

[haustoria.orgmode]
transport = "sftp"

[haustoria.orgmode.sftp]
host = "127.0.0.1"
port = $SFTP_PORT
user = "test"
known-hosts-file = "$BATS_TEST_TMPDIR/bogus_known_hosts"

[haustoria.folders.notes]
path = "$BATS_TEST_TMPDIR/remote-org/notes"
type = "!md"
EOF

  put_org_file "notes" "dummy" "Dummy task"

  # Status should fail because the host key doesn't match.
  run_dodder status
  assert_failure
  popd || return 1
}

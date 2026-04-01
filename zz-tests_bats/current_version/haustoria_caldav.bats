#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"
  export output

  start_radicale
}

teardown() {
  chflags_nouchg
  stop_radicale
}

# bats file_tags=haustoria,caldav

function start_radicale {
  local data_dir="$BATS_TEST_TMPDIR/radicale"
  mkdir -p "$data_dir/collections"

  radicale \
    --server-hosts "127.0.0.1:0" \
    --auth-type none \
    --storage-filesystem-folder "$data_dir/collections" \
    &>"$BATS_TEST_TMPDIR/radicale.log" &

  RADICALE_PID=$!

  # Discover the OS-assigned port from Radicale's log output.
  # Radicale logs: "Listening on '127.0.0.1:PORT'"
  local port=""
  local i
  for i in $(seq 1 30); do
    if [[ -f "$BATS_TEST_TMPDIR/radicale.log" ]]; then
      port=$(grep -oP "Listening on '127\.0\.0\.1:\K[0-9]+" "$BATS_TEST_TMPDIR/radicale.log" 2>/dev/null | head -1)
      if [[ -n $port ]]; then
        break
      fi
    fi
    sleep 0.1
  done

  if [[ -z $port ]]; then
    fail "Radicale failed to bind. Log: $(cat "$BATS_TEST_TMPDIR/radicale.log" 2>/dev/null)"
  fi

  RADICALE_PORT="$port"
  export CALDAV_URL="http://127.0.0.1:$port/dodder/tasks/"
  export CALDAV_USERNAME="dodder"
  export CALDAV_PASSWORD="dodder"

  # Wait for HTTP ready
  for i in $(seq 1 20); do
    if curl -s -o /dev/null "http://127.0.0.1:$port/" 2>/dev/null; then
      break
    fi
    sleep 0.1
  done

  # Create calendar
  local http_code
  http_code=$(curl -s -o /dev/null -w "%{http_code}" -X MKCALENDAR \
    -u "$CALDAV_USERNAME:$CALDAV_PASSWORD" \
    "$CALDAV_URL" \
    -H 'Content-Type: application/xml' \
    -d '<?xml version="1.0" encoding="UTF-8"?>
<mkcalendar xmlns="urn:ietf:params:xml:ns:caldav">
  <set xmlns="DAV:">
    <prop>
      <displayname>Tasks</displayname>
      <supported-calendar-component-set xmlns="urn:ietf:params:xml:ns:caldav">
        <comp name="VTODO"/>
      </supported-calendar-component-set>
    </prop>
  </set>
</mkcalendar>')
  if [[ $http_code -ge 400 ]]; then
    fail "MKCALENDAR $CALDAV_URL failed with HTTP $http_code on port $RADICALE_PORT. Log: $(cat "$BATS_TEST_TMPDIR/radicale.log" | tail -10)"
  fi
}

function stop_radicale {
  if [[ -n "${RADICALE_PID:-}" ]]; then
    kill "$RADICALE_PID" 2>/dev/null || true
    wait "$RADICALE_PID" 2>/dev/null || true
    unset RADICALE_PID
  fi
}

function put_vtodo {
  local uid="$1"
  local summary="$2"
  local categories="${3:-}"
  local description="${4:-}"

  local ical="BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//dodder-test//test//EN
BEGIN:VTODO
UID:$uid
SUMMARY:$summary
STATUS:NEEDS-ACTION"

  if [[ -n $categories ]]; then
    ical="$ical
CATEGORIES:$categories"
  fi

  if [[ -n $description ]]; then
    ical="$ical
DESCRIPTION:$description"
  fi

  ical="$ical
END:VTODO
END:VCALENDAR"

  local http_code
  http_code=$(curl -s -o /dev/null -w "%{http_code}" -X PUT \
    -u "$CALDAV_USERNAME:$CALDAV_PASSWORD" \
    -H "Content-Type: text/calendar" \
    "${CALDAV_URL}${uid}.ics" \
    -d "$ical")

  if [[ $http_code -ge 400 ]]; then
    fail "PUT VTODO $uid failed with HTTP $http_code"
  fi
}

function bootstrap_haustoria_workspace {
  local parent_dir="$BATS_TEST_TMPDIR/parent"
  local workspace_dir="$BATS_TEST_TMPDIR/workspace"
  mkdir -p "$parent_dir" "$workspace_dir"

  pushd "$parent_dir" || return 1
  run_dodder_init_disable_age "test-parent"
  popd || return 1

  pushd "$workspace_dir" || return 1
  run_dodder init-workspace \
    -haustoria caldav \
    -parent "$parent_dir" \
    ${cmd_dodder_def[@]} \
    haustoria-ws
  assert_success
  popd || return 1
}

function status_shows_caldav_resources { # @test
  put_vtodo "task-1" "Buy groceries" "errands,shopping" "milk eggs bread"
  put_vtodo "task-2" "Call dentist" "health" "schedule cleaning"

  bootstrap_haustoria_workspace

  pushd "$BATS_TEST_TMPDIR/workspace" || return 1
  run_dodder status
  assert_success
  assert_output_unsorted - <<-'EOM'
		        untracked [task-1 @blake2b256-sfghaeqe5dwr74mqpkttk402jua5um9ql3jgcdgpmvqx6znsf0ws8scpua !task "Buy groceries" errands shopping]
		        untracked [task-2 @blake2b256-eac5sj9j602ktwhy9zv355rm7nt82gqsyn525fkj2wxvqe4vuevqcy9n57 !task "Call dentist" health]
	EOM
  popd || return 1
}

function checkin_creates_zettels_from_caldav { # @test
  put_vtodo "task-1" "Write report" "work"
  put_vtodo "task-2" "Fix bike" "errands"

  bootstrap_haustoria_workspace

  pushd "$BATS_TEST_TMPDIR/workspace" || return 1

  run_dodder checkin :
  assert_success

  run_dodder show :
  assert_success
  assert_output_unsorted - <<-EOM
		[one/dos !task "Write report" work]
		[one/uno !task "Fix bike" errands]
	EOM
  popd || return 1
}

function checkout_decompiles_zettels_to_caldav { # @test
  # First: create VTODOs on CalDAV and checkin to create dodder objects
  put_vtodo "task-1" "Deploy v2.0" "ops" "rollback plan ready"
  put_vtodo "task-2" "Update docs" "docs"

  bootstrap_haustoria_workspace

  pushd "$BATS_TEST_TMPDIR/workspace" || return 1

  run_dodder checkin :
  assert_success

  # Delete the CalDAV tasks so checkout has to recreate them
  curl -s -X DELETE \
    -u "$CALDAV_USERNAME:$CALDAV_PASSWORD" \
    "${CALDAV_URL}task-1.ics" >/dev/null 2>&1
  curl -s -X DELETE \
    -u "$CALDAV_USERNAME:$CALDAV_PASSWORD" \
    "${CALDAV_URL}task-2.ics" >/dev/null 2>&1

  # Verify CalDAV is now empty
  run_dodder status
  assert_success
  assert_output ""

  # Checkout decompiles dodder objects back to CalDAV
  run_dodder checkout :z
  assert_success

  # Verify VTODOs were recreated on CalDAV
  run_dodder status
  assert_success
  assert_output --partial "Deploy v2.0"
  assert_output --partial "Update docs"
  popd || return 1
}

function checkin_empty_calendar_no_error { # @test
  bootstrap_haustoria_workspace

  pushd "$BATS_TEST_TMPDIR/workspace" || return 1

  run_dodder checkin :
  assert_success

  run_dodder show :
  assert_success
  assert_output ""
  popd || return 1
}

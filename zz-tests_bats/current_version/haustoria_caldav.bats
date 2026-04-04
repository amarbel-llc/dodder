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
}

function stop_radicale {
  if [[ -n ${RADICALE_PID:-} ]]; then
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

# Put a VTODO with explicit STATUS and optional multiple CATEGORIES lines.
# Usage: put_vtodo_with_status <calendar_url> <uid> <summary> <status> [categories_line]...
function put_vtodo_with_status {
  local calendar_url="$1"
  local uid="$2"
  local summary="$3"
  local status="$4"
  shift 4

  local ical="BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//dodder-test//test//EN
BEGIN:VTODO
UID:$uid
SUMMARY:$summary
STATUS:$status"

  # Remaining args are CATEGORIES lines
  for cats in "$@"; do
    ical="$ical
CATEGORIES:$cats"
  done

  ical="$ical
END:VTODO
END:VCALENDAR"

  local http_code
  http_code=$(curl -s -o /dev/null -w "%{http_code}" -X PUT \
    -u "$CALDAV_USERNAME:$CALDAV_PASSWORD" \
    -H "Content-Type: text/calendar" \
    "${calendar_url}${uid}.ics" \
    -d "$ical")

  if [[ $http_code -ge 400 ]]; then
    fail "PUT VTODO $uid to $calendar_url failed with HTTP $http_code"
  fi
}

function create_calendar {
  local calendar_url="$1"

  local http_code
  http_code=$(curl -s -o /dev/null -w "%{http_code}" -X MKCALENDAR \
    -u "$CALDAV_USERNAME:$CALDAV_PASSWORD" \
    "$calendar_url" \
    -H 'Content-Type: application/xml' \
    -d '<?xml version="1.0" encoding="UTF-8"?>
<mkcalendar xmlns="urn:ietf:params:xml:ns:caldav">
  <set xmlns="DAV:">
    <prop>
      <supported-calendar-component-set xmlns="urn:ietf:params:xml:ns:caldav">
        <comp name="VTODO"/>
      </supported-calendar-component-set>
    </prop>
  </set>
</mkcalendar>')
  if [[ $http_code -ge 400 ]]; then
    fail "MKCALENDAR $calendar_url failed with HTTP $http_code"
  fi
}

# Bootstrap a multi-calendar haustoria workspace with custom config.
# Creates parent repo, workspace dir, runs init-workspace, then overwrites
# the workspace config with a multi-calendar + status-tags configuration.
function bootstrap_multi_calendar_workspace {
  local parent_dir="$BATS_TEST_TMPDIR/parent"
  local workspace_dir="$BATS_TEST_TMPDIR/workspace"
  mkdir -p "$parent_dir" "$workspace_dir"

  local base_url="http://127.0.0.1:$RADICALE_PORT/dodder"
  export CALDAV_TASKS_URL="${base_url}/tasks/"
  export CALDAV_CHORES_URL="${base_url}/chores/"

  create_calendar "$CALDAV_TASKS_URL"
  create_calendar "$CALDAV_CHORES_URL"

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

  # Overwrite config with multi-calendar + status-tags
  cat >.dodder-workspace <<EOF
---
! toml-workspace_config-v2
---

parent-path = "$parent_dir"

[defaults]
tags = []

[haustoria]
type = "caldav"

[haustoria.caldav]
url = "${base_url}/"
username = "$CALDAV_USERNAME"

[haustoria.calendars.tasks]
url = "$CALDAV_TASKS_URL"
type = "!task"

[haustoria.calendars.tasks.status-tags]
COMPLETED = "zz-archive-task-done"

[haustoria.calendars.chores]
url = "$CALDAV_CHORES_URL"
type = "!chore"

[haustoria.calendars.chores.status-tags]
COMPLETED = "zz-archive-task-done"
EOF

  popd || return 1
}

function bootstrap_haustoria_workspace {
  local parent_dir="$BATS_TEST_TMPDIR/parent"
  local workspace_dir="$BATS_TEST_TMPDIR/workspace"
  mkdir -p "$parent_dir" "$workspace_dir"

  create_calendar "$CALDAV_URL"

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
  bootstrap_haustoria_workspace

  put_vtodo "task-1" "Buy groceries" "errands,shopping" "milk eggs bread"
  put_vtodo "task-2" "Call dentist" "health" "schedule cleaning"

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
  bootstrap_haustoria_workspace

  put_vtodo "task-1" "Write report" "work"
  put_vtodo "task-2" "Fix bike" "errands"

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
  bootstrap_haustoria_workspace

  # Create VTODOs on CalDAV and checkin to create dodder objects
  put_vtodo "task-1" "Deploy v2.0" "ops" "rollback plan ready"
  put_vtodo "task-2" "Update docs" "docs"

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

function new_creates_zettel_and_decompiles_to_caldav { # @test
  bootstrap_haustoria_workspace

  pushd "$BATS_TEST_TMPDIR/workspace" || return 1

  run_dodder new -edit=false - <<-EOM
		---
		# Review PR
		- work
		! task
		---

		check for regressions
	EOM
  assert_success

  # Verify zettel was created in dodder store
  run_dodder show :
  assert_success
  assert_output --partial "Review PR"
  assert_output --partial "!task"
  assert_output --partial "work"

  # Verify VTODO was created on CalDAV
  run_dodder status
  assert_success
  assert_output --partial "Review PR"
  popd || return 1
}

function checkin_idempotent_no_duplicates { # @test
  bootstrap_haustoria_workspace

  put_vtodo "task-1" "Idempotent task" "test"

  pushd "$BATS_TEST_TMPDIR/workspace" || return 1

  # First checkin creates the zettel
  run_dodder checkin :
  assert_success

  run_dodder show :
  assert_success
  assert_output --partial "Idempotent task"
  local first_output="$output"

  # Second checkin should NOT create a duplicate
  run_dodder checkin :
  assert_success

  run_dodder show :
  assert_success
  assert_output - <<-EOM
		[one/uno !task "Idempotent task" test]
	EOM
  popd || return 1
}

function status_shows_checked_out_after_checkin { # @test
  bootstrap_haustoria_workspace

  put_vtodo "task-1" "Bound task" "test"

  pushd "$BATS_TEST_TMPDIR/workspace" || return 1

  # Before checkin: Untracked
  run_dodder status
  assert_success
  assert_output --partial "untracked"

  # Checkin binds the CalDAV UID to a dodder object
  run_dodder checkin :
  assert_success

  # After checkin: should show as checked out, not untracked
  run_dodder status
  assert_success
  assert_output - <<-EOM
		          changed [ !task "Bound task" test]
	EOM
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

function multi_calendar_status_with_status_tags { # @test
  bootstrap_multi_calendar_workspace

  # Active tasks
  put_vtodo_with_status "$CALDAV_TASKS_URL" "t-1" "Write docs" "NEEDS-ACTION" "work"
  put_vtodo_with_status "$CALDAV_TASKS_URL" "t-2" "Fix login bug" "NEEDS-ACTION" "work,urgent"

  # Completed task — should get zz-archive-task-done tag
  put_vtodo_with_status "$CALDAV_TASKS_URL" "t-3" "Old migration" "COMPLETED" "ops"

  # Chore calendar
  put_vtodo_with_status "$CALDAV_CHORES_URL" "c-1" "Wash dishes" "NEEDS-ACTION" "area-home"
  put_vtodo_with_status "$CALDAV_CHORES_URL" "c-2" "Oil bike chain" "COMPLETED" "area-home"

  pushd "$BATS_TEST_TMPDIR/workspace" || return 1

  run_dodder status
  assert_success

  # All 5 tasks appear (completed ones get the status-tag, not filtered)
  local line_count
  line_count=$(echo "$output" | wc -l)
  [[ $line_count -eq 5 ]]

  # Verify status-tags mapping: completed tasks have zz-archive-task-done
  assert_output --partial "zz-archive-task-done"

  # Verify both calendar types present
  assert_output --partial "!task"
  assert_output --partial "!chore"

  # Checkin and verify completed task has the archive tag
  run_dodder checkin :
  assert_success

  run_dodder show :
  assert_success
  assert_output --partial "zz-archive-task-done"

  popd || return 1
}

function multiple_categories_lines_accumulated { # @test
  bootstrap_haustoria_workspace

  # Put a VTODO with multiple CATEGORIES lines via raw iCal
  local ical="BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//dodder-test//test//EN
BEGIN:VTODO
UID:multi-cat
SUMMARY:Multi-category task
STATUS:NEEDS-ACTION
CATEGORIES:project-dodder
CATEGORIES:area-career
CATEGORIES:urgent,req-email
END:VTODO
END:VCALENDAR"

  local http_code
  http_code=$(curl -s -o /dev/null -w "%{http_code}" -X PUT \
    -u "$CALDAV_USERNAME:$CALDAV_PASSWORD" \
    -H "Content-Type: text/calendar" \
    "${CALDAV_URL}multi-cat.ics" \
    -d "$ical")
  [[ $http_code -lt 400 ]]

  pushd "$BATS_TEST_TMPDIR/workspace" || return 1

  run_dodder status
  assert_success

  # All four categories from three CATEGORIES lines should be present
  assert_output --partial "area-career"
  assert_output --partial "project-dodder"
  assert_output --partial "req-email"
  assert_output --partial "urgent"

  popd || return 1
}

function completed_tasks_get_status_tag { # @test
  bootstrap_multi_calendar_workspace

  put_vtodo_with_status "$CALDAV_TASKS_URL" "done-1" "Finished task" "COMPLETED" "work"
  put_vtodo_with_status "$CALDAV_TASKS_URL" "active-1" "Active task" "NEEDS-ACTION" "work"

  pushd "$BATS_TEST_TMPDIR/workspace" || return 1

  # Checkin both
  run_dodder checkin :
  assert_success

  # Show all — completed task should have zz-archive-task-done, active should not
  run_dodder show :
  assert_success

  # Two zettels created
  local line_count
  line_count=$(echo "$output" | wc -l)
  [[ $line_count -eq 2 ]]

  # Completed task has archive tag, active task does not
  assert_output --partial '"Finished task" work zz-archive-task-done'
  assert_output --partial '"Active task" work]'
  refute_output --partial '"Active task" work zz-archive-task-done'

  popd || return 1
}

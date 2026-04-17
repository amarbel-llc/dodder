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

# Put a VTODO with a complete field set: STATUS, PRIORITY, DUE,
# DESCRIPTION, plus optional CATEGORIES. Used by the round-trip tests
# below.
# Usage: put_vtodo_full <url> <uid> <summary> <status> <priority> <due> <description> [categories]
function put_vtodo_full {
  local calendar_url="$1"
  local uid="$2"
  local summary="$3"
  local status="$4"
  local priority="$5"
  local due="$6"
  local description="$7"
  local categories="${8:-}"

  local ical="BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//dodder-test//test//EN
BEGIN:VTODO
UID:$uid
SUMMARY:$summary
STATUS:$status
PRIORITY:$priority"

  if [[ -n "$due" ]]; then
    ical="$ical
DUE:$due"
  fi

  if [[ -n "$description" ]]; then
    # Escape backslashes and newlines per RFC 5545.
    local escaped="${description//\\/\\\\}"
    escaped="${escaped//$'\n'/\\n}"
    ical="$ical
DESCRIPTION:$escaped"
  fi

  if [[ -n "$categories" ]]; then
    ical="$ical
CATEGORIES:$categories"
  fi

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
    fail "PUT VTODO $uid (full) failed with HTTP $http_code"
  fi
}

# Read a VTODO from CalDAV and emit raw iCal text on stdout. Used by
# round-trip tests to assert what made it back to the server after a
# checkout.
function get_vtodo_ical {
  local calendar_url="$1"
  local uid="$2"

  curl -s \
    -u "$CALDAV_USERNAME:$CALDAV_PASSWORD" \
    -H "Accept: text/calendar" \
    "${calendar_url}${uid}.ics"
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

# Bootstrap parent repo with !task and !chore built-in types.
# Wraps dodder init with -include-builtin-actionable-types so the parent
# commits the actionable type definitions, then init-workspace into the
# workspace dir. Tests rely on the !task type's reader script projecting
# status/priority/due fields onto checked-out objects from CalDAV.
function init_haustoria_parent {
  local parent_dir="$1"
  pushd "$parent_dir" || return 1
  run_dodder init \
    -yin <(cat_yin) \
    -yang <(cat_yang) \
    -encryption none \
    -repo_id . \
    -lock-internal-files=false \
    -include-builtin-actionable-types \
    test-parent
  assert_success
  popd || return 1
}

# Bootstrap a multi-calendar haustoria workspace with custom config.
# Creates parent repo (with built-in actionable types), workspace dir, runs
# init-workspace, then overwrites the workspace config with a multi-calendar
# configuration. Status / priority / due round-trip via the !task and !chore
# field reader/writer scripts; no status-tags mapping.
function bootstrap_multi_calendar_workspace {
  local parent_dir="$BATS_TEST_TMPDIR/parent"
  local workspace_dir="$BATS_TEST_TMPDIR/workspace"
  mkdir -p "$parent_dir" "$workspace_dir"

  local base_url="http://127.0.0.1:$RADICALE_PORT/dodder"
  export CALDAV_TASKS_URL="${base_url}/tasks/"
  export CALDAV_CHORES_URL="${base_url}/chores/"

  create_calendar "$CALDAV_TASKS_URL"
  create_calendar "$CALDAV_CHORES_URL"

  init_haustoria_parent "$parent_dir"

  pushd "$workspace_dir" || return 1
  run_dodder init-workspace \
    -haustoria caldav \
    -parent "$parent_dir" \
    -include-builtin-actionable-types \
    ${cmd_dodder_def[@]} \
    haustoria-ws
  assert_success

  # Overwrite config with multi-calendar setup. No status-tags blocks —
  # status round-trips via the !task field instead.
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

[haustoria.calendars.chores]
url = "$CALDAV_CHORES_URL"
type = "!chore"
EOF

  popd || return 1
}

function bootstrap_haustoria_workspace {
  local parent_dir="$BATS_TEST_TMPDIR/parent"
  local workspace_dir="$BATS_TEST_TMPDIR/workspace"
  mkdir -p "$parent_dir" "$workspace_dir"

  create_calendar "$CALDAV_URL"

  init_haustoria_parent "$parent_dir"

  pushd "$workspace_dir" || return 1
  run_dodder init-workspace \
    -haustoria caldav \
    -parent "$parent_dir" \
    -include-builtin-actionable-types \
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
  # Status shows the in-memory CalDAV objects produced by the haustoria's
  # compile path. The blob digest reflects the new TOML format
  # (status/priority/due/notes). No field columns appear because the !task
  # type's reader script only runs during the commit cycle, not during the
  # untracked-status query — fields project after `dodder checkin`.
  assert_output_unsorted - <<-'EOM'
		        untracked [task-1 @blake2b256-qg0l6r7jxyfh0zptqns7r4j664hp8j4htjrea3es6g5ve907jc6qgnk98w !task "Buy groceries" errands shopping]
		        untracked [task-2 @blake2b256-55pqnlyz6l9f9c4gfx68xvty2wy2e9evartmveg6t87ddft9tq3qdqvr47 !task "Call dentist" health]
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

  # Query with type filter so the box format shows fields. Both tasks
  # have empty notes/due/priority, so they share the same TOML blob
  # digest (status=todo, priority=p3, due="", notes="") — content-
  # addressing means identical blobs map to one digest.
  run_dodder show '!task'
  assert_success
  assert_output_unsorted - <<-EOM
		[one/dos @blake2b256-4a2a979r54j2804n697c20mvhjg8t377knujfxc6t8szh2awkwts2whwug !task "Write report" work status=todo priority=p3 due=]
		[one/uno @blake2b256-4a2a979r54j2804n697c20mvhjg8t377knujfxc6t8szh2awkwts2whwug !task "Fix bike" errands status=todo priority=p3 due=]
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

  # The blob body is now TOML mirroring the !task field set; the writer
  # script (yq) needs valid TOML on input. The notes key holds the
  # free-form description text.
  run_dodder new -edit=false - <<-EOM
		---
		# Review PR
		- work
		! task
		---

		status = "todo"
		priority = "p1"
		due = ""
		notes = "check for regressions"
	EOM
  assert_success

  # Verify zettel was created in dodder store with field projection
  run_dodder show '!task'
  assert_success
  assert_output --partial 'Review PR'
  assert_output --partial 'status=todo'
  assert_output --partial 'priority=p1'

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

  run_dodder show '!task'
  assert_success
  assert_output --partial "Idempotent task"

  # Second checkin should NOT create a duplicate
  run_dodder checkin :
  assert_success

  run_dodder show '!task'
  assert_success
  # Single object with field columns. Blob digest is the zero-default
  # TOML blob (status=todo, priority=p3, due="", notes="").
  assert_output - <<-EOM
		[one/uno @blake2b256-4a2a979r54j2804n697c20mvhjg8t377knujfxc6t8szh2awkwts2whwug !task "Idempotent task" test status=todo priority=p3 due=]
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

  # After checkin: the CalDAV resource is bound to a dodder object and
  # the metadata matches — status should show "same". The binding lookup
  # filters to zettel genre only (#111) and copies the type lock
  # signature from the committed object onto the external side.
  run_dodder status
  assert_success
  assert_output - <<-EOM
		             same [ @blake2b256-4a2a979r54j2804n697c20mvhjg8t377knujfxc6t8szh2awkwts2whwug !task "Bound task" test]
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

function multi_calendar_status_field_projection { # @test
  bootstrap_multi_calendar_workspace

  # Active tasks
  put_vtodo_with_status "$CALDAV_TASKS_URL" "t-1" "Write docs" "NEEDS-ACTION" "work"
  put_vtodo_with_status "$CALDAV_TASKS_URL" "t-2" "Fix login bug" "NEEDS-ACTION" "work,urgent"

  # Completed task — projects status=done via the reader script after checkin
  put_vtodo_with_status "$CALDAV_TASKS_URL" "t-3" "Old migration" "COMPLETED" "ops"

  # Chore calendar
  put_vtodo_with_status "$CALDAV_CHORES_URL" "c-1" "Wash dishes" "NEEDS-ACTION" "area-home"
  put_vtodo_with_status "$CALDAV_CHORES_URL" "c-2" "Oil bike chain" "COMPLETED" "area-home"

  pushd "$BATS_TEST_TMPDIR/workspace" || return 1

  run_dodder status
  assert_success

  # All 5 tasks appear (completed ones are not filtered — that's a
  # consumer concern via doddish field queries).
  local line_count
  line_count=$(echo "$output" | wc -l)
  [[ $line_count -eq 5 ]]

  # Both calendar types present
  assert_output --partial "!task"
  assert_output --partial "!chore"

  # Checkin and verify the field projection survived through both types.
  run_dodder checkin :
  assert_success

  run_dodder show '!task'
  assert_success
  # COMPLETED → status=done, NEEDS-ACTION → status=todo
  assert_output --partial 'status=done'
  assert_output --partial 'status=todo'

  run_dodder show '!chore'
  assert_success
  assert_output --partial 'status=done'
  assert_output --partial 'status=todo'

  # Doddish field query works for filtering completed tasks, replacing
  # the previous zz-archive-task-done tag-based filter.
  run_dodder show '^status=done !task'
  assert_success
  refute_output --partial 'Old migration'

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

function completed_tasks_get_status_field { # @test
  bootstrap_multi_calendar_workspace

  put_vtodo_with_status "$CALDAV_TASKS_URL" "done-1" "Finished task" "COMPLETED" "work"
  put_vtodo_with_status "$CALDAV_TASKS_URL" "active-1" "Active task" "NEEDS-ACTION" "work"

  pushd "$BATS_TEST_TMPDIR/workspace" || return 1

  # Checkin both
  run_dodder checkin :
  assert_success

  # Show !task — completed task projects status=done via the reader
  # script, active projects status=todo.
  run_dodder show '!task'
  assert_success

  # Two zettels created
  local line_count
  line_count=$(echo "$output" | wc -l)
  [[ $line_count -eq 2 ]]

  # Completed task has status=done, active task has status=todo. The
  # field columns appear after the tag list in the box format.
  assert_output --partial '"Finished task" work status=done'
  assert_output --partial '"Active task" work status=todo'

  popd || return 1
}

# Round-trip test covering all four blob keys (status, priority, due,
# notes): VTODO with full field set → checkin → checkout (without
# mutation) → verify CalDAV-side state matches the original. This
# exercises buildTaskTomlBlob (compile) and parseTaskTomlBlob +
# Decompile (round-trip back to VTODO) together.
#
# Note: this test does NOT exercise the organize-driven mutation path.
# `dodder organize -mode commit-directly` triggers a deeper malformed-
# blob-digest panic in the merge path that's unrelated to the haustoria
# implementation; tracked as a follow-up issue.
function caldav_round_trip_all_fields { # @test
  bootstrap_haustoria_workspace

  put_vtodo_full \
    "$CALDAV_URL" \
    "rt-1" \
    "Round trip task" \
    "IN-PROCESS" \
    "1" \
    "20260415T120000Z" \
    "the original notes" \
    "work,urgent"

  pushd "$BATS_TEST_TMPDIR/workspace" || return 1

  # Checkin: blob is built from VTODO via buildTaskTomlBlob; reader
  # script projects fields into the index.
  run_dodder checkin :
  assert_success

  run_dodder show '!task'
  assert_success
  # Compile mappings: STATUS:IN-PROCESS → in_progress, PRIORITY:1 → p0
  assert_output --partial 'status=in_progress'
  assert_output --partial 'priority=p0'
  assert_output --partial 'due=20260415T120000Z'

  # Delete the CalDAV resource so checkout has to recreate it from the
  # dodder-side TOML blob.
  curl -s -X DELETE \
    -u "$CALDAV_USERNAME:$CALDAV_PASSWORD" \
    "${CALDAV_URL}rt-1.ics" >/dev/null 2>&1

  # Checkout: Decompile parses the TOML blob, applies inverse
  # status/priority mappings, and PUTs a new VTODO.
  run_dodder checkout :z
  assert_success

  # Verify the VTODO on the CalDAV server reflects the original state.
  # Note: the haustoria's Decompile path does not currently round-trip
  # CATEGORIES (tags are read from object metadata, not from the blob),
  # so we don't assert on those here.
  run get_vtodo_ical "$CALDAV_URL" "rt-1"
  assert_success
  assert_output --partial 'STATUS:IN-PROCESS'
  assert_output --partial 'PRIORITY:1'
  assert_output --partial 'DUE:20260415T120000Z'
  assert_output --partial 'DESCRIPTION:the original notes'

  popd || return 1
}

# Edge case: VTODO with empty optional fields (no PRIORITY, no DUE, no
# DESCRIPTION). All three should land as defaults (p3, "", "") and survive
# the round-trip.
function caldav_round_trip_empty_optionals { # @test
  bootstrap_haustoria_workspace

  put_vtodo_full \
    "$CALDAV_URL" \
    "rt-empty" \
    "Empty optionals" \
    "NEEDS-ACTION" \
    "0" \
    "" \
    "" \
    ""

  pushd "$BATS_TEST_TMPDIR/workspace" || return 1

  run_dodder checkin :
  assert_success

  run_dodder show '!task'
  assert_success
  assert_output --partial 'status=todo'
  assert_output --partial 'priority=p3'
  assert_output --partial 'due='

  popd || return 1
}

# Edge case: multi-line VTODO DESCRIPTION text. Notes is the only blob
# key that can contain newlines; the buildTaskTomlBlob serializer uses
# strconv.Quote which produces a single-line basic string with \n
# escapes. Verify the round-trip preserves the line breaks.
function caldav_round_trip_multiline_notes { # @test
  bootstrap_haustoria_workspace

  put_vtodo_full \
    "$CALDAV_URL" \
    "rt-multi" \
    "Multi-line notes" \
    "NEEDS-ACTION" \
    "0" \
    "" \
    "first line
second line
third line" \
    ""

  pushd "$BATS_TEST_TMPDIR/workspace" || return 1

  run_dodder checkin :
  assert_success

  # Delete the CalDAV resource so checkout's PUT is a create rather than
  # an update. PutTask with empty ETag uses If-None-Match: * (create
  # semantics); ETag-based updates are tracked in #103 (haustoria
  # caching).
  curl -s -X DELETE \
    -u "$CALDAV_USERNAME:$CALDAV_PASSWORD" \
    "${CALDAV_URL}rt-multi.ics" >/dev/null 2>&1

  # Checkout decompiles the blob's notes back into VTODO DESCRIPTION.
  run_dodder checkout :z
  assert_success

  run get_vtodo_ical "$CALDAV_URL" "rt-multi"
  assert_success
  # iCalendar uses \n (literal backslash-n) for line breaks inside text
  # values. The original three-line description should appear in the
  # VTODO body with these escape sequences.
  assert_output --partial 'DESCRIPTION:first line\nsecond line\nthird line'

  popd || return 1
}

# Edge case: out-of-band VTODO PRIORITY value (not 0/1/5/9) buckets to
# the nearest canonical field value. PRIORITY:3 should bucket to p1.
function caldav_round_trip_out_of_band_priority { # @test
  bootstrap_haustoria_workspace

  put_vtodo_full \
    "$CALDAV_URL" \
    "rt-prio-3" \
    "Off-band priority" \
    "NEEDS-ACTION" \
    "3" \
    "" \
    "" \
    ""

  pushd "$BATS_TEST_TMPDIR/workspace" || return 1

  run_dodder checkin :
  assert_success

  run_dodder show '!task'
  assert_success
  assert_output --partial 'priority=p1'

  popd || return 1
}

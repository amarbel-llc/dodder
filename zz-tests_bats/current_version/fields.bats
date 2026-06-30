#! /usr/bin/env bats

# file_tags=user_story:fields

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

  # for shellcheck SC2154
  export output

  run_dodder_init_disable_age
}

teardown() {
  chflags_nouchg
}

function create_task_type {
  cat - >task.type <<-'EOM'
		file-extension = "toml"
		vim-syntax-type = "toml"

		[[fields]]
		name = "status"
		kind = "enum"
		values = ["todo", "in-progress", "blocked", "done", "cancelled"]
		default = "todo"

		[fields-reader]
		script = "yq -p toml -o json '{\"status\": .status}'"

		[fields-writer]
		script = "yq -p toml -o toml -i \".status = \\\"$DODDER_FIELD_status\\\"\" \"$DODDER_BLOB_PATH\""
	EOM

  run_dodder checkin -delete task.type
  assert_success
}

function create_task {
  local description="$1"
  local status="${2:-todo}"

  run_dodder new -edit=false - <<-EOM
		---
		# ${description}
		! task
		---

		status = "${status}"
	EOM
  assert_success
}

# Same as create_task_type but with NO fields-reader and NO fields-writer
# scripts. Only [[fields]] field definitions are present. Used to verify
# whether fields set directly on metadata.Index survive the commit cycle when
# tryWriteFields and tryReadFields both early-return at the script-nil gate.
# This is the codec behavior the haustoria depends on.
function create_task_type_no_scripts {
  cat - >task.type <<-'EOM'
		file-extension = "toml"
		vim-syntax-type = "toml"

		[[fields]]
		name = "status"
		kind = "enum"
		values = ["todo", "in-progress", "blocked", "done", "cancelled"]
		default = "todo"
	EOM

  run_dodder checkin -delete task.type
  assert_success
}

# Same as create_task_type but with ONLY a fields-reader (no fields-writer).
# Used to test whether organize can mutate a field when there's no writer
# script to project the new value back into the blob.
function create_task_type_reader_only {
  cat - >task.type <<-'EOM'
		file-extension = "toml"
		vim-syntax-type = "toml"

		[[fields]]
		name = "status"
		kind = "enum"
		values = ["todo", "in-progress", "blocked", "done", "cancelled"]
		default = "todo"

		[fields-reader]
		script = "yq -p toml -o json '{\"status\": .status}'"
	EOM

  run_dodder checkin -delete task.type
  assert_success
}

# Type with the FULL three-field set (status, priority, due) and both
# reader/writer scripts, mirroring what !task would ship with. Used to probe
# whether the haustoria-style flow (write a TOML blob with fields baked in,
# then commit) round-trips correctly.
function create_task_type_full {
  cat - >task.type <<-'EOM'
		file-extension = "toml"
		vim-syntax-type = "toml"

		[[fields]]
		name = "status"
		kind = "enum"
		values = ["todo", "in_progress", "done", "cancelled"]
		default = "todo"

		[[fields]]
		name = "priority"
		kind = "enum"
		values = ["p0", "p1", "p2", "p3"]
		default = "p3"

		[[fields]]
		name = "due"
		kind = "string"

		[fields-reader]
		script = "yq -p toml -o json '{\"status\": .status, \"priority\": .priority, \"due\": .due}'"

		[fields-writer]
		script = "yq -p toml -o toml -i \".status = \\\"$DODDER_FIELD_status\\\" | .priority = \\\"$DODDER_FIELD_priority\\\" | .due = \\\"$DODDER_FIELD_due\\\"\" \"$DODDER_BLOB_PATH\""
	EOM

  run_dodder checkin -delete task.type
  assert_success
}

function field_projection_on_commit { # @test
  create_task_type
  create_task "my first task" "todo"

  run_dodder show '!task'
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-qhdgjzc945w6v4pw4j4hr66gehgzwf8hq4v9rch9e7px5lf525sqfjcuaa !task "my first task" status=todo]
	EOM
}

function field_query_equality { # @test
  create_task_type
  create_task "task one" "todo"
  create_task "task two" "done"

  run_dodder show 'status=todo'
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-qhdgjzc945w6v4pw4j4hr66gehgzwf8hq4v9rch9e7px5lf525sqfjcuaa !task "task one" status=todo]
	EOM
}

function field_query_negation { # @test
  create_task_type
  create_task "task one" "todo"
  create_task "task two" "done"

  run_dodder show '^status=todo'
  assert_success
  assert_output - <<-EOM
		[one/dos @blake2b256-fh80v4xn6qv49r66rlkpkuvq2wlm4ztpuwjy6chexfjdrk0zhatqngn3sd !task "task two" status=done]
	EOM
}

function field_enum_validation_rejects_invalid { # @test
  create_task_type

  run_dodder new -edit=false - <<-EOM
		---
		# bad task
		! task
		---

		status = "invalid_value"
	EOM
  assert_failure
}

function field_organize_mutation { # @test
  create_task_type
  run_dodder init-workspace -experimental-repo=false
  create_task "my task" "todo"

  run_dodder organize -mode commit-directly '!task' <<-EOM
		- [one/uno !task status=done] my task
	EOM
  assert_success

  run_dodder show '!task'
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-fh80v4xn6qv49r66rlkpkuvq2wlm4ztpuwjy6chexfjdrk0zhatqngn3sd !task "my task" status=done]
	EOM
}

function field_organize_no_change { # @test
  create_task_type
  run_dodder init-workspace -experimental-repo=false
  create_task "my task" "todo"

  # same field value — blob should not change
  run_dodder organize -mode commit-directly '!task' <<-EOM
		- [one/uno !task status=todo] my task
	EOM
  assert_success

  run_dodder show '!task'
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-qhdgjzc945w6v4pw4j4hr66gehgzwf8hq4v9rch9e7px5lf525sqfjcuaa !task "my task" status=todo]
	EOM
}

function field_organize_with_tag_and_field_change { # @test
  create_task_type
  run_dodder init-workspace -experimental-repo=false
  create_task "my task" "todo"

  # change field AND add tag simultaneously
  run_dodder organize -mode commit-directly '!task' <<-EOM
		# urgent
		- [one/uno !task status=in-progress] my task
	EOM
  assert_success

  run_dodder show '!task'
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-0wcxp6kzf6g9vle04vfaqy6g7q75adcceyk8f5sml5dt09rckdxqrgpn8u !task "my task" urgent status=in-progress]
	EOM
}

function field_organize_description_and_field_change { # @test
  create_task_type
  run_dodder init-workspace -experimental-repo=false
  create_task "my task" "todo"

  # change field AND description simultaneously
  run_dodder organize -mode commit-directly '!task' <<-EOM
		- [one/uno !task status=done] updated description
	EOM
  assert_success

  run_dodder show '!task'
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-fh80v4xn6qv49r66rlkpkuvq2wlm4ztpuwjy6chexfjdrk0zhatqngn3sd !task "updated description" status=done]
	EOM
}

# Probe: when the type defines [[fields]] but provides NO fields-reader and NO
# fields-writer scripts, do field values set via organize survive the commit
# cycle? This is the haustoria CalDAV bridge's assumption: fields it sets on
# metadata.Index should persist via the binary stream-index codec without
# needing any script-based projection.
#
# RESULT: NO. The status field is dropped entirely. Organize-set field values
# do not survive commit when no fields-writer script is configured. The blob
# digest also disappears (the new value's blob is empty since no body was
# provided to dodder new).
#
# Implication: the haustoria CANNOT rely on setting Metadata.Index.Fields
# directly. Either ship a fields-writer script for !task that projects field
# values into a TOML blob, or add a code path that bypasses the script gate.
function field_persists_without_any_scripts { # @test
  create_task_type_no_scripts
  run_dodder init-workspace -experimental-repo=false

  run_dodder new -edit=false - <<-EOM
		---
		# my task
		! task
		---
	EOM
  assert_success

  run_dodder organize -mode commit-directly '!task' <<-EOM
		- [one/uno !task status=done] my task
	EOM
  assert_success

  run_dodder show '!task'
  assert_success
  # Locked-in: field is gone, blob digest is gone. Confirms fields require a
  # writer script to round-trip via organize.
  assert_output - <<-EOM
		[one/uno !task "my task"]
	EOM
}

# Probe: same question, but with a fields-reader script (so the initial blob
# → field projection works) and NO fields-writer.
#
# RESULT: the field is DUPLICATED in show output (`status=todo status=todo`)
# AND the user's organize-set value (`status=done`) is silently dropped — the
# reader re-projected from the unchanged blob and won. This is a bug in the
# organize/reader interaction independent of the haustoria question, but
# worth flagging.
function field_persists_with_reader_only_no_writer { # @test
  create_task_type_reader_only
  run_dodder init-workspace -experimental-repo=false
  create_task "my task" "todo"

  run_dodder organize -mode commit-directly '!task' <<-EOM
		- [one/uno !task status=done] my task
	EOM
  assert_success

  run_dodder show '!task'
  assert_success
  # Locked-in: duplicated field, user's edit dropped. Two regressions in one
  # output line. See dodder issue (to file).
  assert_output - <<-EOM
		[one/uno @blake2b256-qhdgjzc945w6v4pw4j4hr66gehgzwf8hq4v9rch9e7px5lf525sqfjcuaa !task "my task" status=todo status=todo]
	EOM
}

# Probe: full !task type with three fields and reader+writer. Create a task
# with a TOML blob that has all three field values baked in.
#
# RESULT: works. All three fields project from the blob via the reader script
# and appear in show output. This is the "option 2" haustoria flow.
function field_full_task_three_fields_from_blob { # @test
  create_task_type_full

  run_dodder new -edit=false - <<-EOM
		---
		# my task
		! task
		---

		status = "in_progress"
		priority = "p1"
		due = "20260415T120000Z"
	EOM
  assert_success

  run_dodder show '!task'
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-sehqwrekuhk346cppprq72mxk0qug5qfe6ghdqcd0tanf4plu6lsape8ag !task "my task" status=in_progress priority=p1 due=20260415T120000Z]
	EOM
}

# Probe: full !task type, organize-mutate one of three fields. Verifies the
# writer script projects the changed field while preserving the others.
#
# RESULT: works. status changes to "done", priority and due remain. The new
# blob digest is different (writer rewrote the TOML). Round-trip is clean.
function field_full_task_organize_mutate_one_of_three { # @test
  create_task_type_full
  run_dodder init-workspace -experimental-repo=false

  run_dodder new -edit=false - <<-EOM
		---
		# my task
		! task
		---

		status = "todo"
		priority = "p2"
		due = "20260415T120000Z"
	EOM
  assert_success

  run_dodder organize -mode commit-directly '!task' <<-EOM
		- [one/uno !task status=done priority=p2 due=20260415T120000Z] my task
	EOM
  assert_success

  run_dodder show '!task'
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-j0yn4cd0fvng3daxg0xrgjrag55v9l3gfsyhnawma3pm3szxpmtsgzaxs9 !task "my task" status=done priority=p2 due=20260415T120000Z]
	EOM
}

# Probe: full !task type, organize-set fields starting from an EMPTY blob.
# This simulates the haustoria's "first checkin if no blob is provided" case.
#
# RESULT: same as field_persists_without_any_scripts — fields are dropped,
# blob digest disappears. Even with a writer script configured, an empty
# starting blob means tryWriteFields can't run (the writer needs an existing
# blob path to mutate). Implication: the haustoria MUST write a non-empty
# starter blob before setting fields, OR write the full TOML blob with fields
# already baked in (option 2 — see field_full_task_three_fields_from_blob).
# A throwaway type carrying a `marker` string field and an on_commit_fields hook
# that MUTATES that field (untouched -> touched). Proves the RFC 0006 Phase 1
# commit-time field write-back mechanism end-to-end: the hook's mutation is
# persisted via the single bounded, hook-free write-back pass (tryWriteFields +
# tryReadFields), not just reflected in the in-memory index.
function create_marker_type {
  cat - >marker.type <<-'EOM'
		file-extension = "toml"
		vim-syntax-type = "toml"
		hooks = """
		return {
		  on_commit_fields = function(kinder, mutter)
		    local f = kinder.Fields
		    if f and f.marker == "untouched" then
		      kinder.Fields.marker = "touched"
		    end
		  end,
		}
		"""

		[[fields]]
		name = "marker"
		kind = "string"

		[fields-reader]
		script = "yq -p toml -o json '{\"marker\": .marker}'"

		[fields-writer]
		script = "yq -p toml -o toml -i \".marker = \\\"$DODDER_FIELD_marker\\\"\" \"$DODDER_BLOB_PATH\""
	EOM

  run_dodder checkin -delete marker.type
  assert_success
}

# Commit-time field mutation: the on_commit_fields hook rewrites marker from
# "untouched" to "touched", and the bounded write-back persists it. The commit
# COMPLETES (a cycle would hang or blow the stack), and `dodder show` reflects
# the rewritten value re-projected from the rewritten blob.
function field_hook_mutates_field_and_persists { # @test
  command -v yq >/dev/null || skip "yq not available"

  create_marker_type

  run_dodder new -edit=false - <<-EOM
		---
		# probe object
		! marker
		---

		marker = "untouched"
	EOM
  assert_success

  run_dodder show '!marker'
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-884d8x9xlt30hdrzm3f8wy5sd5a0x8g5qqufaqrvcrg9afx7u8xssurxag !marker "probe object" marker=touched]
	EOM
}

# Idempotency: re-committing the already-mutated object is stable. The hook sees
# marker already "touched", the untouched-guard is false, no field changes, so
# the bounded write-back is skipped and the object stays touched.
function field_hook_mutation_is_idempotent { # @test
  command -v yq >/dev/null || skip "yq not available"

  create_marker_type
  run_dodder init-workspace -experimental-repo=false

  run_dodder new -edit=false - <<-EOM
		---
		# probe object
		! marker
		---

		marker = "untouched"
	EOM
  assert_success

  # re-commit with the same (already-touched) value via organize
  run_dodder organize -mode commit-directly '!marker' <<-EOM
		- [one/uno !marker marker=touched] probe object
	EOM
  assert_success

  run_dodder show '!marker'
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-884d8x9xlt30hdrzm3f8wy5sd5a0x8g5qqufaqrvcrg9afx7u8xssurxag !marker "probe object" marker=touched]
	EOM
}

function field_full_task_organize_from_empty_blob { # @test
  create_task_type_full
  run_dodder init-workspace -experimental-repo=false

  run_dodder new -edit=false - <<-EOM
		---
		# my task
		! task
		---
	EOM
  assert_success

  run_dodder organize -mode commit-directly '!task' <<-EOM
		- [one/uno !task status=in_progress priority=p1 due=20260415T120000Z] my task
	EOM
  assert_success

  run_dodder show '!task'
  assert_success
  assert_output - <<-EOM
		[one/uno !task "my task"]
	EOM
}

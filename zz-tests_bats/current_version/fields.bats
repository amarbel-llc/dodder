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

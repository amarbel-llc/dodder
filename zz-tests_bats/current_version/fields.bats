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
		values = ["todo", "in-progress", "blocked", "completed", "cancelled"]
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
  assert_output --partial 'status=todo'
}

function field_query_equality { # @test
  create_task_type
  create_task "task one" "todo"
  create_task "task two" "completed"

  run_dodder show 'status=todo'
  assert_success
  assert_output --partial '"task one"'
  refute_output --partial '"task two"'
}

function field_query_negation { # @test
  # https://github.com/amarbel-llc/dodder/issues/98
  # Negated field query status^=todo returns empty.
  skip "negated field queries need investigation"
  create_task_type
  create_task "task one" "todo"
  create_task "task two" "completed"

  run_dodder show 'status^=todo'
  assert_success
  assert_output --partial '"task two"'
  refute_output --partial '"task one"'
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

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

function field_projection_on_commit { # @test
  create_task_type

  run_dodder new -edit=false - <<-EOM
		---
		# my first task
		! task
		---

		status = "todo"
		summary = "buy milk"
	EOM
  assert_success

  run_dodder show -format box '!task'
  assert_success
  assert_output --partial 'status="todo"'
}

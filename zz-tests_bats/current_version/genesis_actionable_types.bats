#! /usr/bin/env bats

# file_tags=user_story:builtin_types

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

  # for shellcheck SC2154
  export output
}

teardown() {
  chflags_nouchg
}

# Run dodder init with the standard fixture flags. Pass extra args (like
# -include-builtin-actionable-types) as positional arguments.
function init_fixture {
  run_dodder init \
    -yin <(cat_yin) \
    -yang <(cat_yang) \
    -encryption none \
    -repo_id . \
    "$@" \
    test-repo-id
  assert_success
}

# Without the opt-in flag, dodder init does NOT commit !task or !chore.
# Only the existing !md default type is present.
function genesis_default_omits_actionable_types { # @test
  init_fixture

  run_dodder show ':t'
  assert_success
  assert_output --regexp - <<-EOM
		\[!md @blake2b256-.+ !toml-type-v2]
	EOM
}

# With -include-builtin-actionable-types, !task and !chore are committed
# during genesis alongside !md, each with the actionable field set
# (status, priority, due) and yq reader/writer scripts.
function genesis_includes_actionable_types_when_opted_in { # @test
  init_fixture -include-builtin-actionable-types

  run_dodder show ':t'
  assert_success
  assert_output_unsorted --regexp - <<-EOM
		\[!chore @blake2b256-.+ !toml-type-v2]
		\[!md @blake2b256-.+ !toml-type-v2]
		\[!task @blake2b256-.+ !toml-type-v2]
	EOM
}

# The !task type blob exposes the actionable field definitions and the
# yq reader/writer scripts. Verifies the type blob format the haustoria
# depends on.
function genesis_task_type_blob_has_fields_and_scripts { # @test
  init_fixture -include-builtin-actionable-types

  run_dodder show -format blob '!task:t'
  assert_success
  # Note: tommy's TOML encoder serializes script as a triple-quoted multiline
  # string and unescapes backslashes (the Script field has the `multiline`
  # tommy tag). The Go-side definition uses double-quoted string literals
  # with escaped backslashes; these become triple-quoted multilines on disk.
  assert_output - <<-'EOM'
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
		script = """
		yq -p toml -o json '{"status": .status, "priority": .priority, "due": .due}'"""

		[fields-writer]
		script = """
		yq -p toml -o toml -i ".status = \"$DODDER_FIELD_status\" | .priority = \"$DODDER_FIELD_priority\" | .due = \"$DODDER_FIELD_due\"" "$DODDER_BLOB_PATH""""
	EOM
}

# Verifies the field round-trip via `dodder new`: after init with the
# opt-in flag, a freshly-created !task with a TOML body has all three
# fields projected by the reader script and visible in `dodder show`.
function genesis_dodder_new_task_projects_fields { # @test
  init_fixture -include-builtin-actionable-types
  run_dodder init-workspace -experimental-repo=false

  run_dodder new -edit=false - <<-EOM
		---
		# my probe task
		! task
		---

		status = "in_progress"
		priority = "p1"
		due = "20260415T120000Z"
		notes = "probe content"
	EOM
  assert_success

  run_dodder show '!task'
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-g9ch2vs6vwqhhx8jgmj6xhq02u3ze39mgjj5ranpn3jv9kdeu8rsz2r9xj !task "my probe task" status=in_progress priority=p1 due=20260415T120000Z]
	EOM
}

# !chore should have the same field set as !task — same field defs, same
# scripts. The blob digests should match because the bodies are
# byte-identical.
function genesis_chore_type_matches_task_type { # @test
  init_fixture -include-builtin-actionable-types

  run_dodder show -format object-id-blob-digest '!task:t'
  assert_success
  task_line="$output"

  run_dodder show -format object-id-blob-digest '!chore:t'
  assert_success
  chore_line="$output"

  task_digest="${task_line##* }"
  chore_digest="${chore_line##* }"

  if [[ "$task_digest" != "$chore_digest" ]]; then
    echo "task digest:  $task_digest" >&2
    echo "chore digest: $chore_digest" >&2
    false
  fi
}

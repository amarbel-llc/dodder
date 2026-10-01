#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"
  load "$(dirname "$BATS_TEST_FILE")/../lib/clone.bash"

  # for shellcheck SC2154
  export output
}

teardown() {
  chflags_nouchg
}

# bats file_tags=user_story:builtin_types,user_story:pull,user_story:remote

# dodder#403: type-declared dormancy (FDR 0025) depends on projected fields.
# These tests prove whether a done !task's projected fields -- and so its
# dormancy -- survive reindex and remote transfer (pull, clone).

# Creates a repo at $1 with the built-in actionable types and one done !task.
function bootstrap_done_task {
  mkdir -p "$1"
  (
    pushd "$1" || exit 1

    run_dodder init \
      -yin <(cat_yin) \
      -yang <(cat_yang) \
      -encryption none \
      -include-builtin-actionable-types \
      .default
    assert_success

    run_dodder init-workspace -experimental-repo=false
    assert_success

    run_dodder new -edit=false - <<-EOM
			---
			# finished task
			! task
			---

			status = "done"
			priority = "p1"
			due = "2026-07-01"
		EOM
    assert_success
  )
}

function assert_done_task_dormant_with_fields {
  run_dodder show '!task'
  assert_success
  assert_output ''

  run_dodder show '!task?z'
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-a0ydpcr67wt7ty9e0k6p2wc6j5cypcs44gs5rq02g24p94vwu37qg2sene !task "finished task" status=done priority=p1 due=2026-07-01 effort=]
	EOM
}

function type_dormancy_survives_reindex { # @test
  command -v yq >/dev/null || skip "yq not available"

  bootstrap_done_task them
  cd them || exit 1

  assert_done_task_dormant_with_fields

  run_dodder reindex
  assert_success

  assert_done_task_dormant_with_fields
}

function type_dormancy_survives_pull_direct { # @test
  command -v yq >/dev/null || skip "yq not available"

  bootstrap_done_task them

  mkdir us
  cd us || exit 1

  run_dodder_init_disable_age

  run_dodder pull -direct "$(realpath ../them)" +zettel,typ,etikett
  assert_success

  assert_done_task_dormant_with_fields
}

function type_dormancy_survives_clone { # @test
  command -v yq >/dev/null || skip "yq not available"

  bootstrap_done_task them

  mkdir us
  cd us || exit 1

  run_clone_default_with \
    .default \
    toml-repo-local_override_path-v0 \
    "$(realpath ../them)" \
    +zettel,typ,etikett
  assert_success

  assert_done_task_dormant_with_fields
}

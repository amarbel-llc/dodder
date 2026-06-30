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
    "$@" \
    .default
  assert_success
}

# Without the opt-in flag, dodder init does NOT commit !task / !chore /
# !habit. Only the default !md type is present.
function genesis_default_omits_actionable_types { # @test
  init_fixture

  run_dodder show ':t'
  assert_success
  assert_output --regexp - <<-EOM
		\[!md @blake2b256-.+ !toml-type-v2]
	EOM
}

# With -include-builtin-actionable-types, all three actionable types
# (!task, !chore, !habit) are committed during genesis alongside !md.
function genesis_includes_actionable_types_when_opted_in { # @test
  init_fixture -include-builtin-actionable-types

  run_dodder show ':t'
  assert_success
  assert_output_unsorted --regexp - <<-EOM
		\[!chore @blake2b256-.+ !toml-type-v2]
		\[!habit @blake2b256-.+ !toml-type-v2]
		\[!md @blake2b256-.+ !toml-type-v2]
		\[!task @blake2b256-.+ !toml-type-v2]
	EOM
}

# The !task type blob exposes the actionable field definitions (status,
# urgency, priority, due), the yq reader/writer scripts, and the
# blob-backed pandoc `text` formatter that renders the dang-typed body.
function genesis_task_type_blob_has_fields_scripts_and_formatter { # @test
  init_fixture -include-builtin-actionable-types

  run_dodder show -format blob '!task:t'
  assert_success
  # Note: tommy's TOML encoder serializes script as a triple-quoted
  # multiline string and unescapes backslashes (the Script field has the
  # `multiline` tommy tag). urgency carries no default (untriaged reads as
  # unset). The text formatter pipes the TOML `body` key through yq, strips
  # the leading `#!dang` convention line, then normalizes via pandoc. The
  # `hooks` value is the archive-on-status on_commit_fields lua hook,
  # serialized by tommy's plain string encoder as a single escaped line.
  assert_output - <<-'EOM'
		file-extension = "toml"
		vim-syntax-type = "toml"
		hooks = "return {\n  on_commit_fields = function(kinder, mutter)\n    local f = kinder.Fields\n    local status = f and f.status\n    if status == \"cancelled\" then\n      kinder.Etiketten[\"zz-archive\"] = true\n    elseif status == \"done\" and kinder.Typ == \"!task\" then\n      kinder.Etiketten[\"zz-archive\"] = true\n    end\n  end,\n}\n"

		[formatters.text]
		description = "Render the dang-typed body with pandoc"
		script = """
		yq -p toml -r '.body' | sed '1{/^#!dang/d}' | pandoc --data-dir="$DODDER_BLOB_TREE" --defaults=dodder-edit"""
		file-extension = "md"

		[[fields]]
		name = "status"
		kind = "enum"
		values = ["todo", "in_progress", "done", "cancelled"]
		default = "todo"

		[[fields]]
		name = "urgency"
		kind = "enum"
		values = ["0_hour", "1_day", "2_week", "3_month", "4_quarter", "5_episode", "6_year"]

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
		yq -p toml -o json '{"status": .status, "urgency": .urgency, "priority": .priority, "due": .due} | with_entries(select(.value != null))'"""

		[fields-writer]
		script = """
		yq -p toml -o toml -i ".status = \"$DODDER_FIELD_status\" | .urgency = \"$DODDER_FIELD_urgency\" | .priority = \"$DODDER_FIELD_priority\" | .due = \"$DODDER_FIELD_due\"" "$DODDER_BLOB_PATH""""
	EOM
}

# A freshly-created !task whose TOML blob sets urgency (plus status /
# priority / due) has all four fields projected by the reader script and
# visible in `dodder show`. urgency is optional (no default); when the blob
# supplies it, it projects through. The urgency-less case (unset =
# untriaged, commit must still succeed) is covered by
# actionable_task_omits_urgency_when_unset below.
function actionable_task_projects_urgency_field { # @test
  init_fixture -include-builtin-actionable-types
  run_dodder init-workspace -experimental-repo=false

  run_dodder new -edit=false - <<-EOM
		---
		# my probe task
		! task
		---

		status = "in_progress"
		urgency = "2_week"
		priority = "p1"
		due = "20260415T120000Z"
	EOM
  assert_success

  run_dodder show '!task'
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-mypv50rw9hr79g7c0r4fe06uurrldnre5s3va70wvrwlvc48tf3qpjqgx4 !task "my probe task" status=in_progress urgency=2_week priority=p1 due=20260415T120000Z]
	EOM
}

# A !task whose TOML blob omits urgency (only status / priority, as the
# CalDAV haustoria emits) MUST still commit: urgency is an optional no-default
# enum, so an empty value is treated as unset and dropped rather than rejected.
# Here the empty value is writer-manufactured — the fields-writer runs during
# the new/checkout commit and expands the unset DODDER_FIELD_urgency to "" — so
# this also exercises the writer-injection path. urgency does not appear in
# `dodder show`; the no-default STRING field `due` keeps its empty value and
# still renders as `due=`.
function actionable_task_omits_urgency_when_unset { # @test
  init_fixture -include-builtin-actionable-types
  run_dodder init-workspace -experimental-repo=false

  run_dodder new -edit=false - <<-EOM
		---
		# untriaged task
		! task
		---

		status = "todo"
		priority = "p3"
	EOM
  assert_success

  run_dodder show '!task'
  assert_success
  assert_output --regexp - <<-EOM
		\[one/uno @blake2b256-.+ !task "untriaged task" status=todo priority=p3 due=]
	EOM
}

# !chore carries a recurrence field on top of the actionable triple, so a
# recurrence value in the blob projects into the index. !task has no
# recurrence field, so the same key in a !task blob is NOT projected.
function actionable_chore_has_recurrence_task_does_not { # @test
  init_fixture -include-builtin-actionable-types
  run_dodder init-workspace -experimental-repo=false

  run_dodder new -edit=false - <<-EOM
		---
		# weekly chore
		! chore
		---

		status = "todo"
		urgency = "2_week"
		priority = "p2"
		due = "20260415T120000Z"
		recurrence = "P1W"
	EOM
  assert_success

  run_dodder new -edit=false - <<-EOM
		---
		# a plain task
		! task
		---

		status = "todo"
		urgency = "2_week"
		priority = "p2"
		due = "20260415T120000Z"
		recurrence = "P1W"
	EOM
  assert_success

  run_dodder show '!chore'
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-hxu6jqr2ecq6ntnzax4spk0usv39havp73cvr8qsvv8q5wys35rqe3lc0w !chore "weekly chore" status=todo urgency=2_week priority=p2 due=20260415T120000Z recurrence=P1W]
	EOM

  # The same recurrence key in a !task blob is NOT projected: !task has no
  # recurrence field, so the index carries no recurrence on the task. (The
  # task blob digest matches the chore's because the TOML bytes are
  # identical — content addressing collapses them.)
  run_dodder show '!task'
  assert_success
  assert_output - <<-EOM
		[one/dos @blake2b256-hxu6jqr2ecq6ntnzax4spk0usv39havp73cvr8qsvv8q5wys35rqe3lc0w !task "a plain task" status=todo urgency=2_week priority=p2 due=20260415T120000Z]
	EOM
}

# With both -include-builtin-actionable-types and the pandoc tools, the
# !task `text` formatter renders the dang-typed `body` blob key via
# blob-backed pandoc, stripping the leading `#!dang` convention line.
# format-blob runs the type's text formatter on the object's blob (unlike
# `show -format text`, which emits the raw zettel-text serialization).
function actionable_body_renders_via_pandoc { # @test
  command -v pandoc >/dev/null || skip "pandoc not available"
  command -v yq >/dev/null || skip "yq not available"

  init_fixture -include-builtin-actionable-types -include-default-pandoc-tools
  run_dodder init-workspace -experimental-repo=false

  run_dodder new -edit=false - <<-EOM
		---
		# body task
		! task
		---

		status = "todo"
		urgency = "2_week"
		priority = "p1"
		body = """
		#!dang md
		# Hello
		"""
	EOM
  assert_success

  run_dodder format-blob one/uno text
  assert_success
  assert_output - <<-EOM
		# Hello
	EOM
}

# A !task committed with status = "done" is archived: the on_commit_fields
# hook adds the genesis-seeded dormant archive tag (zz-archive), so the task is
# absent from the default listing and only visible with the dormant (?) sigil.
function actionable_task_archives_on_done { # @test
  init_fixture -include-builtin-actionable-types
  run_dodder init-workspace -experimental-repo=false

  run_dodder new -edit=false - <<-EOM
		---
		# done task
		! task
		---

		status = "done"
		priority = "p1"
	EOM
  assert_success

  # hidden from the default (non-dormant) listing
  run_dodder show '!task'
  assert_success
  assert_output ''

  # visible with the dormant sigil, carrying the archive tag
  run_dodder show '!task?z'
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-53nnqmn2c6eny28wu06vz0lzjxynx94taxx2l459l97kxn45yqzq69uegz !task "done task" zz-archive status=done priority=p1 due=]
	EOM
}

# status = "cancelled" archives every actionable type (!task, !chore, !habit):
# the shared hook adds the archive tag regardless of type, so all three become
# dormant and drop out of the default zettel listing.
function actionable_cancelled_archives_all_types { # @test
  init_fixture -include-builtin-actionable-types
  run_dodder init-workspace -experimental-repo=false

  run_dodder new -edit=false - <<-EOM
		---
		# cancelled task
		! task
		---

		status = "cancelled"
		priority = "p1"
	EOM
  assert_success

  run_dodder new -edit=false - <<-EOM
		---
		# cancelled chore
		! chore
		---

		status = "cancelled"
		priority = "p1"
		recurrence = "P1W"
	EOM
  assert_success

  run_dodder new -edit=false - <<-EOM
		---
		# cancelled habit
		! habit
		---

		status = "cancelled"
		priority = "p1"
		recurrence = "P1D"
	EOM
  assert_success

  # each type's default (non-dormant) listing is empty -- all archived
  run_dodder show '!task'
  assert_success
  assert_output ''

  run_dodder show '!chore'
  assert_success
  assert_output ''

  run_dodder show '!habit'
  assert_success
  assert_output ''

  # all three are visible with the dormant sigil, each carrying the archive tag
  run_dodder show ':?z'
  assert_success
  assert_output_unsorted - <<-EOM
		[one/uno @blake2b256-lhud3f7ausygpq5n946xhqa8jhfsu0p2vxvggufev9seeaftr7fsrrue2k !task "cancelled task" zz-archive status=cancelled priority=p1 due=]
		[one/dos @blake2b256-qjs3evuzk5a4vcgrpqpgrcqtxcyf40qvguy3jqpj3znflqujzzgq2cc885 !chore "cancelled chore" zz-archive status=cancelled priority=p1 due= recurrence=P1W]
		[two/uno @blake2b256-nns0qun0lejzas0pc9hp72r0vlpgaza537fkrlf7mh5rwmeekt0svujpa9 !habit "cancelled habit" zz-archive status=cancelled priority=p1 due= recurrence=P1D]
	EOM
}

# A recurring actionable type (!chore / !habit) with status = "done" is NOT
# archived: the shared hook gates the done branch on kinder.Typ == "!task", so
# a recurring "done" stays active (recurrence handling is deferred).
function actionable_recurring_done_not_archived { # @test
  init_fixture -include-builtin-actionable-types
  run_dodder init-workspace -experimental-repo=false

  run_dodder new -edit=false - <<-EOM
		---
		# done chore
		! chore
		---

		status = "done"
		priority = "p1"
		recurrence = "P1W"
	EOM
  assert_success

  run_dodder new -edit=false - <<-EOM
		---
		# done habit
		! habit
		---

		status = "done"
		priority = "p1"
		recurrence = "P1D"
	EOM
  assert_success

  # both remain in the default (non-dormant) listing -- not archived
  run_dodder show ':z'
  assert_success
  assert_output_unsorted - <<-EOM
		[one/uno @blake2b256-smjh7lktppj3ufunh87dvhgt4sydmhm53jc0wj8czrdshqgh52nq362757 !chore "done chore" status=done priority=p1 due= recurrence=P1W]
		[one/dos @blake2b256-d3684kxr0tu839mcg0zmglr78asered75qh060cvhe06rcn2lxds249pjy !habit "done habit" status=done priority=p1 due= recurrence=P1D]
	EOM
}

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
# !habit. Only the default !md type (with its pandoc tool tree, default-on
# since #208) and the two pandoc tool types are present. Golden-backed so the
# genesis snapshot regenerates with the pandoc default.
function genesis_default_omits_actionable_types { # @test
  init_fixture

  run_dodder show ':t'
  assert_success
  assert_golden_unsorted genesis_default_omits_actionable_types
}

# With -include-builtin-actionable-types, all three actionable types
# (!task, !chore, !habit) are committed during genesis alongside !md, plus the
# !lua tool type that locks the actionable-common.lua blob reference each
# actionable type carries, and the two pandoc tool types (default-on since
# #208). Each actionable type also carries the three pandoc tool-blob
# references alongside actionable-common.lua. Golden-backed: the normalizer
# masks the per-init ed25519 signatures while keeping the deterministic
# content-addressed blob digests verbatim.
function genesis_includes_actionable_types_when_opted_in { # @test
  init_fixture -include-builtin-actionable-types

  run_dodder show ':t'
  assert_success
  assert_golden_unsorted genesis_includes_actionable_types_when_opted_in
}

# The !task type blob exposes the actionable field definitions (status,
# urgency, priority, due), the yq reader/writer scripts, and the
# blob-backed pandoc body formatters (text/html/html-gdoc; no pdf-beamer —
# slides don't fit task prose).
function genesis_task_type_blob_has_fields_scripts_and_formatter { # @test
  init_fixture -include-builtin-actionable-types

  run_dodder show -format blob '!task:t'
  assert_success
  # Note: tommy's TOML encoder serializes script as a triple-quoted
  # multiline string (the Script field has the `multiline` tommy tag). The
  # fields-writer reads values via yq's strenv() env accessor and is
  # single-quoted so the shell performs no interpolation, keeping a
  # quote-bearing field value as string data rather than expression text
  # (#297). urgency carries no default (untriaged reads as
  # unset). Each body formatter pipes the TOML `body` key through yq, strips
  # the leading `#!dang` convention line, then renders via pandoc. The
  # `hooks` value is now the THIN loader: it require()s the blob-backed
  # actionable-common module (delivered as a blob reference on the type object)
  # and returns its hooks table. The recurrence/completed-date logic lives in
  # actionable-common.lua, not this inline string. status declares its
  # terminal (dormant) values: done and cancelled (FDR 0025).
  assert_output - <<-'EOM'
		file-extension = "toml"
		vim-syntax-type = "toml"
		hooks = "local common = require(\"actionable-common\")\nreturn common.hooks\n"

		[formatters.html]
		description = "Render the dang-typed body to an HTML fragment with pandoc"
		script = """
		yq -p toml -r '.body' | sed '1{/^#!dang/d}' | pandoc --data-dir="$DODDER_BLOB_TREE" --defaults=dodder-html"""
		file-extension = "html"
		[formatters.html-gdoc]
		description = "Render the dang-typed body to standalone HTML for pasting into Google Docs"
		script = """
		yq -p toml -r '.body' | sed '1{/^#!dang/d}' | pandoc --data-dir="$DODDER_BLOB_TREE" --defaults=dodder-gdoc"""
		file-extension = "html"
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
		terminal = ["done", "cancelled"]

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

		[[fields]]
		name = "effort"
		kind = "string"
		omit-empty = true

		[fields-reader]
		script = """
		yq -p toml -o json '{"status": .status, "urgency": .urgency, "priority": .priority, "due": .due, "effort": .effort} | with_entries(select(.value != null))'"""

		[fields-writer]
		script = """
		yq -p toml -o toml -i '.status = strenv(DODDER_FIELD_status) | .urgency = strenv(DODDER_FIELD_urgency) | .priority = strenv(DODDER_FIELD_priority) | .due = strenv(DODDER_FIELD_due) | .effort = strenv(DODDER_FIELD_effort)' "$DODDER_BLOB_PATH""""
	EOM
}

# effort is an open-set string field (#400): any value, unit included, projects
# through unvalidated, is queryable as a field, and an unset effort reads as
# unset (omit-empty) rather than `effort=`.
function actionable_task_projects_open_set_effort { # @test
  init_fixture -include-builtin-actionable-types
  run_dodder init-workspace -experimental-repo=false

  run_dodder new -edit=false - <<-EOM
		---
		# sized task
		! task
		---

		status = "todo"
		priority = "p2"
		effort = "2pom"
	EOM
  assert_success

  run_dodder show '!task'
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-fcy860vz0sf4zzcrhr4dzyvaqwwtkfp85uw7jnpehn9ggcfnd26qzgw3cz !task "sized task" status=todo priority=p2 due= effort=2pom]
	EOM

  run_dodder new -edit=false - <<-EOM
		---
		# uncountable task
		! task
		---

		effort = "uncountable"
	EOM
  assert_success

  run_dodder new -edit=false - <<-EOM
		---
		# unsized task
		! task
		---

		status = "todo"
	EOM
  assert_success

  run_dodder show 'effort=2pom'
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-fcy860vz0sf4zzcrhr4dzyvaqwwtkfp85uw7jnpehn9ggcfnd26qzgw3cz !task "sized task" status=todo priority=p2 due= effort=2pom]
	EOM

  run_dodder show '!task'
  assert_success
  assert_output_unsorted - <<-EOM
		[one/dos @blake2b256-vpk26d42syfelm3tgs3jy86e962zmsrwmt26j46dy5xuj82rem9qqhga4j !task "uncountable task" status=todo priority=p3 due= effort=uncountable]
		[one/uno @blake2b256-fcy860vz0sf4zzcrhr4dzyvaqwwtkfp85uw7jnpehn9ggcfnd26qzgw3cz !task "sized task" status=todo priority=p2 due= effort=2pom]
		[two/uno @blake2b256-y4hnl7xvp98ejlzw86dkf3hwkm85sjpngzj4et08fsctfx0sd56s980svu !task "unsized task" status=todo priority=p3 due=]
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
		[one/uno @blake2b256-fknj3d0nfzp0mg4u6ewz46adzsl6qez76qle768dflgg59xy6npsyk2szj !task "my probe task" status=in_progress urgency=2_week priority=p1 due=20260415T120000Z]
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
		[one/uno @blake2b256-da8z8uqaagmfj2xe8tl45rmjcunaz38fnqpndkt3wtrjjhqzs5rs4epyaq !chore "weekly chore" status=todo urgency=2_week priority=p2 due=20260415T120000Z recurrence=P1W]
	EOM

  # The same recurrence key in a !task blob is NOT projected: !task has no
  # recurrence field, so the index carries no recurrence on the task. (The
  # task blob digest matches the chore's because the TOML bytes are
  # identical — content addressing collapses them.)
  run_dodder show '!task'
  assert_success
  assert_output - <<-EOM
		[one/dos @blake2b256-da8z8uqaagmfj2xe8tl45rmjcunaz38fnqpndkt3wtrjjhqzs5rs4epyaq !task "a plain task" status=todo urgency=2_week priority=p2 due=20260415T120000Z]
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

  init_fixture -include-builtin-actionable-types
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

# The actionable `html` formatter mirrors `text`: the dang-typed body is
# extracted with yq, the `#!dang` convention line is stripped, then rendered
# to an HTML fragment via the blob-backed dodder-html defaults. (There is
# deliberately no pdf-beamer on actionable types — slides don't fit task
# prose.)
function actionable_body_renders_html_via_pandoc { # @test
  command -v pandoc >/dev/null || skip "pandoc not available"
  command -v yq >/dev/null || skip "yq not available"

  init_fixture -include-builtin-actionable-types
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

  run_dodder format-blob one/uno html
  assert_success
  assert_output - <<-EOM
		<h1 id="hello">Hello</h1>
	EOM
}

# The actionable `html-gdoc` formatter renders the body as a standalone
# HTML document; only the load-bearing shape is asserted (the bulk is
# pandoc's default template).
function actionable_body_renders_gdoc_html_via_pandoc { # @test
  command -v pandoc >/dev/null || skip "pandoc not available"
  command -v yq >/dev/null || skip "yq not available"

  init_fixture -include-builtin-actionable-types
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

  run_dodder format-blob one/uno html-gdoc
  assert_success
  assert_output --regexp '^<!DOCTYPE html>'
  assert_output --regexp '<title>dodder</title>'
  assert_output --regexp '<h1 id="hello">Hello</h1>'
  assert_output --regexp '</html>$'
}

# B2 resolution proof: the actionable type's Hooks string carries NO
# completed-date logic -- it is a thin `require("actionable-common")` loader.
# The logic lives solely in the actionable-common.lua blob, delivered as a blob
# REFERENCE on the type object and preloaded into the hook VM by name
# (oscar/store). So a done !task getting its empty `due` stamped proves
# require() resolved that blob-backed module through the object graph
# (FDR-0000).
function actionable_hook_resolves_via_blob_reference { # @test
  init_fixture -include-builtin-actionable-types
  run_dodder init-workspace -experimental-repo=false

  run_dodder new -edit=false - <<-EOM
		---
		# graph-resolved done task
		! task
		---

		status = "done"
		priority = "p2"
	EOM
  assert_success

  # due-stamped: only reachable if the thin hook's
  # require("actionable-common") loaded the blob-referenced module.
  run_dodder show '!task?z'
  assert_success
  assert_output --regexp - <<-EOM
		\[one/uno @blake2b256-.+ !task "graph-resolved done task" status=done priority=p2 due=[0-9]{4}-[0-9]{2}-[0-9]{2}]
	EOM
}

# A !task committed with status = "done" is dormant: !task declares done
# terminal on its status field (FDR 0025), so the task is absent from the
# default listing and only visible with the dormant (?) sigil -- without any
# archive tag.
function actionable_task_dormant_on_done { # @test
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

  # visible with the dormant sigil, with no archive tag. The B1
  # completed-date auto-stamp fills the empty `due` with today (UTC), so the
  # blob digest and `due` value are date-dependent -- match with --regexp.
  run_dodder show '!task?z'
  assert_success
  assert_output --regexp - <<-EOM
		\[one/uno @blake2b256-.+ !task "done task" status=done priority=p1 due=[0-9]{4}-[0-9]{2}-[0-9]{2}]
	EOM
}

# A type-dormant object (terminal status, FDR 0025) must be hidden by the
# empty-predicate genre-only query ':z' exactly like a tag-dormant object
# (dormant-add), not leak through as a bare row. It stays visible with the
# dormant sigil ':?z'.
function actionable_task_dormant_on_done_hidden_under_empty_genre_query { # @test
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

  # hidden from the empty-predicate genre-only listing
  run_dodder show ':z'
  assert_success
  assert_output ''

  # hidden from the default listing
  run_dodder show
  assert_success
  assert_output ''

  # hidden from status
  run_dodder status
  assert_success
  assert_output ''

  # visible with the dormant sigil, with full metadata and no archive tag.
  # B1 stamps today into the empty `due`, so blob digest + `due` are
  # date-dependent -- match with --regexp.
  run_dodder show ':?z'
  assert_success
  assert_output --regexp - <<-EOM
		\[one/uno @blake2b256-.+ !task "done task" status=done priority=p1 due=[0-9]{4}-[0-9]{2}-[0-9]{2}]
	EOM
}

# status = "cancelled" is terminal for every actionable type (!task, !chore,
# !habit), so all three become dormant and drop out of the default zettel
# listing.
function actionable_cancelled_dormant_all_types { # @test
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

  # each type's default (non-dormant) listing is empty -- all dormant
  run_dodder show '!task'
  assert_success
  assert_output ''

  run_dodder show '!chore'
  assert_success
  assert_output ''

  run_dodder show '!habit'
  assert_success
  assert_output ''

  # all three are visible with the dormant sigil. B1 stamps today (UTC) into
  # each empty `due` on cancel, so the blob
  # digests and `due` values are date-dependent -- match with --regexp.
  run_dodder show ':?z'
  assert_success
  assert_output_unsorted --regexp - <<-EOM
		\[one/uno @blake2b256-.+ !task "cancelled task" status=cancelled priority=p1 due=[0-9]{4}-[0-9]{2}-[0-9]{2}]
		\[one/dos @blake2b256-.+ !chore "cancelled chore" status=cancelled priority=p1 due=[0-9]{4}-[0-9]{2}-[0-9]{2} recurrence=P1W]
		\[two/uno @blake2b256-.+ !habit "cancelled habit" status=cancelled priority=p1 due=[0-9]{4}-[0-9]{2}-[0-9]{2} recurrence=P1D]
	EOM
}

# A recurring actionable type (!chore / !habit) committed with status = "done"
# is NOT archived: the shared hook recurs it instead. When the blob carries a
# `due` date, the hook advances due by the recurrence duration (via the
# dodder_advance_date host helper) and resets status to "todo". The commit
# COMPLETES (a recurrence cycle would hang); `dodder show` reflects the
# rewritten values re-projected from the rewritten blob.
function actionable_chore_recurs_on_done { # @test
  command -v yq >/dev/null || skip "yq not available"

  init_fixture -include-builtin-actionable-types
  run_dodder init-workspace -experimental-repo=false

  run_dodder new -edit=false - <<-EOM
		---
		# weekly chore
		! chore
		---

		status = "done"
		priority = "p1"
		due = "2026-07-01"
		recurrence = "P1W"
	EOM
  assert_success

  # recurred, not archived: still in the default (non-dormant) listing, due
  # advanced exactly one week, status reset to todo.
  run_dodder show '!chore'
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-xy9wwu5dtf7rp0mtcmp4myhvnsp6ack7w69uk56rf5rvp2r6fvgqu84gsj !chore "weekly chore" status=todo priority=p1 due=2026-07-08 recurrence=P1W]
	EOM
}

# A !habit recurs the same way: P1D advances due by one day and resets status
# to todo. Confirms the recurrence hook is shared across recurring types, not
# !chore-specific.
function actionable_habit_recurs_on_done { # @test
  command -v yq >/dev/null || skip "yq not available"

  init_fixture -include-builtin-actionable-types
  run_dodder init-workspace -experimental-repo=false

  run_dodder new -edit=false - <<-EOM
		---
		# daily habit
		! habit
		---

		status = "done"
		priority = "p1"
		due = "2026-07-01"
		recurrence = "P1D"
	EOM
  assert_success

  run_dodder show '!habit'
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-qf3dswek2tuh69n33g36ajhm4j9haewq4fw69f2czkrv33nav64s4n4dse !habit "daily habit" status=todo priority=p1 due=2026-07-02 recurrence=P1D]
	EOM
}

# Empty-due guard: a recurring type completed with no `due` value recurs by
# resetting status to todo only -- there is nothing to advance, so due stays
# empty. Still not archived.
function actionable_recurring_done_resets_status_with_empty_due { # @test
  command -v yq >/dev/null || skip "yq not available"

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

  run_dodder show '!chore'
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-gtzv2d7z7kqg7fusthxhdj5hc6ty6wnrh44y4eah5jj2zhpfxdusflux4m !chore "done chore" status=todo priority=p1 due= recurrence=P1W]
	EOM
}

# Idempotency: the recurred chore is now status=todo. Re-committing it (via
# organize, with the already-recurred field values) does NOT advance due again
# -- the hook only acts on status="done", so a todo re-commit is a no-op for
# recurrence. due stays at the once-advanced 2026-07-08.
function actionable_chore_recurrence_is_idempotent { # @test
  command -v yq >/dev/null || skip "yq not available"

  init_fixture -include-builtin-actionable-types
  run_dodder init-workspace -experimental-repo=false

  run_dodder new -edit=false - <<-EOM
		---
		# weekly chore
		! chore
		---

		status = "done"
		priority = "p1"
		due = "2026-07-01"
		recurrence = "P1W"
	EOM
  assert_success

  # re-commit the already-recurred (status=todo, due=2026-07-08) values
  base_line="$(get_organize_base '!chore')"

  run_dodder organize -mode commit-directly '!chore' <<-EOM
		---
		$base_line
		---

		- [one/uno !chore status=todo priority=p1 due=2026-07-08 recurrence=P1W] weekly chore
	EOM
  assert_success

  # due unchanged: no second advance (same blob digest as the single-recur
  # case confirms idempotency at the content level)
  run_dodder show '!chore'
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-xy9wwu5dtf7rp0mtcmp4myhvnsp6ack7w69uk56rf5rvp2r6fvgqu84gsj !chore "weekly chore" status=todo priority=p1 due=2026-07-08 recurrence=P1W]
	EOM
}

# Regression (#297): a free-form field value containing a double-quote must
# round-trip through the fields-writer. The writer reads values via yq's
# strenv() env accessor rather than shell-interpolating them into the yq
# expression, so a `"` in the free-form `due` field is treated as string data,
# not expression text. With the old shell-interpolated writer the `"` closed
# the yq string early, breaking the yq parse (or injecting expression syntax)
# and failing the commit. Here the quote is driven through the real !task
# writer via an organize field mutation, then read back intact.
function actionable_field_writer_survives_quote_in_due { # @test
  command -v yq >/dev/null || skip "yq not available"

  init_fixture -include-builtin-actionable-types
  run_dodder init-workspace -experimental-repo=false

  run_dodder new -edit=false - <<-EOM
		---
		# quote task
		! task
		---

		status = "todo"
		priority = "p1"
		due = "20260415T120000Z"
	EOM
  assert_success

  # mutate `due` to a value containing a double-quote; the fields-writer
  # projects it back into the TOML blob via strenv(), so the yq parse is not
  # broken and the value round-trips.
  base_line="$(get_organize_base '!task')"

  run_dodder organize -mode commit-directly '!task' <<-EOM
		---
		$base_line
		---

		- [one/uno !task status=todo priority=p1 due="he said \"hi\""] quote task
	EOM
  assert_success

  run_dodder show '!task'
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-e9fdrlj6gd3qt9qf2gnyp48jcrffnyqek96fy5s93y4agsk26scqguwfza !task "quote task" status=todo priority=p1 due="he said \"hi\""]
	EOM
}

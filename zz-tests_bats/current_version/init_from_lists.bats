#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

  # for shellcheck SC2154
  export output

  set_xdg "$BATS_TEST_TMPDIR"

  # A user-scoped blob store the source repo populates and init-from-lists
  # reads source blobs from via -blob-source.
  run_madder init shared
  assert_success

  run_dodder init \
    -yin <(cat_yin) \
    -yang <(cat_yang) \
    -encryption none \
    -blob_store-id shared \
    .default
  assert_success

  run_dodder init-workspace -experimental-repo=false

  create_test_zettels

  # Under the write_through multi default the named store is a read-only
  # fallback, so the zettel blobs land in .default-local; copy them into
  # shared so init-from-lists can resolve them as a -blob-source.
  run_madder sync .default-local shared
  assert_success

  # Export the source repo's full object graph (zettels, tags, types) to a
  # list file the consolidation consumes.
  run_dodder export -print-time=true +z,e,t
  assert_success
  echo "$output" >list
  list="$(realpath list)"
}

teardown() {
  chflags_nouchg
}

# bats file_tags=user_story:transform

# dodder#392: init-from-lists genesises a fresh repo, applies a transform to the
# union of the given list files, and imports the result re-signed under the
# newborn's key. Here a tag-cleanup transform tags every object "consolidated".
function init_from_lists_consolidates_with_transform { # @test
  cat >s.lua <<-'EOM'
		local l = dodder.list()

		for object in l:each() do
		  object.Etiketten["consolidated"] = true
		end

		return l
	EOM
  script="$(realpath s.lua)"

  mkdir consolidated
  cd consolidated || exit 1

  run_dodder init-from-lists \
    -encryption none \
    -script "$script" \
    -blob-source shared \
    .default \
    "$list"
  assert_success

  run_dodder show :z
  assert_success
  assert_output_unsorted - <<-EOM
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" consolidated tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" consolidated tag-3 tag-4]
	EOM

  # Self-containment (dodder#392): the newborn does NOT reference the shared
  # -blob-source store in its own config, so a clean fsck (which reads only the
  # newborn's stores) proves every referenced blob — including the !md type
  # blob that lived only in shared — was copied into the newborn. It survives
  # deleting the sources.
  run_dodder fsck
  assert_success
}

# dodder#392: exact (id,tai,digest) duplicates across the union collapse to one
# — they must NOT be reassigned to spurious extra revisions. Passing the SAME
# list twice unions it with itself; the result must equal a single-list run.
function init_from_lists_union_collapses_exact_duplicates { # @test
  cat >s.lua <<-'EOM'
		return dodder.list()
	EOM
  script="$(realpath s.lua)"

  mkdir consolidated
  cd consolidated || exit 1

  run_dodder init-from-lists \
    -encryption none \
    -script "$script" \
    -blob-source shared \
    .default \
    "$list" "$list"
  assert_success

  # The union of the doubled input (12 raw entries) collapses to the same 6
  # distinct objects a single list yields — the exact duplicates are dropped,
  # not reassigned to spurious extra revisions.
  assert_line 'union of 2 list(s): 6 object(s)'

  run_dodder show :z
  assert_success
  assert_output_unsorted - <<-EOM
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM
}

# dodder#399 / FDR 0025: an imported !task whose status is terminal is dormant
# in the newborn just by being committed -- no dormancy tag, no dormant-index
# entry. The consolidation transform is the identity; the newborn recomputes
# dormancy from the !task type's declared terminal status values.
function init_from_lists_terminal_status_is_dormant_without_tags { # @test
  command -v yq >/dev/null || skip "yq not available"

  mkdir source
  pushd source || exit 1

  run_dodder init \
    -yin <(cat_yin) \
    -yang <(cat_yang) \
    -encryption none \
    -blob_store-id shared \
    -include-builtin-actionable-types \
    .default
  assert_success

  run_dodder init-workspace -experimental-repo=false

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

  run_dodder new -edit=false - <<-EOM
		---
		# open task
		! task
		---

		status = "todo"
		priority = "p2"
		due = "2026-07-02"
	EOM
  assert_success

  run_madder sync .default-local shared
  assert_success

  run_dodder export -print-time=true '+?z,e,t'
  assert_success
  echo "$output" >list
  source_list="$(realpath list)"

  popd || exit 1

  cat >s.lua <<-'EOM'
		return dodder.list()
	EOM
  script="$(realpath s.lua)"

  mkdir consolidated
  cd consolidated || exit 1

  run_dodder init-from-lists \
    -encryption none \
    -script "$script" \
    -blob-source shared \
    .default \
    "$source_list"
  assert_success

  # only the open task is active. Its fields are projected on import, which
  # requires resolving the !task type committed earlier in the same unflushed
  # import batch.
  run_dodder show '!task'
  assert_success
  assert_output - <<-EOM
		[one/dos @blake2b256-hcnh97v0zf4nrn4jzwrqc90a24nf8zygxpng7mln2fcw6rypef2sje98ql !task "open task" status=todo priority=p2 due=2026-07-02 effort=]
	EOM

  # the finished task is dormant and carries no tags
  run_dodder show '!task?z'
  assert_success
  assert_output_unsorted - <<-EOM
		[one/uno @blake2b256-a0ydpcr67wt7ty9e0k6p2wc6j5cypcs44gs5rq02g24p94vwu37qg2sene !task "finished task" status=done priority=p1 due=2026-07-01 effort=]
		[one/dos @blake2b256-hcnh97v0zf4nrn4jzwrqc90a24nf8zygxpng7mln2fcw6rypef2sje98ql !task "open task" status=todo priority=p2 due=2026-07-02 effort=]
	EOM
}

# dodder#392: -plan-only builds and reports the plan's classification without
# committing. It reports the union, prints the dry-run marker, and leaves the
# freshly genesised repo empty (nothing imported, no source blobs copied) —
# the fast feedback loop for iterating a consolidation transform and the
# Gate-1 classification surface for the take4 personal-data consolidation.
function init_from_lists_plan_only_reports_without_committing { # @test
  cat >s.lua <<-'EOM'
		return dodder.list()
	EOM
  script="$(realpath s.lua)"

  mkdir consolidated
  cd consolidated || exit 1

  run_dodder init-from-lists \
    -encryption none \
    -plan-only \
    -script "$script" \
    -blob-source shared \
    .default \
    "$list"
  assert_success
  assert_line 'union of 1 list(s): 6 object(s)'
  assert_line 'dry run: not committed'

  # Nothing was committed: the newborn holds no zettels.
  run_dodder show :z
  assert_success
  assert_output ''
}

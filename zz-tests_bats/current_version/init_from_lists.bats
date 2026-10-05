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
		effort = "2pom"
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
		[one/dos @blake2b256-tv7h22q0qd265fh8agheqtczs9lvaqvfaz5mvju3lvcteup9pscsf4lq8s !task "open task" status=todo priority=p2 due=2026-07-02 effort=2pom]
	EOM

  # the finished task is dormant and carries no tags
  run_dodder show '!task?z'
  assert_success
  assert_output_unsorted - <<-EOM
		[one/uno @blake2b256-a0ydpcr67wt7ty9e0k6p2wc6j5cypcs44gs5rq02g24p94vwu37qg2sene !task "finished task" status=done priority=p1 due=2026-07-01]
		[one/dos @blake2b256-tv7h22q0qd265fh8agheqtczs9lvaqvfaz5mvju3lvcteup9pscsf4lq8s !task "open task" status=todo priority=p2 due=2026-07-02 effort=2pom]
	EOM
}

# take4 (#16): an object the script adds (list:add) is allocated a zettel id
# that the imported union does not already use. The newborn's id index starts
# empty, so without reserving the union's ids first the allocator could hand
# the new object one/uno or one/dos; the pool here (one × uno,dos,tres) leaves
# exactly one free id. The added object also carries a description and a
# metadata reference set by the script.
function init_from_lists_added_object_avoids_union_ids { # @test
  cat >s.lua <<-'EOM'
		local l = dodder.list()

		local added = l:add()
		added.Typ = "md"
		added.Bezeichnung = "spun off"
		added.References[#added.References + 1] = "one/uno"

		return l
	EOM
  script="$(realpath s.lua)"

  mkdir consolidated
  cd consolidated || exit 1

  run_dodder init-from-lists \
    -encryption none \
    -yin <(printf 'one\n') \
    -yang <(printf 'uno\ndos\ntres\n') \
    -script "$script" \
    -blob-source shared \
    .default \
    "$list"
  assert_success

  # the reference is locked to one/uno's signature under the newborn's fresh
  # (non-deterministic) key
  run_dodder show :z
  assert_success
  assert_output_unsorted --regexp - <<-'EOM'
		\[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4\]
		\[one/tres !md "spun off" <one/uno@ed25519_sig-[a-z0-9]+\]
		\[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4\]
	EOM
}

# take4 (#16): -write-blob_store-id adopts an EXISTING store as the newborn's
# write store instead of creating the scope-shared default-local, so a
# consolidation into a pre-filled store copies nothing. `shared` already holds
# every source blob (setup syncs them in); the user-scoped newborn writes
# into it, never creates default-local, and reads its zettels back clean.
function init_from_lists_adopts_named_write_store { # @test
  cat >s.lua <<-'EOM'
		return dodder.list()
	EOM
  script="$(realpath s.lua)"

  run_dodder init-from-lists \
    -encryption none \
    -write-blob_store-id shared \
    -script "$script" \
    take4 \
    "$list"
  assert_success

  run test -e "$XDG_DATA_HOME/madder/blob_stores/default-local"
  assert_failure

  run_dodder show -repo_id take4 :z
  assert_success
  assert_output_unsorted - <<-EOM
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM

  run_dodder fsck -repo_id take4
  assert_success
}

# take4 (#16): objects whose type exists only because GENESIS created it
# (here the built-in !task from -include-builtin-actionable-types; the source
# list carries no !task) commit in the same init-from-lists run, and the
# genesis objects stay resolvable by id in later runs (a later `new -type
# task` locks against the genesis !task). Genesis used to reset the indexes
# AFTER its own commit, closing the object probe pages in-process: same-run
# lookups failed ("failed to write type lock") and the next flush rewrote the
# pages without the genesis rows.
function init_from_lists_genesis_types_resolve_in_run { # @test
  cat >s.lua <<-'EOM'
		local l = dodder.list()

		local task = l:add()
		task.Typ = "task"
		task.Bezeichnung = "added task"
		task.Blob = blobs.write("status = \"todo\"\npriority = \"p2\"\n")

		return l
	EOM
  script="$(realpath s.lua)"

  run_dodder init-from-lists \
    -encryption none \
    -yin <(cat_yin) \
    -yang <(cat_yang) \
    -include-builtin-actionable-types \
    -script "$script" \
    -blob-source shared \
    take4 \
    "$list"
  assert_success

  run_dodder show -repo_id take4 '!task'
  assert_success
  assert_output - <<-'EOM'
		[two/uno @blake2b256-5ztwtk8e0c2jpm6ycda6jwl253u0237uax678qgekrzvlt0dg4lsnjxpvz !task "added task" status=todo priority=p2 due=]
	EOM

  run_dodder new -repo_id take4 -edit=false -type task -description "later task" \
    -blob 'status = "todo"'
  assert_success

  run_dodder fsck -repo_id take4
  assert_success
}

# take4 (#16): -removed-list archives what the script dropped. Every version of
# one/dos is tagged with a drop-class marker and removed; the newborn holds only
# one/uno, and the removed list carries one/dos with the marker the script set
# before removing it. (The list holds only the removed objects, not the types
# they are locked to, so it is an archive, not a standalone import source.)
function init_from_lists_removed_list_archives_drops { # @test
  cat >s.lua <<-'EOM'
		local l = dodder.list()

		for object in l:each() do
		  if object.Kennung == "one/dos" then
		    object.Etiketten["zz-dropped-test"] = true
		    l:remove(object)
		  end
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
    -removed-list "$BATS_TEST_TMPDIR/removed" \
    .default \
    "$list"
  assert_success

  run_dodder show :z
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM

  # one line per removed object; keys and signatures are per-run
  run cat "$BATS_TEST_TMPDIR/removed"
  assert_success
  assert_output --regexp - <<-'EOM'
		^---
		! inventory_list-v1
		---

		\[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd [0-9]+\.[0-9]+ dodder-repo-public_key-v1@ed25519_pub-[a-z0-9]+ dodder-object-sig-v3@ed25519_sig-[a-z0-9]+ !md@ed25519_sig-[a-z0-9]+ "wow ok again" tag-3 tag-4 zz-dropped-test\]$
	EOM
}

# #407: -removed-list archives the drops of whichever binding the script
# returned. Same scenario as above through the opt-in English-keyed
# dodder.list_v2(); an archive read off the V1 binding would be empty.
function init_from_lists_removed_list_archives_drops_list_v2 { # @test
  cat >s.lua <<-'EOM'
		local l = dodder.list_v2()

		for object in l:each() do
		  if object.ObjectId == "one/dos" then
		    object.Tags["zz-dropped-test"] = true
		    l:remove(object)
		  end
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
    -removed-list "$BATS_TEST_TMPDIR/removed" \
    .default \
    "$list"
  assert_success

  run_dodder show :z
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM

  # one line per removed object; keys and signatures are per-run
  run cat "$BATS_TEST_TMPDIR/removed"
  assert_success
  assert_output --regexp - <<-'EOM'
		^---
		! inventory_list-v1
		---

		\[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd [0-9]+\.[0-9]+ dodder-repo-public_key-v1@ed25519_pub-[a-z0-9]+ dodder-object-sig-v3@ed25519_sig-[a-z0-9]+ !md@ed25519_sig-[a-z0-9]+ "wow ok again" tag-3 tag-4 zz-dropped-test\]$
	EOM
}

# a store that does not exist cannot be adopted
function init_from_lists_write_store_must_exist { # @test
  cat >s.lua <<-'EOM'
		return dodder.list()
	EOM
  script="$(realpath s.lua)"

  run_dodder init-from-lists \
    -encryption none \
    -write-blob_store-id no_such_store \
    -script "$script" \
    take4 \
    "$list"
  assert_failure
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

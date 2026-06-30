#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"
  export output
}

teardown() {
  chflags_nouchg
}

# bats file_tags=user_story:query,user_story:tag

# #307: a materialized tag (a !toml-tag-v1 object created with no filter) must
# stay usable as a query filter term — querying by its name returns the objects
# carrying it, exactly like a string-only (unmaterialized) tag. Before the fix,
# the empty tag blob was built as a Lua query filter whose empty script failed
# VM validation, aborting the whole query.
function query_by_materialized_filterless_tag { # @test
  run_dodder_init

  # Materialize a bare tag-object (no filter).
  run_dodder new -edit=false -object-id mytag307
  assert_success

  # A zettel carrying that tag.
  run_dodder new -edit=false - <<-EOM
		---
		# repro zettel
		- mytag307
		! md
		---

		body
	EOM
  assert_success

  # Querying by the materialized tag name must return the carrying zettel.
  run_dodder show mytag307
  assert_success
  assert_output --regexp '^\[one/uno @blake2b256-.+ !md "repro zettel" mytag307\]$'
}

# Guard the unaffected path: a string-only tag (never materialized as an object)
# was already queryable and must remain so.
function query_by_string_only_tag { # @test
  run_dodder_init

  run_dodder new -edit=false - <<-EOM
		---
		# string only zettel
		- stringonly307
		! md
		---

		body
	EOM
  assert_success

  run_dodder show stringonly307
  assert_success
  assert_output --regexp '^\[one/uno @blake2b256-.+ !md "string only zettel" stringonly307\]$'
}

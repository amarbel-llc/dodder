#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

  # for shellcheck SC2154
  export output
}

teardown() {
  chflags_nouchg
}

# bats file_tags=export

function export_index_is_superset_of_query_export { # @test
  run_dodder_init_disable_age
  create_test_zettels

  run_dodder export '+?z,t,k,e'
  assert_success
  printf '%s\n' "$output" | grep '^\[' | sort >"$BATS_TEST_TMPDIR/query-lines"

  run_dodder export-index
  assert_success
  printf '%s\n' "$output" | grep '^\[' | sort >"$BATS_TEST_TMPDIR/index-lines"

  # The raw index dump reads below the query layer (no genre mask, no
  # query semantics), so every state the query export surfaces must
  # appear verbatim in the index dump.
  missing=$(comm -23 "$BATS_TEST_TMPDIR/query-lines" "$BATS_TEST_TMPDIR/index-lines")
  [[ -z $missing ]] || fail "query-export lines missing from index dump:
$missing"

  query_count=$(wc -l <"$BATS_TEST_TMPDIR/query-lines")
  index_count=$(wc -l <"$BATS_TEST_TMPDIR/index-lines")
  [[ $index_count -ge $query_count ]] || fail "index dump ($index_count) smaller than query export ($query_count)"
}

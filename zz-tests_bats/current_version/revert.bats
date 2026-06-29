#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

  # for shellcheck SC2154
  export output

  copy_from_version "$DIR"
}

teardown() {
  chflags_nouchg
}

cmd_def_organize=(
  "${cmd_dodder_def[@]}"
  -prefix-joints=false
  -refine=true
)

# Regression: `dodder revert` of an object whose checked-out working copy
# still matches HEAD refreshes that working copy to the reverted state, the
# same way organize does. Store.RevertTo now commits with MergeCheckedOut,
# which routes through the store's ReadExternalAndMergeIfNecessary (clean
# working copy -> UpdateCheckoutFromCheckedOut). one/dos is mutated via
# organize (adding tag 'test', which refreshes the checkout), then reverted;
# status must report it `same` WITHOUT 'test', proving the clean checkout was
# refreshed back rather than left at the reverted-from state. The leading
# status-column whitespace is matched loosely; the meaningful signal is the
# presence of 'test' after organize and its absence after revert.
function revert_refreshes_clean_checkout { # @test
  run_dodder_init_workspace

  run_dodder checkout one/dos
  assert_success
  assert_output --regexp '^ +checked out \[one/dos\.zettel @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4\]$'

  # Mutate one/dos (and one/uno) via organize, adding the tag 'test'.
  # organize refreshes the clean checkout, so one/dos.zettel now carries
  # 'test' on disk — the divergence we revert below.
  run_dodder organize "${cmd_def_organize[@]}" -mode commit-directly :z <<-EOM
		---
		- test
		---

		- [one/dos  !md tag-3 tag-4] wow ok again
		- [one/uno  !md tag-3 tag-4] wow the first
	EOM
  assert_success

  # Only one/dos is checked out, so status shows just it — now `same` and
  # carrying 'test' (organize refreshed the working copy).
  run_dodder status
  assert_success
  assert_output --regexp '^ +same \[one/dos\.zettel @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4 test\]$'

  # Revert one/dos to its mother (the pre-organize state). The clean checkout
  # must refresh back to that state, not stay at the reverted-from one.
  run_dodder revert one/dos
  assert_success

  run_dodder status
  assert_success
  assert_output --regexp '^ +same \[one/dos\.zettel @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4\]$'
}

function revert_one_zettel { # @test
  run_dodder revert one/uno
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 !md "wow ok" tag-1 tag-2]
	EOM
}

function revert_all_zettels { # @test
  run_dodder revert :z
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 !md "wow ok" tag-1 tag-2]
	EOM
}

function revert_last { # @test
  # TODO fix issue with output
  skip
  run_dodder revert -last
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 !md "wow ok" tag-1 tag-2]
	EOM

  run_dodder last
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 !md "wow ok" tag-1 tag-2]
	EOM

  run_dodder show one/uno
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 !md "wow ok" tag-1 tag-2]
	EOM
}

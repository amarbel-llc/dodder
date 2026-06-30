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

function migration_status_empty { # @test
  run_dodder status
  assert_failure
}

function migration_validate_schwanzen { # @test
  run_dodder show -format log :z,e,t
  assert_output_unsorted - <<-EOM
		[!md @$(get_type_blob_sha) !toml-type-v2]
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM
}

function migration_validate_history { # @test
  run_dodder show -format log +z,e,t
  assert_output_unsorted - <<-EOM
		[!md @$(get_type_blob_sha) !toml-type-v2]
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
		[one/uno @blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 !md "wow ok" tag-1 tag-2]
	EOM
}

function migration_reindex { # @test
  tree .madder
  run_dodder reindex
  assert_success
  assert_output

  # konfig is no longer an indexed object (FDR 0020, Task 8), so it is
  # diverted from the stream-index rebuild and no longer appears in genre
  # queries; the konfig token itself now errors. The remaining indexed
  # objects (types + zettels) are unchanged by the migration.
  run_dodder show +e,t,z
  assert_success
  assert_output_unsorted - <<-EOM
		[!md @$(get_type_blob_sha) !toml-type-v2]
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
		[one/uno @blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 !md "wow ok" tag-1 tag-2]
	EOM

  # reindex backfilled the config log from the old konfig object history,
  # so show-config now reads the migrated config straight from the log.
  run_dodder show-config
  assert_success
  assert_output 'blob-stores = [".default"]

[defaults]
type = "!md"
tags = []

[file-extensions]
config = "konfig"
conflict = "conflict"
lockfile = "object-lockfile"
organize = "md"
repo = "repo"
tag = "tag"
type = "type"
zettel = "zettel"

[cli-output]
print-blob_digests = true
print-colors = true
print-empty-blob_digests = false
print-flush = true
print-include-description = true
print-include-types = true
print-inventory_lists = true
print-matched-dormant = false
print-tags-always = true
print-time = true
print-unchanged = true

[cli-output.abbreviations]
zettel_ids = true
merkle_ids = true

[tools]
merge = ["vimdiff"]'

  # The migrated history is a single root entry (object-sig, no mother),
  # re-signed during migration. Its blob digest is preserved from the old
  # konfig object, so it equals the fixture konfig blob sha; the signature
  # is re-emitted (fixture key, deterministic) but matched by regexp for
  # robustness against key/format churn.
  run_dodder show-config -history
  assert_success
  # #294/FDR-0021 T4: the konfig is SELF provenance, so show-config -history
  # (no -repo_id -> empty handle) renders the bare `ed25519_pub-...` form, not
  # the foreign `dodder-repo-public_key-v1@...` purpose-prefixed form.
  assert_line --index 0 --regexp "^\[konfig @$(get_konfig_sha) [0-9.]+ ed25519_pub-[a-z0-9]+ dodder-object-sig-v2@ed25519_sig-[a-z0-9]+ !toml-config-v2\]\$"
  [ "${#lines[@]}" -eq 1 ]

  history_after_first_reindex="$output"

  # Idempotency: a second reindex must NOT re-backfill (the config log
  # now exists), so the history is byte-identical -- no duplicated entry.
  run_dodder reindex
  assert_success

  run_dodder show-config -history
  assert_success
  assert_output "$history_after_first_reindex"
  [ "${#lines[@]}" -eq 1 ]
}

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

# Line shapes: TAIs are wall-clock (dynamic), keys/signatures are per-init
# ed25519 material (dynamic), and inventory-list blob digests hash over
# signatures (dynamic) -- all matched structurally. Content-addressed blob
# digests of fixed seed content (the pandoc tool types, the test zettel
# bodies) ARE deterministic and are hardcoded per the repo assertion
# conventions.

function export_inventory_lists_lists_only { # @test
  run_dodder_init_disable_age
  create_test_zettels

  # Four commits (genesis + three zettel checkins) => four list objects,
  # TAI-ascending, wrapped in the inventory_list-v2 hyphence header.
  run_dodder export-inventory_lists
  assert_success
  assert_output --regexp - <<-'EOM'
	^---
	! inventory_list-v2
	---
	
	(\[[0-9]+\.[0-9]+ @blake2b256-[a-z0-9]+ dodder-repo-public_key-v1@ed25519_pub-[a-z0-9]+ dodder-object-sig-v3@ed25519_sig-[a-z0-9]+ !inventory_list-v2\]
	){3}\[[0-9]+\.[0-9]+ @blake2b256-[a-z0-9]+ dodder-repo-public_key-v1@ed25519_pub-[a-z0-9]+ dodder-object-sig-v3@ed25519_sig-[a-z0-9]+ !inventory_list-v2\]$
	EOM
}

function export_inventory_lists_contents { # @test
  run_dodder_init_disable_age
  create_test_zettels

  # Each list object is followed by its decoded members: the genesis list
  # carries the three pandoc tool types, each zettel-checkin list carries
  # its zettel. The third checkin rewrites one/uno, so its line also
  # carries the mother-sig lineage field.
  run_dodder export-inventory_lists -contents
  assert_success
  assert_output --regexp - <<-'EOM'
	^---
	! inventory_list-v2
	---
	
	\[[0-9]+\.[0-9]+ @blake2b256-[a-z0-9]+ dodder-repo-public_key-v1@ed25519_pub-[a-z0-9]+ dodder-object-sig-v3@ed25519_sig-[a-z0-9]+ !inventory_list-v2\]
	\[!pandoc-defaults @blake2b256-zcfmrghzp36r4r4qxtrh4t8xcd5g0f3mkpm8f3swac0vr5x503msyfsu3d [0-9]+\.[0-9]+ dodder-repo-public_key-v1@ed25519_pub-[a-z0-9]+ dodder-object-sig-v3@ed25519_sig-[a-z0-9]+ !toml-type-v2\]
	\[!pandoc-lua_filter @blake2b256-afnd989ttt3vmeunlj2asss5hjtkqe75vhupupuz2y9uv8wfx8hs6q8szw [0-9]+\.[0-9]+ dodder-repo-public_key-v1@ed25519_pub-[a-z0-9]+ dodder-object-sig-v3@ed25519_sig-[a-z0-9]+ !toml-type-v2\]
	\[!md @blake2b256-e3ew5ma0s399rmk3akms90ah2kdmr88l4jluckmdqylnlqtzu7dq60533j [0-9]+\.[0-9]+ dodder-repo-public_key-v1@ed25519_pub-[a-z0-9]+ dodder-object-sig-v3@ed25519_sig-[a-z0-9]+ !toml-type-v2 .+\]
	\[[0-9]+\.[0-9]+ @blake2b256-[a-z0-9]+ dodder-repo-public_key-v1@ed25519_pub-[a-z0-9]+ dodder-object-sig-v3@ed25519_sig-[a-z0-9]+ !inventory_list-v2\]
	\[one/uno @blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 [0-9]+\.[0-9]+ dodder-repo-public_key-v1@ed25519_pub-[a-z0-9]+ dodder-object-sig-v3@ed25519_sig-[a-z0-9]+ !md@ed25519_sig-[a-z0-9]+ "wow ok" tag-1 tag-2\]
	\[[0-9]+\.[0-9]+ @blake2b256-[a-z0-9]+ dodder-repo-public_key-v1@ed25519_pub-[a-z0-9]+ dodder-object-sig-v3@ed25519_sig-[a-z0-9]+ !inventory_list-v2\]
	\[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd [0-9]+\.[0-9]+ dodder-repo-public_key-v1@ed25519_pub-[a-z0-9]+ dodder-object-sig-v3@ed25519_sig-[a-z0-9]+ !md@ed25519_sig-[a-z0-9]+ "wow ok again" tag-3 tag-4\]
	\[[0-9]+\.[0-9]+ @blake2b256-[a-z0-9]+ dodder-repo-public_key-v1@ed25519_pub-[a-z0-9]+ dodder-object-sig-v3@ed25519_sig-[a-z0-9]+ !inventory_list-v2\]
	\[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd [0-9]+\.[0-9]+ dodder-repo-public_key-v1@ed25519_pub-[a-z0-9]+ dodder-object-mother-sig-v3@ed25519_sig-[a-z0-9]+ dodder-object-sig-v3@ed25519_sig-[a-z0-9]+ !md@ed25519_sig-[a-z0-9]+ "wow the first" tag-3 tag-4\]$
	EOM
}

function export_inventory_lists_survives_cache_wipe { # @test
  run_dodder_init_disable_age
  create_test_zettels

  run_dodder export-inventory_lists -contents
  assert_success
  before="$output"

  # The whole point (dodder recovery toolbox): the output must depend only
  # on the inventory-list log and the blob store, never the stream index.
  # Wipe the rebuildable cache category (which holds the index) and assert
  # the export is byte-identical.
  rm -rf .dodder/local/cache

  run_dodder export-inventory_lists -contents
  assert_success
  assert_output "$before"
}

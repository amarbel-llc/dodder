#! /usr/bin/env bats

# Tests for JSON inventory list format specifically testing signature verification

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

  # for shellcheck SC2154
  export output
}

teardown() {
  chflags_nouchg
}

function json_init_and_checkin { # @test
  # Test that JSON inventory list format works end-to-end with signature verification

  # Initialize repo with JSON inventory list type
  run_dodder init \
    -yin <(cat_yin) \
    -yang <(cat_yang) \
    -repo_id . \
    -encryption generate \
    -inventory_list-type inventory_list-json-v0 \
    test-repo-id

  assert_success

  # Verify inventory list is JSON format
  run_dodder show :b
  assert_success
  assert_output --regexp - <<-'EOM'
		\[[0-9]+\.[0-9]+ @blake2b256-.+ !inventory_list-json-v0]
	EOM

  # Initialize workspace
  run_dodder init-workspace -experimental-repo=false
  assert_success

  # Checkout files
  run_dodder checkout :t,e
  assert_success

  # Create one directory and modify a file
  mkdir -p one

  cat >one/uno.zettel <<-EOM
		---
		# modified with json format
		- test-tag
		! md
		---

		test body
	EOM

  # Checkin the file - this will test signature creation with JSON format
  run_dodder checkin one/uno.zettel
  assert_success
  # Just verify it succeeded - the actual file name might vary
  assert_output --regexp '\[one/[^ ]+ @blake2b256-.+ !md "modified with json format" test-tag\]'

  # Show the inventory list to verify it's still JSON
  run_dodder show :b
  assert_success
  assert_output --regexp - <<-'EOM'
		\[[0-9]+\.[0-9]+ @blake2b256-.+ !inventory_list-json-v0]
	EOM
}

function json_checkin_multiple_versions { # @test
  # Test that JSON format preserves signatures across multiple versions

  # Initialize repo with JSON inventory list type
  run_dodder init \
    -yin <(cat_yin) \
    -yang <(cat_yang) \
    -repo_id . \
    -encryption generate \
    -inventory_list-type inventory_list-json-v0 \
    test-repo-id

  assert_success

  # Initialize workspace
  run_dodder init-workspace -experimental-repo=false
  assert_success

  # Create a zettel
  mkdir -p test
  cat >test/example.zettel <<-EOM
		---
		# version 1
		- tag-1
		! md
		---

		body 1
	EOM

  run_dodder checkin test/example.zettel
  assert_success

  # Create version 2
  cat >test/example.zettel <<-EOM
		---
		# version 2
		- tag-2
		! md
		---

		body 2
	EOM

  run_dodder checkin test/example.zettel
  assert_success

  # Verify both versions are in JSON format
  run_dodder show :b
  assert_success
  assert_output --regexp - <<-'EOM'
		\[[0-9]+\.[0-9]+ @blake2b256-.+ !inventory_list-json-v0]
	EOM
}

function json_signature_verification { # @test
  # Test that signature verification works correctly with JSON inventory lists

  # Create a repo with JSON format
  run_dodder init \
    -yin <(cat_yin) \
    -yang <(cat_yang) \
    -repo_id . \
    -encryption generate \
    -inventory_list-type inventory_list-json-v0 \
    test-repo-id

  assert_success

  # Initialize workspace
  run_dodder init-workspace -experimental-repo=false
  assert_success

  # Checkout
  run_dodder checkout :t
  assert_success

  # Create directory and file
  mkdir -p one
  cat >one/uno.zettel <<-EOM
		---
		# version 1
		- tag-1
		! md
		---

		body 1
	EOM

  run_dodder checkin one/uno.zettel
  assert_success

  cat >one/uno.zettel <<-EOM
		---
		# version 2
		- tag-2
		! md
		---

		body 2
	EOM

  run_dodder checkin one/uno.zettel
  assert_success

  # Verify fsck passes (includes signature verification)
  run_dodder fsck
  assert_success

  # Show should work without signature errors - verify any zettels show
  run_dodder show :z
  assert_success
  # Just verify we get output with the tag-2
  assert_output --regexp 'tag-2'
}

# bats test_tags=user_story:referenced_objects
function json_inventory_list_preserves_all_lock_kinds { # @test
  # Verify that type locks, tag locks, referenced object locks + aliases,
  # and blob references + type locks + aliases survive the JSON inventory
  # list persist→read cycle.

  run_dodder init \
    -yin <(cat_yin) \
    -yang <(cat_yang) \
    -repo_id . \
    -encryption generate \
    -inventory_list-type inventory_list-json-v0 \
    test-repo-id

  assert_success

  run_dodder init-workspace -experimental-repo=false
  assert_success

  # Create a type with reference discovery that outputs typed blob refs
  cat >ref-blob.type <<-'TYPEFILE'
		---
		! toml-type-v2
		---

		file-extension = "md"
		vim-syntax-type = "markdown"

		[references]
		shell = ["bash", "-c"]
		script = "grep -oP '(@blake2b256-[a-z0-9]+|\\[\\[(.+?)\\]\\])' | sed 's/\\[\\[//;s/\\]\\]//' | sed 's/^@\\(blake2b256-[a-z0-9]*\\)/@\\1 !md/'"
	TYPEFILE

  run_dodder checkin -delete ref-blob.type
  assert_success

  # Checkout types so they're available
  run_dodder checkout :t,e
  assert_success

  # Create a referenced zettel first (one/uno gets assigned)
  run_dodder new -edit=false - <<-EOM
		---
		# target zettel
		! md
		---

		target content
	EOM
  assert_success

  # Create a zettel with both a wiki-link reference and a blob reference
  run_dodder new -edit=false - <<-EOM
		---
		# zettel with all lock kinds
		- test-tag
		! ref-blob
		---

		See [[one/uno]] and blob @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd for details.
	EOM
  assert_success

  # Show in text format — verify type lock, tag, referenced object, and blob reference
  run_dodder show -format text one/dos:
  assert_success

  # Type lock present
  assert_output --regexp '! ref-blob@ed25519_sig-.+'
  # Tag present
  assert_output --partial 'test-tag'
  # Referenced object with lock
  assert_output --regexp '- one/uno@ed25519_sig-.+'
  # Blob reference with type lock
  assert_output --regexp '@blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md@ed25519_sig-.+'
}

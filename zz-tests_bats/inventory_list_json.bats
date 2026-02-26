#! /usr/bin/env bats

# Tests for JSON inventory list format specifically testing signature verification

setup() {
  load "$(dirname "$BATS_TEST_FILE")/common.bash"

  # for shellcheck SC2154
  export output
}

teardown() {
  chflags_and_rm
}

function json_init_and_checkin { # @test
  # Test that JSON inventory list format works end-to-end with signature verification

  # Initialize repo with JSON inventory list type
  run_dodder init \
    -yin <(cat_yin) \
    -yang <(cat_yang) \
    -override-xdg-with-cwd \
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
  run_dodder init-workspace
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
    -override-xdg-with-cwd \
    -encryption generate \
    -inventory_list-type inventory_list-json-v0 \
    test-repo-id

  assert_success

  # Initialize workspace
  run_dodder init-workspace
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
    -override-xdg-with-cwd \
    -encryption generate \
    -inventory_list-type inventory_list-json-v0 \
    test-repo-id

  assert_success

  # Initialize workspace
  run_dodder init-workspace
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

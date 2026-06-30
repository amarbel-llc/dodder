#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

  # for shellcheck SC2154
  export output
}

teardown() {
  chflags_nouchg
}

# bats file_tags=init

# init-default creates a CWD-local repository with no -yin/-yang files
# supplied: the zettel-id vocabulary comes from the embedded defaults and
# the signing key is auto-detected from the SSH agent (or generated fresh
# when none is available, as in the sandbox -- hence the warning line,
# which is ignored by the per-line assertions here). A fresh store
# generates a new key each run, so the object blob digests are matched by
# regexp.
function init_default_creates_repo { # @test
  run_dodder init-default
  assert_success
  assert_line --regexp '^\[!md @blake2b256-.+ !toml-type-v2\]$'

  run test -f .dodder/local/share/repos/default/config-seed
  assert_success
}

# A second init-default in an already-initialized directory is a no-op
# (init is not re-runnable), so the bootstrap is safe to re-run: it
# returns before printing anything.
function init_default_idempotent { # @test
  run_dodder init-default
  assert_success

  run_dodder init-default
  assert_success
  assert_output ''
}

# Generated zettel ids draw from the embedded default vocabulary: the
# first id is the first yin word over the first yang word (arbor/amber).
# The blob digest is over the body content (deterministic).
function init_default_seeds_embedded_vocab { # @test
  run_dodder init-default
  assert_success

  run_dodder init-workspace -experimental-repo=false
  assert_success

  run_dodder new -edit=false - <<-EOM
		---
		# a note
		! md
		---

		body
	EOM
  assert_success
  assert_output '[arbor/amber @blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc !md "a note"]'
}

#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

  # for shellcheck SC2154
  export output
}

teardown() {
  chflags_nouchg
}

function generate { # @test
  function run_dodder {
    cmd="$1"
    shift
    #shellcheck disable=SC2068,SC2154
    run timeout --preserve-status "2s" "$DODDER_BIN" "$cmd" ${cmd_dodder_def_no_debug[@]} "$@"
  }

  run_dodder info store-version
  assert_success
  assert_output --regexp '[0-9]+'

  # shellcheck disable=SC2034
  storeVersionCurrent="$output"

  run_dodder_init_disable_age

  run_dodder show :b
  assert_success
  assert_output

  run_dodder last
  assert_success
  assert_output

  run_dodder info store-version
  assert_success
  assert_output "$storeVersionCurrent"

  # FDR-0020: config is no longer a queryable object, so the type is
  # checked via the genre query here and the config via show-config below
  # (the bare `:konfig` query now errors with "config is no longer an
  # object; use show-config / edit-config").
  run_dodder show !md:t
  assert_success
  # Pandoc tools default-on (#208): the !md type carries three blob references
  # after its type, so the box line no longer ends at !toml-type-v2].
  assert_line --regexp '\[!md @blake2b256-.+ !toml-type-v2 .+]'

  run_dodder show-config
  assert_success
  assert_output - <<-EOM
		blob-stores = [".default"]

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
		merge = ["vimdiff"]
	EOM

  create_test_zettels

  run_dodder show -format tags one/uno
  assert_success
  assert_output "tag-3, tag-4"
}

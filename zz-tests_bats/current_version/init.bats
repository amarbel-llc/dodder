#! /usr/bin/env bats

setup() {
	load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

	# for shellcheck SC2154
	export output
}

teardown() {
	chflags_nouchg
}

function init_compression { # @test
	run_dodder info store-version
	assert_success
	assert_output --regexp '[0-9]+'

	# shellcheck disable=SC2034
	storeVersionCurrent="$output"

	run_dodder_init_disable_age

	function output_immutable_config() {
		cat - <<-EOM
			---
			! toml-config-immutable-v2
			---

			public-key = "dodder-repo-public_key-v1.*"
			store-version = $storeVersionCurrent
			id = ""
			inventory_list-type = "!inventory_list-v2"
			object-sig-type = "dodder-object-sig-v2"
		EOM
	}

	run_dodder info-repo config-immutable
	assert_success
	output_immutable_config | assert_output --regexp -

	run_madder cat "$(get_konfig_sha)"
	assert_success
	assert_output
}

function init_and_reindex { # @test
	run_dodder_init_disable_age

	run test -f .dodder/local/share/repos/default/config-seed
	assert_success

	# Config is read through show-config (FDR 0020); :konfig no longer queries.
	run_dodder show-config
	assert_success
	assert_output

	run_dodder reindex
	assert_success
	run_dodder show :t
	assert_success
	assert_output_unsorted - <<-EOM
		[!md @$(get_type_blob_sha) !toml-type-v2]
	EOM

	run_dodder reindex
	assert_success
	run_dodder show :t
	assert_success
	assert_output_unsorted - <<-EOM
		[!md @$(get_type_blob_sha) !toml-type-v2]
	EOM
}

function init_and_deinit { # @test
	run_dodder_init_disable_age

	run test -f .dodder/local/share/repos/default/config-seed
	assert_success

	# Config is read through show-config (FDR 0020); :konfig no longer queries.
	run_dodder show-config
	assert_success
	assert_output

	# run_dodder deinit
	# assert_success
	# assert_output wow

	# run test ! -f .dodder/KonfigAngeboren
	# run ls .dodder/
	# assert_success
	# assert_output wow
}

function init_and_with_another_age { # @test
	run_dodder_init
	age_id="$("$DODDER_BIN" gen madder-private_key-v1)"

	mkdir inner
	pushd inner || exit 1

	run_dodder_init -yin <(cat_yin) -yang <(cat_yang) -encryption "$age_id" .default
	assert_success

	run_madder info-repo encryption
	assert_success
	assert_output "$age_id"
}

function init_with_non_xdg { # @test
	run_dodder_init
	run tree .dodder
	assert_output

	run_dodder show +t
	assert_success
	assert_output_unsorted - <<-EOM
		[!md @$(get_type_blob_sha) !toml-type-v2]
	EOM
}

function non_repo_failure { # @test
	set_xdg "$BATS_TEST_TMPDIR"
	run_dodder show +t
	assert_failure
	assert_output --partial 'not in a dodder directory'
}

function init_and_init { # @test
	run_dodder_init
	assert_success

	{
		echo "---"
		echo "# wow"
		echo "- tag"
		echo "! md"
		echo "---"
		echo
		echo "body"
	} >to_add

	run_dodder new -edit=false to_add
	assert_success
	assert_output_unsorted - <<-EOM
		[one/uno @blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc !md "wow" tag]
	EOM

	run_dodder show one/uno
	assert_success
	assert_output - <<-EOM
		[one/uno @blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc !md "wow" tag]
	EOM

	run_dodder init .default
	assert_failure
	assert_output --partial ': file exists'

	run_dodder show one/uno
	assert_success
	assert_output - <<-EOM
		[one/uno @blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc !md "wow" tag]
	EOM

	run_dodder show :
	assert_success
	assert_output - <<-EOM
		[one/uno @blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc !md "wow" tag]
	EOM
}

function init_without_age { # @test
	run_dodder_init_disable_age
	assert_success
}

function init_with_age { # @test
	run_dodder init \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-encryption generate \
		.default

	assert_success
	assert_output - <<-EOM
		[!md @blake2b256-45v3c002j9xfjguu2a7ljxnf68tqglg8fa0csjgnn7d2n36ltp0snfjxgj !toml-type-v2]
	EOM

	run test -f .xdg/data/dodder/config-permanent

	run_madder info-repo encryption
	assert_success
	assert_output
}

function init_inventory_archive_with_encryption { # @test
	run_dodder_init_disable_age
	assert_success

	run_madder init-inventory-archive -encryption generate .archive
	assert_success

	run_madder info-repo .archive encryption
	assert_success
	assert_output --regexp '.+'
}

function init_blob_store_id_missing_fails_fast { # @test
	# Counterpart to init_with_existing_madder_store: when the named
	# store does NOT exist, `dodder init -blob_store-id` should fail
	# with a useful error. Previously it reported success and produced
	# an unusable repo that panicked on every subsequent command with
	# "WriteTo: no write store given" (amarbel-llc/dodder#214).
	set_xdg "$BATS_TEST_TMPDIR"

	# Intentionally do NOT create the named store before init.
	run_dodder init \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-encryption none \
		-blob_store-id never-created \
		default

	assert_failure
	assert_output --partial 'blob store'
	assert_output --partial 'never-created'
}

function init_with_existing_madder_store { # @test
	set_xdg "$BATS_TEST_TMPDIR"

	# Create a user-scoped madder blob store before dodder init
	run_madder init shared
	assert_success

	# Init dodder with the pre-existing blob store
	run_dodder init \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-encryption none \
		-blob_store-id shared \
		default

	assert_success
	assert_output --regexp - <<-'EOM'
		\[!md @blake2b256-[[:alnum:]]+ !toml-type-v2]
	EOM

	run_dodder init-workspace -experimental-repo=false

	# Verify dodder last shows the inventory list with the init objects
	run_dodder last -format inventory_list-sans-tai
	assert_success
	assert_output_unsorted --regexp - <<-'EOM'
		\[!md @blake2b256-[[:alnum:]]+ .* !toml-type-v2]
	EOM
}

function init_with_json_inventory_list_type { # @test
	run_dodder init \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-encryption generate \
		-inventory_list-type inventory_list-json-v0 \
		.default

	assert_success
	assert_output - <<-EOM
		[!md @blake2b256-45v3c002j9xfjguu2a7ljxnf68tqglg8fa0csjgnn7d2n36ltp0snfjxgj !toml-type-v2]
	EOM

	run_dodder show :b
	assert_success
	assert_output --regexp - <<-'EOM'
		\[[0-9]+\.[0-9]+ @blake2b256-.+ !inventory_list-json-v0]
	EOM

	run_dodder last
	assert_success
	assert_output

	run test -f .xdg/data/dodder/config-permanent

	run_madder info-repo encryption
	assert_success
	assert_output
}

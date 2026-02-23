#! /usr/bin/env bats

setup() {
	load "$(dirname "$BATS_TEST_FILE")/common.bash"

	# for shellcheck SC2154
	export output
}

teardown() {
	chflags_and_rm
}

function init_compression { # @test
	run_dodder info store-version
	assert_success
	assert_output --regexp '[0-9]+'

	# shellcheck disable=SC2034
	storeVersionCurrent="$output"

	run_dodder_init_disable_age

	function output_immutable_config() {
		if [[ "$storeVersionCurrent" -le 10 ]]; then
			cat - <<-EOM
				---
				! toml-config-immutable-v1
				---

				public-key = 'dodder-repo-public_key-v1.*'
				store-version = $storeVersionCurrent
				repo-type = 'working-copy'
				id = 'test-repo-id'
				inventory_list-type = '!inventory_list-v2'

				\[blob-store]
				compression-type = 'zstd'
				lock-internal-files = false
			EOM
		else
			cat - <<-EOM
				---
				! toml-config-immutable-v2
				---

				public-key = 'dodder-repo-public_key-v1.*'
				store-version = $storeVersionCurrent
				id = 'test-repo-id'
				inventory_list-type = '!inventory_list-v2'
				object-sig-type = 'dodder-object-sig-v2'
			EOM
		fi
	}

	run_dodder info-repo config-immutable
	assert_success
	output_immutable_config | assert_output --regexp -

	run_dodder blob_store-cat "$(get_konfig_sha)"
	assert_success
	assert_output
}

function init_and_reindex { # @test
	run_dodder_init_disable_age

	run test -f .dodder/local/share/config-seed
	assert_success

	run_dodder show -format log :konfig
	assert_success
	assert_output - <<-EOM
		[konfig @$(get_konfig_sha) !toml-config-v2]
	EOM

	run_dodder reindex
	assert_success
	run_dodder show :t,konfig
	assert_success
	assert_output_unsorted - <<-EOM
		[!md @$(get_type_blob_sha) !toml-type-v1]
		[konfig @$(get_konfig_sha) !toml-config-v2]
	EOM

	run_dodder reindex
	assert_success
	run_dodder show :t,konfig
	assert_success
	assert_output_unsorted - <<-EOM
		[!md @$(get_type_blob_sha) !toml-type-v1]
		[konfig @$(get_konfig_sha) !toml-config-v2]
	EOM
}

function init_and_deinit { # @test
	run_dodder_init_disable_age

	run test -f .dodder/local/share/config-seed
	assert_success

	# run cat .dodder/Objekten/Akten/c1/a8ed3cf288dd5d7ccdfd6b9c8052a925bc56be2ec97ed0bb345ab1d961c685
	# assert_output wow
	run_dodder show -format log :konfig
	assert_success
	assert_output - <<-EOM
		[konfig @$(get_konfig_sha) !toml-config-v2]
	EOM

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

	run_dodder_init -yin <(cat_yin) -yang <(cat_yang) -encryption "$age_id" test-repo-id
	assert_success

	run_dodder blob_store-info-repo encryption
	assert_success
	assert_output "$age_id"
}

function init_with_non_xdg { # @test
	run_dodder_init -override-xdg-with-cwd test-repo-id
	run tree .dodder
	assert_output

	run_dodder show +konfig,t
	assert_success
	assert_output_unsorted - <<-EOM
		[!md @$(get_type_blob_sha) !toml-type-v1]
		[konfig @$(get_konfig_sha) !toml-config-v2]
	EOM
}

function non_repo_failure { # @test
	set_xdg "$BATS_TEST_TMPDIR"
	run_dodder show +konfig,t
	assert_failure
	assert_output --partial 'not in a dodder directory'
}

function init_and_init { # @test
	run_dodder_init test-repo-id
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

	run_dodder init -lock-internal-files=false -override-xdg-with-cwd test-repo-id
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
    -override-xdg-with-cwd \
		-encryption generate \
		test-repo-id

	assert_success
	assert_output - <<-EOM
		[!md @blake2b256-3kj7xgch6rjkq64aa36pnjtn9mdnl89k8pdhtlh33cjfpzy8ek4qnufx0m !toml-type-v1]
		[konfig @$(get_konfig_sha) !toml-config-v2]
	EOM

	run test -f .xdg/data/dodder/config-permanent

	run_dodder blob_store-info-repo encryption
	assert_success
	assert_output
}

function init_inventory_archive_with_encryption { # @test
	run_dodder_init_disable_age
	assert_success

	run_dodder blob_store-init-inventory-archive -encryption generate .archive
	assert_success

	run_dodder blob_store-info-repo .archive encryption
	assert_success
	assert_output --regexp '.+'
}

function init_with_json_inventory_list_type { # @test
	run_dodder init \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
    -override-xdg-with-cwd \
		-encryption generate \
		-inventory_list-type inventory_list-json-v0 \
		test-repo-id

	assert_success
	assert_output - <<-EOM
		[!md @blake2b256-3kj7xgch6rjkq64aa36pnjtn9mdnl89k8pdhtlh33cjfpzy8ek4qnufx0m !toml-type-v1]
		[konfig @$(get_konfig_sha) !toml-config-v2]
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

	run_dodder blob_store-info-repo encryption
	assert_success
	assert_output
}

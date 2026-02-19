#! /usr/bin/env bats

setup() {
	load "$(dirname "$BATS_TEST_FILE")/common.bash"

	# for shellcheck SC2154
	export output
}

teardown() {
	chflags_and_rm
}

function last_after_init { # @test
	run_dodder_init_disable_age

	run_dodder last -format inventory_list-sans-tai
	assert_success
	assert_output_unsorted --regexp - <<-EOM
		\\[!md @$(get_type_blob_sha) .* !toml-type-v1]
		\\[konfig @$(get_konfig_sha) .* !toml-config-v2]
	EOM
}

function last_after_type_mutate { # @test
	run_dodder_init_disable_age

	run_dodder show :b
	assert_success
	assert_output --regexp - <<-'EOM'
		\[[0-9]+\.[0-9]+ @blake2b256-[[:alnum:]]+ !inventory_list-v2]
	EOM

	cat >md.type <<-EOM
		inline-akte = false
		vim-syntax-type = "test"
	EOM

	run_dodder checkin .t
	assert_success
	assert_output - <<-EOM
		[!md @blake2b256-473260as3d3pd4uramcc60877srvpkxs4krlap45dkl3mfvq2npq2duvvq !toml-type-v1]
	EOM

	run_dodder show :b
	assert_success
	assert_output --regexp - <<-'EOM'
		\[[0-9]+\.[0-9]+ @blake2b256-[[:alnum:]]+ !inventory_list-v2]
		\[[0-9]+\.[0-9]+ @blake2b256-[[:alnum:]]+ !inventory_list-v2]
	EOM

	run_dodder show -format blob :b
	assert_success
	assert_output --regexp - <<-EOM
		\\[!md @blake2b256-473260as3d3pd4uramcc60877srvpkxs4krlap45dkl3mfvq2npq2duvvq .* !toml-type-v1]
	EOM

	run_dodder last -format inventory_list-sans-tai
	assert_success
	assert_output --regexp - <<-EOM
		\\[!md @blake2b256-473260as3d3pd4uramcc60877srvpkxs4krlap45dkl3mfvq2npq2duvvq .* !toml-type-v1]
	EOM
}

function last_organize { # @test
	run_dodder_init_disable_age

	cat >md.type <<-EOM
		binary = false
		vim-syntax-type = "test"
	EOM

	run_dodder checkin .t
	assert_success
	assert_output - <<-EOM
		[!md @blake2b256-tugmx90k7ajv6atknze43ptgphz08x4f929c0f0n4y394nh5gh7qmau4w9 !toml-type-v1]
	EOM

	function editor() {
		# shellcheck disable=SC2317
		cat - >"$1" <<-EOM
			- [!md !toml-type-v1 added-tag]
		EOM
	}

	export -f editor

	# shellcheck disable=SC2016
	export EDITOR='bash -c "editor $0"'

	run_dodder last -organize
	assert_success
	assert_output - <<-EOM
		[!md @blake2b256-tugmx90k7ajv6atknze43ptgphz08x4f929c0f0n4y394nh5gh7qmau4w9 !toml-type-v1 added-tag]
	EOM
}

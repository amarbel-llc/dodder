#! /usr/bin/env bats

setup() {
	load "$(dirname "$BATS_TEST_FILE")/common.bash"

	# for shellcheck SC2154
	export output

	copy_from_version "$DIR"

  run_dodder_init_workspace
}

teardown() {
	chflags_and_rm
}

function reindex_simple { # @test
	run_dodder reindex
	assert_success
	run_dodder show +t,e,z,konfig
	assert_success
	assert_output_unsorted - <<-EOM
		[!md @$(get_type_blob_sha) !toml-type-v1]
		[konfig @$(get_konfig_sha) !toml-config-v2]
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
		[one/uno @blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 !md "wow ok" tag-1 tag-2]
	EOM

	run_dodder show -format tags-path :e,z,t
	assert_success
	assert_output_unsorted - <<-EOM
		!md [Paths: [], All: []]
		one/dos [Paths: [TypeDirect:[tag-3] TypeDirect:[tag-4]], All: [tag-3:[TypeDirect:[tag-3]] tag-4:[TypeDirect:[tag-4]]]]
		one/uno [Paths: [TypeDirect:[tag-3] TypeDirect:[tag-4]], All: [tag-3:[TypeDirect:[tag-3]] tag-4:[TypeDirect:[tag-4]]]]
	EOM
}

function reindex_simple_twice { # @test
	expected="$(mktemp)"
	cat - >"$expected" <<-EOM
		[!md @$(get_type_blob_sha) !toml-type-v1]
		[konfig @$(get_konfig_sha) !toml-config-v2]
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
		[one/uno @blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 !md "wow ok" tag-1 tag-2]
	EOM

	run_dodder reindex
	assert_success
	run_dodder show +e,t,z,konfig
	assert_success
	assert_output_unsorted - <"$expected"

	run_dodder reindex
	assert_success
	run_dodder show +e,t,z,konfig
	assert_success
	assert_output_unsorted - <"$expected"
}

function reindex_after_changes { # @test
	run_dodder show !md:t
	assert_success
	assert_output - <<-EOM
		[!md @$(get_type_blob_sha) !toml-type-v1]
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

	function verify() {
		run_dodder show -format blob !md+t
		assert_success
		assert_output - <<-EOM
			file-extension = 'md'
			vim-syntax-type = 'markdown'
			inline-akte = false
			vim-syntax-type = "test"
		EOM

		run_dodder show -format blob !md:t
		assert_success
		assert_output - <<-EOM
			inline-akte = false
			vim-syntax-type = "test"
		EOM
	}

	verify

	run_dodder reindex
	assert_success
	run_dodder show +e,t,z,konfig
	assert_success
	assert_output_unsorted - <<-EOM
		[!md @$(get_type_blob_sha) !toml-type-v1]
		[!md @blake2b256-473260as3d3pd4uramcc60877srvpkxs4krlap45dkl3mfvq2npq2duvvq !toml-type-v1]
		[konfig @$(get_konfig_sha) !toml-config-v2]
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
		[one/uno @blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 !md "wow ok" tag-1 tag-2]
	EOM

	verify
}

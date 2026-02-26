#! /usr/bin/env bats

setup() {
	load "$(dirname "$BATS_TEST_FILE")/common.bash"

	# for shellcheck SC2154
	export output

	copy_from_version "$DIR"
}

teardown() {
	chflags_and_rm
}

function format_mother_sha_one { # @test
	run_dodder show -format sig-bytes-hex one/uno+
	assert_success
	sig="$(echo -n "$output" | head -n1)"

	run_dodder show -format sig-mother-bytes-hex one/uno
	assert_success
	assert_output - <<-EOM
		$sig
	EOM
}

function format_mother_one { # @test
	run_dodder_debug show -format sig one/uno+
	mother_sig="$(run_dodder_debug show -format sig one/uno+ | head -n 1 | cut -d@ -f2)"

	run_dodder show -format sig-mother one/uno
	assert_success
	assert_output "dodder-object-mother-sig-v2@$mother_sig"

	run_dodder show "dodder-object-sig-v2@$mother_sig"+
	assert_success
	assert_output - <<-EOM
		[one/uno @blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 !md "wow ok" tag-1 tag-2]
	EOM
}

function format_mother_all { # @test
	run_dodder show -format mother :
	assert_success
	assert_output - <<-EOM
		[one/uno @blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 !md "wow ok" tag-1 tag-2]
	EOM
}

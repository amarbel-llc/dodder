#! /usr/bin/env bats

setup() {
	load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

	# for shellcheck SC2154
	export output

	copy_from_version "$DIR"

	run_dodder init-workspace -experimental-repo=false
	assert_success

	run_dodder checkout -mode both :z,t,e
	assert_success

	export BATS_TEST_BODY=true
}

teardown() {
	chflags_nouchg
}

function diff_all_same { # @test
	run_dodder diff .
	assert_success
	assert_output_unsorted - <<-EOM
	EOM
}

function diff_all_diff { # @test
	echo wowowow >>one/uno.md
	run_dodder diff one/uno.zettel
	assert_success
	assert_output - <<-EOM
		--- one/uno:zettel
		+++ one/uno.zettel
		@@ -2,6 +2,6 @@
		 # wow the first
		 - tag-3
		 - tag-4
		-@ blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd
		+@ blake2b256-3608lk68nnstfff38qpht4zqdcvahygkksqcnmq5qsm2vjzlsufqt4wr0k
		 ! md@ed25519_sig-2a3ehc2jherahnn05tr9m62zc0sp9s8l8r7h4a9npj92rljd892a9kh62hawyujw475enup3v2z9dy0wlvam30l0lxz0j3n4huu3spsmz36y7
		 ---
	EOM
}

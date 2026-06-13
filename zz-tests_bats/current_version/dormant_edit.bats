#! /usr/bin/env bats

setup() {
	load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

	# for shellcheck SC2154
	export output

	copy_from_version "$DIR"
}

teardown() {
	chflags_nouchg
}

# bats file_tags=user_story:config

function dormant_edit_and_change { # @test
	# EDITOR appends a TOML comment — semantically a no-op but the
	# blob bytes change, so a new digest is announced.
	export EDITOR="bash -c 'echo \"# dormant-edit smoke comment\" >> \"\$0\"'"
	run_dodder dormant-edit
	assert_success
	assert_output --regexp '^\[konfig @blake2b256-[a-z0-9]+ !toml-config-v2\]$'

	new_digest="${output##*@}"
	new_digest="${new_digest%% *}"
	[[ $new_digest != "$(get_konfig_sha)" ]] \
		|| fail "konfig digest unchanged after dormant-edit (got $new_digest)"

	# Round-trip: the next MakeLocalWorkingCopy call invokes
	# loadMutableConfigBlob on the new blob. If the editor flow wrote
	# bytes the load path can't decode, this `show` fails.
	run_dodder show +konfig
	assert_success
	assert_output --regexp '\[konfig @blake2b256-[a-z0-9]+ !toml-config-v2\]'

	# dormant-edit also appends the new config state to the config log
	# (FDR 0020), so show-config -history lists the single new entry.
	# Signatures and tai are non-deterministic; the blob digest matches
	# the digest dormant-edit announced above.
	run_dodder show-config -history
	assert_success
	assert_output --regexp "^\[konfig @${new_digest} [0-9.]+ dodder-repo-public_key-v1@ed25519_pub-[a-z0-9]+ dodder-object-sig-v2@ed25519_sig-[a-z0-9]+ !toml-config-v2\]\$"
}

function dormant_edit_and_dont_change { # @test
	export EDITOR=true
	run_dodder dormant-edit
	assert_success
	assert_output ''

	run_dodder show -format object-id-blob-digest :konfig
	assert_success
	digest="${output##* }"
	[[ $digest == "$(get_konfig_sha)" ]] \
		|| fail "expected fixture digest $(get_konfig_sha), got $digest"
}

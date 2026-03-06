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

function edit_config_and_change { # @test
	export EDITOR="/bin/bash -c 'echo \"# this is the body 2\" >> \"\$0\"'"
	run_dodder edit-config
	assert_success
	assert_output - <<-EOM
		[konfig @blake2b256-tj6za035l8qrefq9ctz0scyazvwf6jv2xtzzye42ghyn3z8kgzeq2jvtnj !toml-config-v2]
	EOM
}

function edit_config_and_dont_change { # @test
	export EDITOR="true"
	run_dodder edit-config
	assert_success
	assert_output ''
}

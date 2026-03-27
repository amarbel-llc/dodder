#! /usr/bin/env bats

setup() {
	load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

	# for shellcheck SC2154
	export output
}

teardown() {
	chflags_nouchg
}

# bats test_tags=pandoc
function init_with_pandoc_tools_creates_type_objects { # @test
	wd="$(mktemp -d)"
	cd "$wd" || exit 1

	run_dodder init \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-encryption none \
		-repo_id . \
		-lock-internal-files=false \
		-include-default-pandoc-tools \
		test-repo-id

	assert_success

	# Genesis should create the pandoc tool types alongside !md and konfig
	assert_line --regexp '\[!pandoc-defaults @blake2b256-.+ !toml-type-v1]'
	assert_line --regexp '\[!pandoc-lua_filter @blake2b256-.+ !toml-type-v1]'

	run_dodder init-workspace -experimental-repo=false

	# The !md type should have blob references for the pandoc tools
	run_dodder show '!md:t'
	assert_success
	assert_output --partial 'pandoc-lua_filter'
	assert_output --partial 'pandoc-defaults'
}

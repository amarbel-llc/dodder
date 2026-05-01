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
		-include-default-pandoc-tools \
		test-repo-id

	assert_success

	# Genesis should create the pandoc tool types alongside !md and konfig
	assert_line --regexp '\[!pandoc-defaults @blake2b256-.+ !toml-type-v2]'
	assert_line --regexp '\[!pandoc-lua_filter @blake2b256-.+ !toml-type-v2]'

	run_dodder init-workspace -experimental-repo=false

	# The !md type should have blob references for the pandoc tools
	run_dodder show '!md:t'
	assert_success
	assert_output --partial 'pandoc-lua_filter'
	assert_output --partial 'pandoc-defaults'
}

# bats test_tags=pandoc
function format_blob_stdin_pandoc_normalizes_markdown { # @test
	wd="$(mktemp -d)"
	cd "$wd" || exit 1

	run_dodder init \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-encryption none \
		-repo_id . \
		-include-default-pandoc-tools \
		test-repo-id

	assert_success

	run_dodder init-workspace -experimental-repo=false

	# Badly-formatted markdown: inconsistent spacing, long unwrapped line
	run_dodder format-blob -stdin text !md <<-'EOM'
		-    item    one
		-  item   two
		-       item three

		this is a paragraph that is way too long and should be wrapped by pandoc because it exceeds the column width of eighty characters which pandoc enforces
	EOM
	assert_success

	# Pandoc normalizes list spacing and wraps long lines at 80 columns
	assert_output - <<-'EOM'
		- item one

		- item two

		-   item three

		this is a paragraph that is way too long and should be wrapped by pandoc because
		it exceeds the column width of eighty characters which pandoc enforces
	EOM
}

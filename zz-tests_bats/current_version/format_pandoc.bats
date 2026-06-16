#! /usr/bin/env bats

# bats file_tags=pandoc

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
		-repo_id .default \
		-include-default-pandoc-tools \
		test-repo-id

	assert_success

	# Genesis should create the pandoc tool types alongside !md
	assert_line --regexp '\[!pandoc-defaults @blake2b256-.+ !toml-type-v2]'
	assert_line --regexp '\[!pandoc-lua_filter @blake2b256-.+ !toml-type-v2]'

	run_dodder init-workspace -experimental-repo=false

	# The !md type metadata should carry exactly three blob references,
	# sorted by content-addressed blob digest. Blob digests are pinned
	# (deterministic from embedded file content); ed25519_sig values
	# vary per init so they are matched as .* via --regexp.
	run_dodder show '!md:t'
	assert_success
	assert_output --regexp - <<-'EOM'
		\[!md @blake2b256-wn23tupt0wdt5ha776v4pavqley07cfazp3mzgsk5t83fgw4k6aqltcd7j !toml-type-v2 "defaults/dodder-edit\.yaml<@blake2b256-amzdh9dljzhu9885kmh654zkyys5mxq62eadx3ej8hwf3ypwd3qq00chz7 !pandoc-defaults@ed25519_sig-[a-z0-9]+" "filters/dodder-edit\.lua<@blake2b256-cr6qfsyckzh38h648zpvu7at8vtnp6tgl8wmf5tm6e3tdrmzmpmswvjk72 !pandoc-lua_filter@ed25519_sig-[a-z0-9]+" "filters/dodder-common\.lua<@blake2b256-zux3d4kspkhk7xk23q9hjes907uf5r0daquvr2ua3nt2tvc2h7ps3hfnul !pandoc-lua_filter@ed25519_sig-[a-z0-9]+"\]
	EOM
}

# bats test_tags=pandoc
function format_blob_stdin_pandoc_normalizes_markdown { # @test
	wd="$(mktemp -d)"
	cd "$wd" || exit 1

	run_dodder init \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-encryption none \
		-repo_id .default \
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

# Default init (without the opt-in flag) must not create pandoc tool
# types and must leave !md formatter-free. When the opt-in -> opt-out
# flip lands (see docs/plans/2026-03-27-pandoc-internal-formatting-
# design.md), this test will fail and the assertions become the canary
# for the flip itself.
# bats test_tags=pandoc
function init_without_pandoc_flag_produces_minimal_md_type { # @test
	wd="$(mktemp -d)"
	cd "$wd" || exit 1

	run_dodder init \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-encryption none \
		-repo_id .default \
		test-repo-id

	assert_success

	# Genesis output should not list either pandoc tool type
	refute_line --partial 'pandoc-defaults'
	refute_line --partial 'pandoc-lua_filter'

	run_dodder init-workspace -experimental-repo=false

	# show :t should list only !md, with no pandoc neighbors
	run_dodder show :t
	assert_success
	refute_output --partial 'pandoc'

	# !md's type object/metadata should mention neither pandoc tools
	# nor any formatters.text section
	run_dodder show '!md:t'
	assert_success
	refute_output --partial 'pandoc'
	refute_output --partial 'formatters.text'
}

# Covers format_object.FormatFromStdin (previously 0% coverage).
# Parallels format_blob_stdin_pandoc_normalizes_markdown to catch
# regressions if the two stdin paths diverge.
# bats test_tags=pandoc
function format_object_stdin_pandoc_normalizes_markdown { # @test
	wd="$(mktemp -d)"
	cd "$wd" || exit 1

	run_dodder init \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-encryption none \
		-repo_id .default \
		-include-default-pandoc-tools \
		test-repo-id

	assert_success

	run_dodder init-workspace -experimental-repo=false

	run_dodder format-object -stdin text !md <<-'EOM'
		-    item    one
		-  item   two
		-       item three

		this is a paragraph that is way too long and should be wrapped by pandoc because it exceeds the column width of eighty characters which pandoc enforces
	EOM
	assert_success

	assert_output - <<-'EOM'
		- item one

		- item two

		-   item three

		this is a paragraph that is way too long and should be wrapped by pandoc because
		it exceeds the column width of eighty characters which pandoc enforces
	EOM
}

# Exercises MaterializeBlobTree's no-blob-refs early-return branch.
# A !md type with a formatter but no blob references must format
# without crashing and must NOT set DODDER_BLOB_TREE in the formatter
# env. The bash formatter prints the env var explicitly so we can
# assert it's empty and that stdin survived the pipeline.
# bats test_tags=pandoc
function format_blob_with_trivial_formatter_no_blob_refs { # @test
	wd="$(mktemp -d)"
	cd "$wd" || exit 1

	# init WITHOUT pandoc flag -> !md has no formatters and no blob refs
	run_dodder init \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-encryption none \
		-repo_id .default \
		test-repo-id

	assert_success

	run_dodder init-workspace -experimental-repo=false

	# Add a trivial formatter that echoes DODDER_BLOB_TREE before the
	# stdin content. When MaterializeBlobTree hits its no-refs branch,
	# the env var is unset and the prefix line is "BLOB_TREE=".
	run_dodder checkout !md:t
	assert_success

	cat >md.type <<-'EOM'
		inline-akte = true
		[formatters.text]
		shell = [
		  "bash",
		  "-c",
		  "printf 'BLOB_TREE=%s\n' \"${DODDER_BLOB_TREE:-}\"; cat",
		]
	EOM

	run_dodder checkin -delete .t
	assert_success

	run_dodder format-blob -stdin text !md <<-'EOM'
		hello world
	EOM
	assert_success
	assert_output - <<-'EOM'
		BLOB_TREE=
		hello world
	EOM
}

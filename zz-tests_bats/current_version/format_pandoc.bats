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
    .default

  assert_success

  # Genesis should create the pandoc tool types alongside !md
  assert_line --regexp '\[!pandoc-defaults @blake2b256-.+ !toml-type-v2]'
  assert_line --regexp '\[!pandoc-lua_filter @blake2b256-.+ !toml-type-v2]'

  run_dodder init-workspace -experimental-repo=false

  # The !md type metadata should carry exactly six blob references (two
  # lua filters + the edit/html/gdoc/beamer defaults), sorted by
  # content-addressed blob digest. Blob digests are pinned (deterministic
  # from embedded file content); ed25519_sig values vary per init so they
  # are matched via --regexp.
  run_dodder show '!md:t'
  assert_success
  assert_output --regexp - <<-'EOM'
		\[!md @blake2b256-6y95wlu53ac7l3nwqyqmf404e2njyaqn2t0ledt5tuqe3wxgszqspmngzv !toml-type-v2 defaults/dodder-beamer\.yaml<@blake2b256-0krzxu5vg9qhza76ydqlwg3dzlwzlx6phkxun4zyc3cd8aas0m7sz6qn50 !pandoc-defaults@ed25519_sig-[a-z0-9]+ defaults/dodder-edit\.yaml<@blake2b256-amzdh9dljzhu9885kmh654zkyys5mxq62eadx3ej8hwf3ypwd3qq00chz7 !pandoc-defaults@ed25519_sig-[a-z0-9]+ defaults/dodder-gdoc\.yaml<@blake2b256-h78yye7mzdyutm5e5sylue7fqffwqq2dcjzh5rc4gtn6gst5jmysvd7t8e !pandoc-defaults@ed25519_sig-[a-z0-9]+ filters/dodder-edit\.lua<@blake2b256-kgh3lg7gu6rpv8ua68ph30q5400afcus5r8ee23fnl0z3hgrud9sytqvzs !pandoc-lua_filter@ed25519_sig-[a-z0-9]+ filters/dodder-common\.lua<@blake2b256-rn433263q9qx43808kl2ehnqv3mhre8l7wwsk6cp9vvpt06t6uqs04umuq !pandoc-lua_filter@ed25519_sig-[a-z0-9]+ defaults/dodder-html\.yaml<@blake2b256-ufhtc0lw0lefaq9reqv5r53nemhp0ekup9mja5fvpqnf0hr4s6cqs5s2z8 !pandoc-defaults@ed25519_sig-[a-z0-9]+\]
	EOM
}

# The html formatter renders markdown to an HTML fragment (no <html>
# scaffolding) via the blob-backed dodder-html defaults.
# bats test_tags=pandoc
function format_blob_stdin_pandoc_renders_html_fragment { # @test
  command -v pandoc >/dev/null || skip "pandoc not available"

  wd="$(mktemp -d)"
  cd "$wd" || exit 1

  run_dodder init \
    -yin <(cat_yin) \
    -yang <(cat_yang) \
    -encryption none \
    .default

  assert_success

  run_dodder init-workspace -experimental-repo=false

  run_dodder format-blob -stdin html !md <<-'EOM'
		# Hello

		some *emphasis* and a [link](https://example.com)
	EOM
  assert_success

  assert_output - <<-'EOM'
		<h1 id="hello">Hello</h1>
		<p>some <em>emphasis</em> and a <a href="https://example.com">link</a></p>
	EOM
}

# The html-gdoc formatter renders a standalone, self-contained HTML
# document (for pasting into Google Docs) via the blob-backed dodder-gdoc
# defaults. The full document embeds pandoc's default template, so only
# the load-bearing shape is asserted: doctype, <html> envelope, the
# warning-silencing <title>, and the rendered body content.
# bats test_tags=pandoc
function format_blob_stdin_pandoc_renders_standalone_gdoc_html { # @test
  command -v pandoc >/dev/null || skip "pandoc not available"

  wd="$(mktemp -d)"
  cd "$wd" || exit 1

  run_dodder init \
    -yin <(cat_yin) \
    -yang <(cat_yang) \
    -encryption none \
    .default

  assert_success

  run_dodder init-workspace -experimental-repo=false

  run_dodder format-blob -stdin html-gdoc !md <<-'EOM'
		# Hello

		a standalone document
	EOM
  assert_success

  assert_output --regexp '^<!DOCTYPE html>'
  assert_output --regexp '<html [^>]*>'
  assert_output --regexp '<title>dodder</title>'
  assert_output --regexp '<h1 id="hello">Hello</h1>'
  assert_output --regexp '<p>a standalone document</p>'
  assert_output --regexp '</html>$'
}

# bats test_tags=pandoc
function format_blob_stdin_pandoc_normalizes_markdown { # @test
  wd="$(mktemp -d)"
  cd "$wd" || exit 1

  run_dodder init \
    -yin <(cat_yin) \
    -yang <(cat_yang) \
    -encryption none \
    .default

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

# Pandoc tools are default-on since #208. Passing -exclude-default-pandoc-tools
# opts back out: genesis must NOT create the pandoc tool types and must leave
# !md formatter-free. (The default-on case — pandoc tools present — is covered
# by init_with_pandoc_tools_creates_type_objects above.)
# bats test_tags=pandoc
function init_with_exclude_flag_produces_minimal_md_type { # @test
  wd="$(mktemp -d)"
  cd "$wd" || exit 1

  run_dodder init \
    -yin <(cat_yin) \
    -yang <(cat_yang) \
    -encryption none \
    -exclude-default-pandoc-tools \
    .default

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
    .default

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

  # init with -exclude-default-pandoc-tools -> !md has no formatters and no
  # blob refs
  run_dodder init \
    -yin <(cat_yin) \
    -yang <(cat_yang) \
    -encryption none \
    -exclude-default-pandoc-tools \
    .default

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

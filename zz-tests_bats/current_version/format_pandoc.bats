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

  # The !md type metadata should carry exactly nine blob references (three
  # lua filters + the edit/render/html/html-partial/gdoc/beamer defaults),
  # sorted by content-addressed blob digest. Blob digests are deterministic
  # from embedded file content; ed25519_sig values vary per init and are
  # masked by the golden normalizer.
  run_dodder show '!md:t'
  assert_success
  assert_golden md_type_blob_references
}

# The builtin !md type blob carries the full formatter matrix (the edit
# pipeline text/html/html-gdoc/pdf-beamer plus the render pipeline
# text-render/html-partial) and the uti-groups bundling them by output
# medium. Every uti-group value must name a formatter above it.
# bats test_tags=pandoc
function md_type_blob_lists_formatters_and_uti_groups { # @test
  wd="$(mktemp -d)"
  cd "$wd" || exit 1

  run_dodder init \
    -yin <(cat_yin) \
    -yang <(cat_yang) \
    -encryption none \
    .default

  assert_success

  run_dodder init-workspace -experimental-repo=false

  run_dodder show -format blob '!md:t'
  assert_success
  assert_golden md_type_blob_formatters_and_uti_groups
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

# The text-render formatter renders markdown for OUTPUT via the blob-backed
# dodder-render defaults (render pipeline): reference-style links, no
# 80-column re-wrap. For plain markdown (no typed code blocks) the render
# filter is a no-op, so no dodder binary round-trip happens.
# bats test_tags=pandoc
function format_blob_stdin_pandoc_text_render_renders_markdown { # @test
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

  run_dodder format-blob -stdin text-render !md <<-'EOM'
		# Hello

		some *emphasis* and a [link](https://example.com)
	EOM
  assert_success

  assert_output - <<-'EOM'
		# Hello

		some *emphasis* and a [link]

		  [link]: https://example.com
	EOM
}

# The html-partial formatter renders the same HTML fragment shape as html,
# but via the RENDER pipeline (dodder-render.lua): for plain markdown the
# two are identical; they diverge only on typed code blocks (render
# replaces them with rendered images instead of inlining text).
# bats test_tags=pandoc
function format_blob_stdin_pandoc_html_partial_renders_fragment { # @test
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

  run_dodder format-blob -stdin html-partial !md <<-'EOM'
		# Hello

		some *emphasis* and a [link](https://example.com)
	EOM
  assert_success

  assert_output - <<-'EOM'
		<h1 id="hello">Hello</h1>
		<p>some <em>emphasis</em> and a <a href="https://example.com">link</a></p>
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

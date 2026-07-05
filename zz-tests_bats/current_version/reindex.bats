#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

  # for shellcheck SC2154
  export output

  copy_from_version "$DIR"

  run_dodder_init_workspace
}

teardown() {
  chflags_nouchg
}

function reindex_simple { # @test
  run_dodder reindex
  assert_success
  run_dodder show +t,e,z
  assert_success
  assert_golden_unsorted reindex_simple

  run_dodder show -format tags-path :e,z,t
  assert_success
  assert_output_unsorted - <<-EOM
		!md [Paths: [], All: []]
		!pandoc-defaults [Paths: [], All: []]
		!pandoc-lua_filter [Paths: [], All: []]
		one/dos [Paths: [TypeDirect:[tag-3] TypeDirect:[tag-4]], All: [tag-3:[TypeDirect:[tag-3]] tag-4:[TypeDirect:[tag-4]]]]
		one/uno [Paths: [TypeDirect:[tag-3] TypeDirect:[tag-4]], All: [tag-3:[TypeDirect:[tag-3]] tag-4:[TypeDirect:[tag-4]]]]
	EOM
}

function reindex_clean_omits_error_headers { # @test
  # A clean reindex (no unidentified errors, no objects with errors)
  # must not print the section headers (#261).
  run_dodder reindex
  assert_success
  refute_output --partial "unidentified errors:"
  refute_output --partial "objects with errors:"
}

function reindex_simple_twice { # @test
  run_dodder reindex
  assert_success
  run_dodder show +e,t,z
  assert_success
  assert_golden_unsorted reindex_simple_twice

  run_dodder reindex
  assert_success
  run_dodder show +e,t,z
  assert_success
  assert_golden_unsorted reindex_simple_twice
}

function reindex_after_changes { # @test
  run_dodder show !md:t
  assert_success
  assert_golden reindex_after_changes_type

  cat >md.type <<-EOM
		inline-akte = false
		vim-syntax-type = "test"
	EOM

  run_dodder checkin .t
  assert_success
  assert_output - <<-EOM
		[!md @blake2b256-473260as3d3pd4uramcc60877srvpkxs4krlap45dkl3mfvq2npq2duvvq !toml-type-v2]
	EOM

  function verify() {
    run_dodder show -format blob !md+t
    assert_success
    # The fixture's genesis !md blob is the pandoc-flavored one (#208): it
    # carries the text/html/html-gdoc/pdf-beamer formatter blocks. The
    # checked-in replacement follows.
    assert_output - <<-'EOM'
			file-extension = "md"
			vim-syntax-type = "markdown"

			[formatters.html]
			description = "Render markdown to an HTML fragment with pandoc"
			script = """
			pandoc --data-dir="$DODDER_BLOB_TREE" --defaults=dodder-html"""
			file-extension = "html"
			[formatters.html-gdoc]
			description = "Render markdown to standalone HTML for pasting into Google Docs"
			script = """
			pandoc --data-dir="$DODDER_BLOB_TREE" --defaults=dodder-gdoc"""
			file-extension = "html"
			[formatters.pdf-beamer]
			description = "Render markdown to a beamer slide PDF (requires a host LaTeX engine)"
			script = """
			tmp="$(mktemp -d)" || exit 1
			trap 'rm -rf "$tmp"' EXIT
			mkfifo "$tmp/out.pdf" || exit 1
			cat "$tmp/out.pdf" &
			if ! pandoc --data-dir="$DODDER_BLOB_TREE" --defaults=dodder-beamer --output="$tmp/out.pdf"; then
			  : >"$tmp/out.pdf"
			  wait
			  exit 1
			fi
			wait"""
			file-extension = "pdf"
			[formatters.text]
			description = "Normalize markdown with pandoc"
			script = """
			pandoc --data-dir="$DODDER_BLOB_TREE" --defaults=dodder-edit"""
			file-extension = "md"
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
  run_dodder show +e,t,z
  assert_success
  assert_golden_unsorted reindex_after_changes_all

  verify
}

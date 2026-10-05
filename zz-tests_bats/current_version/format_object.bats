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

function format_simple { # @test
  run_dodder_init_workspace
  run_dodder checkout !md:t
  assert_success

  cat >md.type <<-EOM
		[formatters.text]
		shell = [
		  "cat",
		]
	EOM

  # run cat .dodder/Objekten/Akten/*/*
  # assert_output ''

  run_dodder checkin -delete .t
  assert_success
  assert_output - <<-EOM
		[!md @blake2b256-tzxk3zzhvmggd66r0x6jfu662zcfe9qyalnjkr5yr4rqen2dg6mslsq8pl !toml-type-v3]
		          deleted [md.type]
	EOM

  run_dodder format-object -mode both one/uno text
  assert_success
  assert_output --regexp - <<EOM
---
# wow the first
- tag-3
- tag-4
! md@.*
---
EOM

  run_dodder checkout one/uno
  assert_success
  cat >one/uno.zettel <<-EOM
		---
		# wow the second
		- tag-3
		- tag-4
		! md
		---

		last time but new
	EOM

  run_dodder format-object -mode both one/uno.zettel text
  assert_success
  assert_output --regexp - <<EOM
---
# wow the second
- tag-3
- tag-4
! md@.*
---
EOM
}

function show_simple_one_zettel_binary { # @test
  run_dodder init-workspace -experimental-repo=false
  assert_success

  echo "binary file" >file.bin
  run_dodder add -delete file.bin
  assert_success
  assert_output_unsorted - <<-EOM
		          deleted [file.bin]
		[!bin !toml-type-v3]
		[two/uno @blake2b256-w9l3z9c2w8lhr42fwekmhrxeqtmzw40s9p46vt88ydgwux4rxxuqnfqsmk !bin "file"]
	EOM

  run_dodder checkout !bin:t
  assert_success
  assert_output_unsorted - <<-EOM
		      checked out [bin.type !toml-type-v3]
	EOM

  cat >bin.type <<-EOM
		---
		! toml-type-v3
		---

		binary = true
	EOM

  run_dodder checkin -delete bin.type
  assert_success
  assert_output_unsorted - <<-EOM
		          deleted [bin.type]
		[!bin @blake2b256-zhvux7vmpch9f44kvnua7n69f8jzgk5s7p9k2s3kuvkrcpjh07lse493jl !toml-type-v3]
	EOM

  run_dodder format-object -mode both two/uno
  assert_success
  assert_output --regexp - <<-EOM
		---
		# file
		@ blake2b256-w9l3z9c2w8lhr42fwekmhrxeqtmzw40s9p46vt88ydgwux4rxxuqnfqsmk
		! bin@.*
		---
	EOM
}

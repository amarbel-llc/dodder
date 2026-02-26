#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/common.bash"

  # for shellcheck SC2154
  export output

  copy_from_version "$DIR"

  # TODO prevent checkouts if workspace is not initialized
  run_dodder_init_workspace

  cat >txt.type <<-EOM
		---
		! toml-type-v1
		---

		binary = false
	EOM

  cat >bin.type <<-EOM
		---
		! toml-type-v1
		---

		binary = true
	EOM

  run_dodder checkin -delete bin.type txt.type
  assert_success
  assert_output_unsorted - <<-EOM
		          deleted [bin.type]
		          deleted [txt.type]
		[!bin @blake2b256-zhvux7vmpch9f44kvnua7n69f8jzgk5s7p9k2s3kuvkrcpjh07lse493jl !toml-type-v1]
		[!txt @blake2b256-qxzg22c3axe9m42tpwqd4usnfag4elp20q7zvnkgmyea4f4rwcwsurfp5e !toml-type-v1]
	EOM
}

teardown() {
  chflags_and_rm
}

function checkout_simple_all { # @test
  run_dodder checkout :z,t,e
  assert_success
  assert_output_unsorted - <<-EOM
		      checked out [txt.type @blake2b256-qxzg22c3axe9m42tpwqd4usnfag4elp20q7zvnkgmyea4f4rwcwsurfp5e !toml-type-v1]
		      checked out [bin.type @blake2b256-zhvux7vmpch9f44kvnua7n69f8jzgk5s7p9k2s3kuvkrcpjh07lse493jl !toml-type-v1]
		      checked out [md.type @$(get_type_blob_sha) !toml-type-v1]
		      checked out [one/dos.zettel @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		      checked out [one/uno.zettel @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM
}

function checkout_simple_zettel { # @test
  run_dodder checkout :
  assert_success
  assert_output_unsorted - <<-EOM
		      checked out [one/dos.zettel @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		      checked out [one/uno.zettel @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM
}

function checkout_non_binary_simple_zettel { # @test
  echo "text file" >file.txt
  run_dodder add -delete file.txt
  assert_success
  assert_output_unsorted - <<-EOM
		          deleted [file.txt]
		[two/uno @blake2b256-eu5uyveldt6hg5ddd80k0qjsjvkt5d5u24gg36084ehr7yppvkws7cac7g !txt "file"]
	EOM

  run_dodder show -format text !txt:z
  assert_success
  assert_output --regexp - <<-EOM
		---
		# file
		! txt@.*
		---

		text file
	EOM
}

function checkout_binary_simple_zettel { # @test
  echo "binary file" >file.bin
  run_dodder add -delete file.bin
  assert_success
  assert_output_unsorted - <<-EOM
		          deleted [file.bin]
		[two/uno @blake2b256-w9l3z9c2w8lhr42fwekmhrxeqtmzw40s9p46vt88ydgwux4rxxuqnfqsmk !bin "file"]
	EOM

  run_dodder checkout !bin:z
  assert_success
  assert_output_unsorted - <<-EOM
		      checked out [two/uno.zettel @blake2b256-w9l3z9c2w8lhr42fwekmhrxeqtmzw40s9p46vt88ydgwux4rxxuqnfqsmk !bin "file"]
	EOM

  run cat two/uno.zettel
  assert_success
  assert_output --regexp - <<-EOM
		---
		# file
		@ blake2b256-w9l3z9c2w8lhr42fwekmhrxeqtmzw40s9p46vt88ydgwux4rxxuqnfqsmk
		! bin@.*
		---
	EOM
}

function checkout_simple_zettel_blob_only { # @test
  run_dodder clean .
  assert_success
  # TODO fail checkouts if working directly has incompatible checkout
  run_dodder checkout -mode blob :z
  assert_success
  assert_output_unsorted - <<-EOM
		                   one/dos.md]
		                   one/uno.md]
		      checked out [one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4
		      checked out [one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4
	EOM
}

function checkout_zettel_several { # @test
  run_dodder checkout one/uno one/dos
  assert_success
  assert_output_unsorted - <<-EOM
		      checked out [one/dos.zettel @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		      checked out [one/uno.zettel @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM
}

function checkout_simple_type { # @test
  run_dodder checkout :t
  assert_success
  assert_output_unsorted - <<-EOM
		      checked out [bin.type @blake2b256-zhvux7vmpch9f44kvnua7n69f8jzgk5s7p9k2s3kuvkrcpjh07lse493jl !toml-type-v1]
		      checked out [md.type @blake2b256-3kj7xgch6rjkq64aa36pnjtn9mdnl89k8pdhtlh33cjfpzy8ek4qnufx0m !toml-type-v1]
		      checked out [txt.type @blake2b256-qxzg22c3axe9m42tpwqd4usnfag4elp20q7zvnkgmyea4f4rwcwsurfp5e !toml-type-v1]
	EOM
}

function checkout_zettel_blob_then_object { # @test
  run_dodder checkout -mode blob one/uno
  assert_success
  assert_output - <<-EOM
		      checked out [one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4
		                   one/uno.md]
	EOM

  run_dodder checkout one/uno
  assert_success
  assert_output - <<-EOM
		      checked out [one/uno.zettel @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM

  run ls one/
  assert_output_unsorted - <<-EOM
		uno.zettel
	EOM

  run_dodder checkout -force one/uno
  assert_success
  assert_output - <<-EOM
		      checked out [one/uno.zettel @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM

  run ls one/
  assert_output - <<-EOM
		uno.zettel
	EOM
}

function mode_both { # @test
  run_dodder new -edit=false - <<-EOM
		---
		! bin
		---

		not really pdf content but that's ok
	EOM
  assert_success
  assert_output - <<-EOM
		[two/uno @blake2b256-jyxyyxxrgdsgt5nwezujm3e037rh3ce4f85wllrzg0y3adg7f5pqg5sw95 !bin]
	EOM

  run_dodder checkout -mode both two/uno
  assert_success
  assert_output - <<-EOM
		      checked out [two/uno.zettel @blake2b256-jyxyyxxrgdsgt5nwezujm3e037rh3ce4f85wllrzg0y3adg7f5pqg5sw95 !bin
		                   two/uno.bin]
	EOM

  run ls two/
  assert_output_unsorted - <<-EOM
		uno.bin
		uno.zettel
	EOM
}

# bats test_tags=user_story:builtin_types
function checkout_builtin_type { # @test
  run_dodder checkout !toml-type-v1:t
  assert_success
  assert_output_unsorted - <<-EOM
		      checked out [md.type @blake2b256-3kj7xgch6rjkq64aa36pnjtn9mdnl89k8pdhtlh33cjfpzy8ek4qnufx0m !toml-type-v1]
		      checked out [txt.type @blake2b256-qxzg22c3axe9m42tpwqd4usnfag4elp20q7zvnkgmyea4f4rwcwsurfp5e !toml-type-v1]
		      checked out [bin.type @blake2b256-zhvux7vmpch9f44kvnua7n69f8jzgk5s7p9k2s3kuvkrcpjh07lse493jl !toml-type-v1]
	EOM
}

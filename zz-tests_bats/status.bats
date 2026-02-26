#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/common.bash"

  # for shellcheck SC2154
  export output

  copy_from_version "$DIR"

  run_dodder_init_workspace
}

teardown() {
  chflags_and_rm
}

function checkout_everything() {
  run_dodder checkout :z,t,e
  assert_success
  assert_output_unsorted - <<-EOM
		      checked out [md.type @$(get_type_blob_sha) !toml-type-v1]
		      checked out [one/dos.zettel @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		      checked out [one/uno.zettel @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM
}

function dirty_new_zettel() {
  run_dodder new -edit=false - <<-EOM
		---
		# the new zettel
		- etikett-one
		! txt
		---

		with a different typ
	EOM

  assert_success
  assert_output --partial - <<-EOM
		[!txt !toml-type-v1]
		[two/uno @blake2b256-x4dstl5rrxp60932zj0sgmaku39ylula4fg3scgcgyj4yyneyy3qdtnzlm !txt "the new zettel" etikett-one]
	EOM
}

function dirty_existing_akte() {
  cat >one/uno.zettel <<-EOM
		---
		# wildly different
		- etikett-one
		@ one/uno.md
		! md
		---
	EOM

  cat >one/uno.md <<-EOM
		newest body but even newer
	EOM
}

function dirty_one_uno() {
  cat >one/uno.zettel <<-EOM
		---
		# wildly different
		- etikett-one
		! md
		---

		newest body
	EOM
}

function dirty_one_dos() {
  cat >one/dos.zettel <<-EOM
		---
		# dos wildly different
		- etikett-two
		! md
		---

		dos newest body
	EOM
}

function dirty_md_typ() {
  cat >md.type <<-EOM
		inline-akte = false
		vim-syntax-type = "test"
	EOM
}

function dirty_da_new_typ() {
  cat >da-new.type <<-EOM
		inline-akte = true
		vim-syntax-type = "da-new"
	EOM
}

function dirty_zz_archive_tag() {
  cat >zz-archive.tag <<-EOM
		hide = true
	EOM
}

function status_simple_one_zettel { # @test
  checkout_everything
  run_dodder status one/uno.zettel
  assert_success
  assert_output - <<-EOM
		             same [one/uno.zettel @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM

  dirty_one_uno

  run_dodder status one/uno.zettel
  assert_success
  assert_output - <<-EOM
		          changed [one/uno.zettel @blake2b256-k87yyah5da3c8h9j4ugf44edeurrqztn7zddh7ksc88pfg4zzx0smqmuf9 !md "wildly different" etikett-one]
	EOM
}

function status_simple_one_zettel_blob_separate { # @test
  checkout_everything
  run_dodder status one/uno.zettel
  assert_success
  assert_output - <<-EOM
		             same [one/uno.zettel @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM

  rm one/uno.zettel

  cat >one/uno.md <<-EOM
		newest body but even newerests
	EOM

  run_dodder status one/uno.zettel
  assert_success
  assert_output - <<-EOM
		          changed [one/uno @blake2b256-dy8ywz7cr2pr4tgf8lfjsyfhmvxpezul5p7mk7yl2x4khjr7a4ns4cnst4 !md "wow the first" tag-3 tag-4
		                   one/uno.md]
	EOM
}

function status_simple_one_zettel_blob_only { # @test
  checkout_everything
  run_dodder clean one/uno.zettel
  assert_success
  # assert_output - <<-EOM
  # 	          deleted [one/uno.zettel]
  # EOM

  run_dodder checkout -mode blob one/uno
  # assert_output - <<-EOM
  # 	      checked out [one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4
  # 	                   one/uno.md]
  # EOM

  run_dodder status one/uno.zettel
  assert_success
  # assert_output - <<-EOM
  # 	             same [one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4
  # 	                   one/uno.md]
  # EOM

  dirty_existing_akte

  run ls one
  assert_success
  assert_output - <<-EOM
		dos.zettel
		uno.md
		uno.zettel
	EOM

  run_dodder status one/uno.zettel
  assert_success
  assert_output - <<-EOM
		          changed [one/uno.zettel @blake2b256-kdw9q3458v3njrejvhc7tjfsddxnzpmg5wt8mdwq7psss20whkesyxdzx7 !md "wildly different" etikett-one
		                   one/uno.md]
	EOM
}

function status_zettel_blob_checkout { # @test
  checkout_everything
  run_dodder clean .
  assert_success

  dirty_new_zettel

  run_dodder checkout -mode blob two/uno
  assert_success
  assert_output - <<-EOM
		      checked out [two/uno @blake2b256-x4dstl5rrxp60932zj0sgmaku39ylula4fg3scgcgyj4yyneyy3qdtnzlm !txt "the new zettel" etikett-one
		                   two/uno.txt]
	EOM

  run_dodder status .z
  assert_success
  assert_output - <<-EOM
		             same [two/uno @blake2b256-x4dstl5rrxp60932zj0sgmaku39ylula4fg3scgcgyj4yyneyy3qdtnzlm !txt "the new zettel" etikett-one
		                   two/uno.txt]
	EOM
}

function status_zettel_hidden { # @test
  checkout_everything
  run_dodder dormant-add tag-3
  assert_success

  run_dodder show :z
  assert_success
  assert_output ''

  run_dodder show :?z
  assert_success
  assert_output_unsorted - <<-EOM
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM

  run_dodder status .z
  assert_success
  assert_output_unsorted - <<-EOM
		             same [one/uno.zettel @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
		             same [one/dos.zettel @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
	EOM

  run_dodder status !md.z
  assert_success
  assert_output_unsorted - <<-EOM
		             same [one/uno.zettel @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
		             same [one/dos.zettel @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
	EOM
}

function status_zettelen_typ { # @test
  checkout_everything
  run_dodder status !md.z
  assert_success
  assert_output_unsorted - <<-EOM
		             same [one/dos.zettel @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		             same [one/uno.zettel @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM

  dirty_one_uno
  dirty_one_dos

  run_dodder status !md.z
  assert_success
  assert_output_unsorted - <<-EOM
		          changed [one/dos.zettel @blake2b256-wn30f7j6g62r7lgz0jhmnapnkem09c7lkkv65k005wv3fnj44m7q6auex2 !md "dos wildly different" etikett-two]
		          changed [one/uno.zettel @blake2b256-k87yyah5da3c8h9j4ugf44edeurrqztn7zddh7ksc88pfg4zzx0smqmuf9 !md "wildly different" etikett-one]
	EOM
}

function status_complex_zettel_tag_negation { # @test
  checkout_everything
  run_dodder status ^-etikett-two.z
  assert_success
  assert_output_unsorted - <<-EOM
		             same [one/dos.zettel @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		             same [one/uno.zettel @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM

  dirty_one_uno

  run_dodder status ^-etikett-two.z
  assert_success
  assert_output_unsorted - <<-EOM
		             same [one/dos.zettel @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		          changed [one/uno.zettel @blake2b256-k87yyah5da3c8h9j4ugf44edeurrqztn7zddh7ksc88pfg4zzx0smqmuf9 !md "wildly different" etikett-one]
	EOM
}

function status_simple_all { # @test
  checkout_everything
  run_dodder status
  assert_success
  assert_output_unsorted - <<-EOM
		             same [md.type @$(get_type_blob_sha) !toml-type-v1]
		             same [one/dos.zettel @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		             same [one/uno.zettel @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM

  dirty_one_uno
  dirty_one_dos
  dirty_md_typ
  dirty_zz_archive_tag
  dirty_da_new_typ

  run_dodder status .
  assert_success
  assert_output_unsorted - <<-EOM
		          changed [md.type @blake2b256-473260as3d3pd4uramcc60877srvpkxs4krlap45dkl3mfvq2npq2duvvq !toml-type-v1]
		          changed [one/dos.zettel @blake2b256-wn30f7j6g62r7lgz0jhmnapnkem09c7lkkv65k005wv3fnj44m7q6auex2 !md "dos wildly different" etikett-two]
		          changed [one/uno.zettel @blake2b256-k87yyah5da3c8h9j4ugf44edeurrqztn7zddh7ksc88pfg4zzx0smqmuf9 !md "wildly different" etikett-one]
		        untracked [da-new.type @blake2b256-9rzlpgryfegathtl4ss3s80cwskx7e5w77usfjxgxrrg4ns80epqnzxjvs !toml-type-v1]
		        untracked [zz-archive.tag @blake2b256-4nnaw9wx7vwsdlx777qf48drgxeatj762ykhlwhe6pykmmutglvsz2szgt]
	EOM
}

function status_simple_typ { # @test
  checkout_everything
  run_dodder status .t
  assert_success
  assert_output_unsorted - <<-EOM
		             same [md.type @$(get_type_blob_sha) !toml-type-v1]
	EOM

  dirty_md_typ
  dirty_da_new_typ

  run_dodder status .t
  assert_success
  assert_output_unsorted - <<-EOM
		          changed [md.type @blake2b256-473260as3d3pd4uramcc60877srvpkxs4krlap45dkl3mfvq2npq2duvvq !toml-type-v1]
		        untracked [da-new.type @blake2b256-9rzlpgryfegathtl4ss3s80cwskx7e5w77usfjxgxrrg4ns80epqnzxjvs !toml-type-v1]
	EOM
}

function status_simple_tag { # @test
  checkout_everything
  run_dodder status .e
  assert_success
  assert_output_unsorted - <<-EOM
	EOM

  dirty_zz_archive_tag

  run_dodder status .e
  assert_success
  assert_output_unsorted - <<-EOM
		        untracked [zz-archive.tag @blake2b256-4nnaw9wx7vwsdlx777qf48drgxeatj762ykhlwhe6pykmmutglvsz2szgt]
	EOM
}

function status_conflict { # @test
  checkout_everything
  run_dodder checkout one/dos
  assert_success
  assert_output_unsorted - <<-EOM
		      checked out [one/dos.zettel @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
	EOM

  cat - >one/dos.zettel <<-EOM
		---
		# wow ok again
		- get_this_shit_merged
		- tag-3
		- tag-4
		! txt
		---

		not another one, conflict time
	EOM

  run_dodder organize -mode commit-directly one/dos <<-EOM
		---
		! txt2
		---

		# new-etikett-for-all
		- [one/dos  tag-3 tag-4] wow ok again
	EOM
  assert_success
  assert_output - <<-EOM
		[!txt2 !toml-type-v1]
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !txt2 "wow ok again" new-etikett-for-all tag-3 tag-4]
	EOM

  run_dodder show -format log new-etikett-for-all:z,e,t
  assert_success
  assert_output - <<-EOM
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !txt2 "wow ok again" new-etikett-for-all tag-3 tag-4]
	EOM

  run_dodder status one/dos.zettel
  assert_success
  assert_output - <<-EOM
		       conflicted [one/dos.zettel]
	EOM
}

# bats test_tags=user_story:fs_blobs
function status_added_untracked_only() { # @test
  cat >test.md <<-EOM
		newest body
	EOM

  run_dodder status .
  assert_success
  assert_output_unsorted - <<-EOM
		        untracked [test.md @blake2b256-k87yyah5da3c8h9j4ugf44edeurrqztn7zddh7ksc88pfg4zzx0smqmuf9]
	EOM
}

# bats test_tags=user_story:fs_blobs
function status_added_untracked() { # @test
  checkout_everything
  cat >test.md <<-EOM
		newest body
	EOM

  run_dodder status .
  assert_success
  assert_output_unsorted - <<-EOM
		        untracked [test.md @blake2b256-k87yyah5da3c8h9j4ugf44edeurrqztn7zddh7ksc88pfg4zzx0smqmuf9]
		             same [one/uno.zettel @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
		             same [md.type @blake2b256-3kj7xgch6rjkq64aa36pnjtn9mdnl89k8pdhtlh33cjfpzy8ek4qnufx0m !toml-type-v1]
		             same [one/dos.zettel @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
	EOM
}

# bats test_tags=user_story:fs_blobs, user_story:recognized_blobs
function status_dot_untracked_recognized_blob_only() { # @test
  run_dodder show -format blob one/uno
  echo "$output" >test.md

  run_dodder status .
  assert_success
  assert_output - <<-EOM
		       recognized [one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4
		                   test.md]
	EOM
}

# bats test_tags=user_story:fs_blobs, user_story:recognized_blobs
function status_explicit_untracked_recognized_blob_only() { # @test
  run_dodder show -format blob one/uno
  echo "$output" >test.md

  run_dodder status test.md
  assert_success
  assert_output - <<-EOM
		       recognized [one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4
		                   test.md]
	EOM
}

# bats test_tags=user_story:fs_blobs, user_story:recognized_blobs
function status_dot_untracked_recognized_blob() { # @test
  checkout_everything
  run_dodder show -format blob one/uno
  echo "$output" >test.md

  run_dodder status .
  assert_success
  assert_output_unsorted - <<-EOM
		       recognized [one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4
		                   test.md]
		             same [md.type @blake2b256-3kj7xgch6rjkq64aa36pnjtn9mdnl89k8pdhtlh33cjfpzy8ek4qnufx0m !toml-type-v1]
		             same [one/dos.zettel @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		             same [one/uno.zettel @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM
}

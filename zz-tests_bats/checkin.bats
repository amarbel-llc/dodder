#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/common.bash"

  # for shellcheck SC2154
  export output

  copy_from_version "$DIR"

  run_dodder_init_workspace

  run_dodder checkout :z,t,e
  assert_success
  assert_output_unsorted - <<-EOM
		      checked out [md.type @$(get_type_blob_sha) !toml-type-v1]
		      checked out [one/dos.zettel @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		      checked out [one/uno.zettel @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM

  run ls
  assert_success
  assert_output_unsorted - <<-EOM
		md.type
		one
	EOM

  cat >one/uno.zettel <<-EOM
		---
		# wildly different
		- etikett-one
		! md
		---

		newest body
	EOM

  cat >one/dos.zettel <<-EOM
		---
		# dos wildly different
		- etikett-two
		! md
		---

		dos newest body
	EOM

  cat >md.type <<-EOM
		binary = true
		vim-syntax-type = "test"
	EOM

  cat >zz-archive.tag <<-EOM
		hide = true
	EOM

  export BATS_TEST_BODY=true
}

teardown() {
  chflags_and_rm
}

function dirty_one_virtual() {
  cat >one/uno.zettel <<-EOM
		---
		# wildly different
		- etikett-one
		- %virtual
		! md
		---

		newest body
	EOM
}

function checkin_simple_one_zettel { # @test
  run_dodder checkin one/uno.zettel
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-k87yyah5da3c8h9j4ugf44edeurrqztn7zddh7ksc88pfg4zzx0smqmuf9 !md "wildly different" etikett-one]
	EOM
}

function checkin_two_zettel_hidden { # @test
  run_dodder dormant-add etikett-one tag-3
  assert_success

  run_dodder checkin .z
  assert_success
  assert_output_unsorted - <<-EOM
		[one/uno @blake2b256-k87yyah5da3c8h9j4ugf44edeurrqztn7zddh7ksc88pfg4zzx0smqmuf9 !md "wildly different" etikett-one]
		[one/dos @blake2b256-wn30f7j6g62r7lgz0jhmnapnkem09c7lkkv65k005wv3fnj44m7q6auex2 !md "dos wildly different" etikett-two]
	EOM
}

function checkin_simple_one_zettel_virtual_tag { # @test
  dirty_one_virtual
  run_dodder checkin one/uno.zettel
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-k87yyah5da3c8h9j4ugf44edeurrqztn7zddh7ksc88pfg4zzx0smqmuf9 !md "wildly different" %virtual etikett-one]
	EOM

  run_dodder show one/uno
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-k87yyah5da3c8h9j4ugf44edeurrqztn7zddh7ksc88pfg4zzx0smqmuf9 !md "wildly different" etikett-one]
	EOM
}

function checkin_complex_zettel_tag_negation { # @test
  run_dodder checkin ^etikett-two.z
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-k87yyah5da3c8h9j4ugf44edeurrqztn7zddh7ksc88pfg4zzx0smqmuf9 !md "wildly different" etikett-one]
	EOM
}

function checkin_simple_all { # @test
  run_dodder checkin .
  assert_success
  assert_output_unsorted - <<-EOM
		[!md @blake2b256-76m5lj0dp3je79ft9z2mdwpcrzrf9sddj04tvewpuk6gyqmll27sz46w72 !toml-type-v1]
		[one/dos @blake2b256-wn30f7j6g62r7lgz0jhmnapnkem09c7lkkv65k005wv3fnj44m7q6auex2 !md "dos wildly different" etikett-two]
		[one/uno @blake2b256-k87yyah5da3c8h9j4ugf44edeurrqztn7zddh7ksc88pfg4zzx0smqmuf9 !md "wildly different" etikett-one]
		[zz-archive @blake2b256-4nnaw9wx7vwsdlx777qf48drgxeatj762ykhlwhe6pykmmutglvsz2szgt]
	EOM

  run_dodder show -format log :?z,e,t
  assert_success
  assert_output_unsorted - <<-EOM
		[!md @blake2b256-76m5lj0dp3je79ft9z2mdwpcrzrf9sddj04tvewpuk6gyqmll27sz46w72 !toml-type-v1]
		[one/dos @blake2b256-wn30f7j6g62r7lgz0jhmnapnkem09c7lkkv65k005wv3fnj44m7q6auex2 !md "dos wildly different" etikett-two]
		[one/uno @blake2b256-k87yyah5da3c8h9j4ugf44edeurrqztn7zddh7ksc88pfg4zzx0smqmuf9 !md "wildly different" etikett-one]
		[zz-archive @blake2b256-4nnaw9wx7vwsdlx777qf48drgxeatj762ykhlwhe6pykmmutglvsz2szgt]
	EOM
}

function checkin_simple_all_dry_run { # @test
  run_dodder checkin -dry-run .
  assert_success
  assert_output_unsorted - <<-EOM
		[!md @blake2b256-76m5lj0dp3je79ft9z2mdwpcrzrf9sddj04tvewpuk6gyqmll27sz46w72 !toml-type-v1]
		[one/dos @blake2b256-wn30f7j6g62r7lgz0jhmnapnkem09c7lkkv65k005wv3fnj44m7q6auex2 !md "dos wildly different" etikett-two]
		[one/uno @blake2b256-k87yyah5da3c8h9j4ugf44edeurrqztn7zddh7ksc88pfg4zzx0smqmuf9 !md "wildly different" etikett-one]
		[zz-archive @blake2b256-4nnaw9wx7vwsdlx777qf48drgxeatj762ykhlwhe6pykmmutglvsz2szgt]
	EOM

  run_dodder show -format log :z,e,t
  assert_success
  assert_output_unsorted - <<-EOM
		[!md @$(get_type_blob_sha) !toml-type-v1]
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM
}

function checkin_simple_typ { # @test
  run_dodder checkin .t
  assert_success
  assert_output - <<-EOM
		[!md @blake2b256-76m5lj0dp3je79ft9z2mdwpcrzrf9sddj04tvewpuk6gyqmll27sz46w72 !toml-type-v1]
	EOM

  run_dodder update :z
  assert_success
  assert_output_unsorted - <<-'EOM'
[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
	EOM

  run_dodder last -format box-archive
  assert_success
  assert_output_unsorted --regexp - <<-'EOM'
\[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd dodder-repo-public_key-v1@.* dodder-object-mother-sig-v2@.* dodder-object-sig-v2@.* !md@.* "wow the first" tag-3 tag-4]
\[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd dodder-repo-public_key-v1@.* dodder-object-mother-sig-v2@.* dodder-object-sig-v2@.* !md@.* "wow ok again" tag-3 tag-4]
	EOM

  run_dodder show -format blob !md:t
  assert_success
  assert_output - <<-EOM
		binary = true
		vim-syntax-type = "test"
	EOM

  run_dodder show !md:t
  assert_success
  assert_output - <<-EOM
		[!md @blake2b256-76m5lj0dp3je79ft9z2mdwpcrzrf9sddj04tvewpuk6gyqmll27sz46w72 !toml-type-v1]
	EOM

  run_dodder show -format type.vim-syntax-type !md:typ
  assert_success
  assert_output 'toml'

  run_dodder show -format type.vim-syntax-type one/uno
  assert_success
  assert_output 'test'
}

function checkin_simple_tag { # @test
  run_dodder checkin zz-archive.tag
  # run_dodder checkin zz-archive.e
  assert_success
  assert_output - <<-EOM
		[zz-archive @blake2b256-4nnaw9wx7vwsdlx777qf48drgxeatj762ykhlwhe6pykmmutglvsz2szgt]
	EOM

  run_dodder last -format inventory_list-sans-tai
  assert_success
  assert_output --regexp - <<-'EOM'
		\[zz-archive @blake2b256-4nnaw9wx7vwsdlx777qf48drgxeatj762ykhlwhe6pykmmutglvsz2szgt .*]
	EOM

  run_dodder show -format blob zz-archive?e
  assert_success
  assert_output - <<-EOM
		hide = true
	EOM
}

function checkin_zettel_typ_has_commit_hook { # @test
  cat >typ_with_hook.type <<-EOM
		hooks = """
		return {
		  on_new = function (kinder)
		    kinder["Etiketten"]["on_new"] = true
		    return nil
		  end,
		  on_pre_commit = function (kinder, mutter)
		    kinder["Etiketten"]["on_pre_commit"] = true
		    return nil
		  end,
		}
		"""
	EOM

  run_dodder checkin -delete typ_with_hook.type
  assert_success
  assert_output - <<-EOM
		[!typ_with_hook @blake2b256-h5ydwl76wjenz32ujma0qgse2fv4xxh992rjyv5k6uxe5vr6ul9qvcjskm !toml-type-v1]
		          deleted [typ_with_hook.type]
	EOM

  run_dodder new -edit=false - <<-EOM
		---
		# test lua
		! typ_with_hook
		---

		should add new etikett
	EOM
  assert_success
  assert_output - <<-EOM
		[two/uno @blake2b256-hhew85kxn9usmuqxalnupnt2jpwwlje3m68y6v0kyr4yqj9w49vq9w79lk !typ_with_hook "test lua" on_new on_pre_commit]
	EOM
}

function checkin_zettel_with_komment { # @test
  run_dodder checkin -print-inventory_list=true -comment "message" one/uno.zettel
  assert_success
  assert_output --regexp - <<-'EOM'
		\[one/uno @blake2b256-k87yyah5da3c8h9j4ugf44edeurrqztn7zddh7ksc88pfg4zzx0smqmuf9 !md "wildly different" etikett-one\]
		\[[0-9]+\.[0-9]+ @blake2b256-.* !inventory_list-v2 "message"\]
	EOM
}

function checkin_via_organize { # @test
  export EDITOR="true"
  run_dodder checkin -organize one/uno.zettel
  assert_success
  assert_output - <<-'EOM'
		[one/uno @blake2b256-k87yyah5da3c8h9j4ugf44edeurrqztn7zddh7ksc88pfg4zzx0smqmuf9 !md "wildly different" etikett-one]
	EOM
}

# bats test_tags=user_story:fs_blobs, user_story:external_ids
function checkin_dot_untracked_fs_blob() { # @test
  cat >test.md <<-EOM
		newest body
	EOM

  run_dodder checkin .
  assert_success
  assert_output_unsorted - <<-EOM
		[!md @blake2b256-76m5lj0dp3je79ft9z2mdwpcrzrf9sddj04tvewpuk6gyqmll27sz46w72 !toml-type-v1]
		[one/dos @blake2b256-wn30f7j6g62r7lgz0jhmnapnkem09c7lkkv65k005wv3fnj44m7q6auex2 !md "dos wildly different" etikett-two]
		[one/uno @blake2b256-k87yyah5da3c8h9j4ugf44edeurrqztn7zddh7ksc88pfg4zzx0smqmuf9 !md "wildly different" etikett-one]
		[two/uno @blake2b256-k87yyah5da3c8h9j4ugf44edeurrqztn7zddh7ksc88pfg4zzx0smqmuf9 !md "test"]
		[zz-archive @blake2b256-4nnaw9wx7vwsdlx777qf48drgxeatj762ykhlwhe6pykmmutglvsz2szgt]
	EOM
}

# bats test_tags=user_story:fs_blobs, user_story:external_ids
function checkin_explicit_untracked_fs_blob() { # @test
  cat >test.md <<-EOM
		newest body
	EOM

  run_dodder checkin test.md
  assert_success
  assert_output - <<-EOM
		[two/uno @blake2b256-k87yyah5da3c8h9j4ugf44edeurrqztn7zddh7ksc88pfg4zzx0smqmuf9 !md "test"]
	EOM
}

# bats test_tags=user_story:fs_blobs, user_story:organize, user_story:editor, user_story:external_ids
function checkin_dot_organize_exclude_untracked_fs_blob() { # @test
  cat >test.md <<-EOM
		newest body
	EOM

  export EDITOR="true"
  run_dodder checkin -organize .
  assert_success
  assert_output_unsorted - <<-EOM
		[!md @blake2b256-76m5lj0dp3je79ft9z2mdwpcrzrf9sddj04tvewpuk6gyqmll27sz46w72 !toml-type-v1]
		[one/dos @blake2b256-wn30f7j6g62r7lgz0jhmnapnkem09c7lkkv65k005wv3fnj44m7q6auex2 !md "dos wildly different" etikett-two]
		[one/uno @blake2b256-k87yyah5da3c8h9j4ugf44edeurrqztn7zddh7ksc88pfg4zzx0smqmuf9 !md "wildly different" etikett-one]
		[two/uno @blake2b256-k87yyah5da3c8h9j4ugf44edeurrqztn7zddh7ksc88pfg4zzx0smqmuf9 !md "test"]
		[zz-archive @blake2b256-4nnaw9wx7vwsdlx777qf48drgxeatj762ykhlwhe6pykmmutglvsz2szgt]
	EOM
}

# bats test_tags=user_story:fs_blobs, user_story:organize, user_story:editor, user_story:external_ids
function checkin_explicit_organize_include_untracked_fs_blob() { # @test
  cat >test.md <<-EOM
		newest body
	EOM

  export EDITOR="bash -c 'true'"
  run_dodder checkin -organize test.md </dev/null
  assert_success
  assert_output - <<-EOM
		[two/uno @blake2b256-k87yyah5da3c8h9j4ugf44edeurrqztn7zddh7ksc88pfg4zzx0smqmuf9 !md "test"]
	EOM
}

# bats test_tags=user_story:fs_blobs, user_story:organize, user_story:editor, user_story:external_ids
function checkin_explicit_organize_include_untracked_fs_blob_change_description() { # @test
  cat >test.md <<-EOM
		newest body
	EOM

  cat >desired_end_state.md <<-EOM
		  - [test.md some_tag] a different description
	EOM

  export EDITOR="bash -c 'cat desired_end_state.md >\$0'"
  run_dodder checkin -organize test.md </dev/null
  assert_success
  assert_output - <<-EOM
		[two/uno @blake2b256-k87yyah5da3c8h9j4ugf44edeurrqztn7zddh7ksc88pfg4zzx0smqmuf9 !md "a different description" some_tag]
	EOM
}

# bats test_tags=user_story:fs_blobs, user_story:organize, user_story:editor, user_story:external_ids
function checkin_dot_organize_include_untracked_fs_blob() { # @test
  cat >test.md <<-EOM
		newest body
	EOM

  export EDITOR="bash -c 'true'"
  run_dodder checkin -organize . </dev/null
  assert_success
  assert_output_unsorted - <<-EOM
		[!md @blake2b256-76m5lj0dp3je79ft9z2mdwpcrzrf9sddj04tvewpuk6gyqmll27sz46w72 !toml-type-v1]
		[one/dos @blake2b256-wn30f7j6g62r7lgz0jhmnapnkem09c7lkkv65k005wv3fnj44m7q6auex2 !md "dos wildly different" etikett-two]
		[one/uno @blake2b256-k87yyah5da3c8h9j4ugf44edeurrqztn7zddh7ksc88pfg4zzx0smqmuf9 !md "wildly different" etikett-one]
		[two/uno @blake2b256-k87yyah5da3c8h9j4ugf44edeurrqztn7zddh7ksc88pfg4zzx0smqmuf9 !md "test"]
		[zz-archive @blake2b256-4nnaw9wx7vwsdlx777qf48drgxeatj762ykhlwhe6pykmmutglvsz2szgt]
	EOM
}

# bats test_tags=user_story:fs_blobs, user_story:organize, user_story:editor, user_story:external_ids
function checkin_dot_include_untracked_fs_blob_with_spaces() { # @test
  cat >"test with spaces.txt" <<-EOM
		newest body
	EOM

  run_dodder checkin "test with spaces.txt" </dev/null
  assert_success
  assert_output_unsorted - <<-EOM
		[!txt !toml-type-v1]
		[two/uno @blake2b256-k87yyah5da3c8h9j4ugf44edeurrqztn7zddh7ksc88pfg4zzx0smqmuf9 !txt "test with spaces"]
	EOM
}

# bats test_tags=user_story:fs_blobs, user_story:organize, user_story:editor, user_story:external_ids
function checkin_dot_organize_include_untracked_fs_blob_with_spaces() { # @test
  cat >"test with spaces.txt" <<-EOM
		newest body
	EOM

  export EDITOR="bash -c 'true'"
  run_dodder checkin -organize "test with spaces.txt" </dev/null
  assert_success
  assert_output_unsorted - <<-EOM
		[!txt !toml-type-v1]
		[two/uno @blake2b256-k87yyah5da3c8h9j4ugf44edeurrqztn7zddh7ksc88pfg4zzx0smqmuf9 !txt "test with spaces"]
	EOM
}

# bats test_tags=user_story:organize,user_story:workspace
function checkin_explicit_workspace_delete_files { # @test
  # shellcheck disable=SC2317
  function editor() (
    sed -i "s/tags = \\[]/tags = ['zz-inbox']/" "$0"
    # sed -i "/type = '!md'/a tags = 'hello'" "$0"
  )

  export -f editor

  export EDITOR="/bin/bash -c 'editor \$0'"
  run_dodder edit-config
  assert_success
  assert_output - <<-EOM
		[konfig @blake2b256-5y2xv5tx28jrkns3u8q4skx4xxjpfhzsl93njr2arpy5n8xsngwqkcay07 !toml-config-v2]
	EOM

  cat >.dodder-workspace <<-EOM
		---
		! toml-workspace_config-v0
		---

		query = "today"

		[defaults]
		tags = ["today"]
	EOM

  run_dodder info-workspace query
  assert_success
  assert_output 'today'

  echo "file one" >1.md
  echo "file two" >2.md

  function editor() {
    # shellcheck disable=SC2317
    cat - >"$1" <<-EOM
			---
			% instructions: to prevent an object from being checked in, delete it entirely
			% delete:true delete once checked in
			- today
			---

			- [1.md]
			- [2.md]
		EOM
  }

  export -f editor

  # shellcheck disable=SC2016
  export EDITOR='bash -c "editor $0"'

  run_dodder checkin -organize -delete 1.md 2.md
  assert_success
  assert_output - <<-EOM
		[two/uno @blake2b256-p7nw3egtdfeacsvdjmcafmf9clmyd44fqdys8myavatteaun9w3sc7yqe7 !md "2" today]
		[one/tres @blake2b256-v2wlxr328lxnhxtyfz92gsfhfxyqslt5q4gux5hnmqugt7qftntszp3d24 !md "1" today]
		          deleted [1.md]
		          deleted [2.md]
	EOM
}

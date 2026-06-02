#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

  # for shellcheck SC2154
  export output

  setup_repo
}

teardown() {
  teardown_repo
}

# bats file_tags=user_story:query

function show_simple_one_zettel { # @test
  run_dodder show -format text one/uno
  assert_success
  assert_output - <<-EOM
		---
		# wow the first
		- tag-3
		- tag-4
		@ blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd
		! md@$(get_fixture_type_sig)
		---
	EOM
}

function show_simple_one_zettel_with_description_with_quotes { # @test
  run_dodder init-workspace -experimental-repo=false
  assert_success

  run_dodder new -edit=false - <<-EOM
		---
		# see these "quotes"
		! md
		---

		last time
	EOM
  assert_success
  assert_output - <<-EOM
		[two/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "see these \"quotes\""]
	EOM

  run_dodder show -format text two/uno:
  assert_success
  assert_output - <<-EOM
		---
		# see these "quotes"
		@ blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd
		! md@$(get_fixture_type_sig)
		---
	EOM
}

function show_simple_one_zettel_with_sigil { # @test
  run_dodder show -format text one/uno:
  assert_success
  assert_output - <<-EOM
		---
		# wow the first
		- tag-3
		- tag-4
		@ blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd
		! md@$(get_fixture_type_sig)
		---
	EOM
}

function show_simple_one_zettel_with_sigil_and_genre { # @test
  run_dodder show -format text one/uno:zettel
  assert_success
  assert_output - <<-EOM
		---
		# wow the first
		- tag-3
		- tag-4
		@ blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd
		! md@$(get_fixture_type_sig)
		---
	EOM
}

function show_simple_one_zettel_checked_out { # @test
  run_dodder init-workspace -experimental-repo=false
  assert_success

  run_dodder checkout one/uno
  assert_success
  assert_output - <<-EOM
		      checked out [one/uno.zettel @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM

  run_dodder show one/uno.zettel
  assert_success
  assert_output --regexp - <<-EOM
		\[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4 object=/.+/one/uno\.zettel]
	EOM
}

function show_simple_one_zettel_hidden { # @test
  run_dodder dormant-add tag-3
  assert_success
  assert_output ''

  run_dodder show :z
  assert_success
  assert_output ''

  run_dodder show :?z
  assert_success
  assert_output_unsorted - <<-EOM
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM

  run_dodder show one/uno
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM
}

function show_simple_one_zettel_hidden_past { # @test
  run_dodder dormant-add tag-1
  assert_success
  assert_output ''

  run_dodder show :?z
  assert_success
  assert_output_unsorted - <<-EOM
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
	EOM

  run_dodder show one/uno
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM
}

function show_all_mother { # @test
  run_dodder show -format sig-mother :
  assert_success
  assert_output_unsorted --regexp - <<-EOM
		ed25519_sig-
	EOM
}

# bats test_tags=user_story:workspace
function show_simple_one_zettel_binary { # @test
  skip
  echo "binary file" >file.bin
  run_dodder add -delete file.bin
  assert_success
  assert_output_unsorted - <<-EOM
		          deleted [file.bin]
		[!bin !toml-type-v2]
		[two/uno @blake2b256-w9l3z9c2w8lhr42fwekmhrxeqtmzw40s9p46vt88ydgwux4rxxuqnfqsmk !bin "file"]
	EOM

  cat >bin.type <<-EOM
		---
		! toml-type-v2
		---

		binary = true
	EOM

  run_dodder checkin -delete bin.type
  assert_success
  assert_output_unsorted - <<-EOM
		          deleted [bin.type]
		[!bin @blake2b256-zhvux7vmpch9f44kvnua7n69f8jzgk5s7p9k2s3kuvkrcpjh07lse493jl !toml-type-v2]
	EOM

  run_dodder show -format text two/uno
  assert_success
  assert_output - <<-EOM
		---
		# file
		! b20c8fea8cb3e467783c5cdadf0707124cac5db72f9a6c3abba79fa0a42df627.bin
		---
	EOM
}

function show_history_one_zettel { # @test
  run_dodder show one/uno+z
  assert_success
  assert_output_unsorted - <<-EOM
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
		[one/uno @blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 !md "wow ok" tag-1 tag-2]
	EOM

  run_dodder show -format text one/uno+z
  assert_success
  assert_output_unsorted - <<-EOM
		---
		# wow ok
		- tag-1
		- tag-2
		@ blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0
		! md@$(get_fixture_type_sig)
		---
		---
		# wow the first
		- tag-3
		- tag-4
		@ blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd
		! md@$(get_fixture_type_sig)
		---
	EOM
}

function show_zettel_tag { # @test
  run_dodder show tag-3:z
  assert_success
  assert_output_unsorted - <<-EOM
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM

  run_dodder show -format blob tag-3:z
  assert_success
  assert_output_unsorted - <<-EOM
		last time
		not another one
	EOM

  run_dodder show -format sku-metadata-sans-tai tag-3:z
  assert_success
  assert_output_unsorted - <<-EOM
		Zettel one/dos blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md tag-3 tag-4 "wow ok again"
		Zettel one/uno blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md tag-3 tag-4 "wow the first"
	EOM
}

function show_zettels_with_tag_no_workspace_folder { # @test
  mkdir -p tag
  echo "wow1" >tag/test1
  echo "wow2" >tag/test2
  run_dodder show tag
  assert_success
  assert_output_unsorted - <<-EOM
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM
}

function show_zettel_tag_complex { # @test
  run_dodder init-workspace -experimental-repo=false
  assert_success

  run_dodder checkout o/u
  assert_success
  assert_output - <<-EOM
		      checked out [one/uno.zettel @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM

  cat >one/uno.zettel <<-EOM
		---
		# wow the first
		- tag-3
		- tag-5
		! md
		---

		last time
	EOM

  # TODO support . operator for checked out
  # run_dodder show -verbose tag-3.z tag-5.z
  # assert_success
  # assert_output_unsorted - <<-EOM
  # 	[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-5]
  # EOM

  run_dodder checkin -delete one/uno.zettel

  run_dodder show [tag-3 tag-5]:z
  assert_success
  assert_output_unsorted - <<-EOM
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-5]
	EOM

  run_dodder show -format blob [tag-3 tag-5]:z
  assert_success
  assert_output_unsorted - <<-EOM
		last time
	EOM

  run_dodder show -format sku-metadata-sans-tai [tag-3 tag-5]:z
  assert_success
  assert_output_unsorted --partial - <<-EOM
		Zettel one/uno blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md tag-3 tag-5 "wow the first"
	EOM
}

function show_complex_zettel_tag_negation { # @test
  run_dodder show ^-etikett-two:z
  assert_success
  assert_output_unsorted - <<-EOM
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM
}

function show_simple_all { # @test
  run_dodder show :z,t
  assert_success
  assert_output_unsorted - <<-EOM
		[!md @$(get_type_blob_sha) !toml-type-v2]
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM

  run_dodder show -format blob :z,t
  assert_success
  assert_output_unsorted - <<-EOM
		file-extension = "md"
		last time
		not another one
		vim-syntax-type = "markdown"
	EOM

  run_dodder show -format sku-metadata-sans-tai :z,t
  assert_success
  assert_output_unsorted - <<-EOM
		Type !md blake2b256-45v3c002j9xfjguu2a7ljxnf68tqglg8fa0csjgnn7d2n36ltp0snfjxgj !toml-type-v2
		Zettel one/dos blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md tag-3 tag-4 "wow ok again"
		Zettel one/uno blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md tag-3 tag-4 "wow the first"
	EOM
}

function show_simple_type_one { # @test
  run_dodder show !md:t
  assert_output_unsorted - <<-EOM
		[!md @$(get_type_blob_sha) !toml-type-v2]
	EOM
}

function show_simple_type_one_history { # @test
  run_dodder show !md+t
  assert_output_unsorted - <<-EOM
		[!md @$(get_type_blob_sha) !toml-type-v2]
	EOM
}

function show_simple_type_tail { # @test
  run_dodder show :t
  assert_output_unsorted - <<-EOM
		[!md @$(get_type_blob_sha) !toml-type-v2]
	EOM
}

function show_simple_type_history { # @test
  run_dodder show +t
  assert_output_unsorted - <<-EOM
		[!md @$(get_type_blob_sha) !toml-type-v2]
	EOM
}

function show_simple_tag_tail { # @test
  run_dodder show :e
  assert_output_unsorted - <<-EOM
	EOM
}

function show_simple_tag_history { # @test
  run_dodder show +e
  assert_output_unsorted - <<-EOM
	EOM
}

function show_konfig { # @test
  run_dodder show +konfig
  assert_output_unsorted - <<-EOM
		[konfig @$(get_konfig_sha) !toml-config-v2]
	EOM

  run_dodder show -format text :konfig
  assert_output - <<-EOM
		blob-stores = [".default"]

		[defaults]
		type = "!md"
		tags = []

		[file-extensions]
		config = "konfig"
		conflict = "conflict"
		lockfile = "object-lockfile"
		organize = "md"
		repo = "repo"
		tag = "tag"
		type = "type"
		zettel = "zettel"

		[cli-output]
		print-blob_digests = true
		print-colors = true
		print-empty-blob_digests = false
		print-flush = true
		print-include-description = true
		print-include-types = true
		print-inventory_lists = true
		print-matched-dormant = false
		print-tags-always = true
		print-time = true
		print-unchanged = true

		[cli-output.abbreviations]
		zettel_ids = true
		merkle_ids = true

		[tools]
		merge = ["vimdiff"]
	EOM
}

function show_history_all { # @test
  run_dodder show +konfig,kasten,typ,etikett,zettel
  assert_output_unsorted - <<-EOM
		[konfig @$(get_konfig_sha) !toml-config-v2]
		[!md @$(get_type_blob_sha) !toml-type-v2]
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
		[one/uno @blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 !md "wow ok" tag-1 tag-2]
	EOM
}

# bats test_tags=user_story:workspace
function show_tag_toml { # @test
  skip
  cat >true.tag <<-EOM
		---
		! toml-tag-v1
		---

		filter = """
		return {
		  contains_sku = function (sk)
		    return true
		  end
		}
		"""
	EOM

  run_dodder checkin -delete true.tag
  assert_success
  assert_output_unsorted - <<-EOM
		          deleted [true.tag]
		[true @1379cb8d553a340a4d262b3be216659d8d8835ad0b4cc48005db8db264a395ed !toml-tag-v1]
	EOM

  run_dodder show true
  assert_success
  assert_output_unsorted - <<-EOM
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM
}

# TODO fix race condition between stderr and stdout
# bats test_tags=user_story:workspace, user_story:lua_tags
function show_tag_lua_v1 { # @test
  skip
  cat >true.tag <<-EOM
		---
		! lua-tag-v1
		---

		return {
		  contains_sku = function (sk)
		    print(Selbst.Kennung)
		    return true
		  end
		}
	EOM

  run_dodder checkin -delete true.tag
  assert_success
  assert_output_unsorted - <<-EOM
		          deleted [true.tag]
		[true @67b7eb3e9ea1c4b3404b34a0b2abcc09f450797c8cc801671463a79429aead37 !lua-tag-v1]
	EOM

  run_dodder show true
  assert_success
  assert_output_unsorted - <<-EOM
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
		true
		true
	EOM
}

# TODO fix race condition between stderr and stdout
# bats test_tags=user_story:workspace, user_story:lua_tags
function show_tag_lua_v2 { # @test
  skip
  cat >true.tag <<-EOM
		---
		! lua-tag-v2
		---

		return {
		  contains_sku = function (sk)
		    print(Self.ObjectId)
		    return true
		  end
		}
	EOM

  run_dodder checkin -delete true.tag
  assert_success
  assert_output_unsorted - <<-EOM
		          deleted [true.tag]
		[true @ed8e3cf53e044fcc1ae040ed5203515d1c6d205decc745f0caafd5dee67efbab !lua-tag-v2]
	EOM

  run_dodder show true
  assert_success
  assert_output_unsorted - <<-EOM
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
		true
		true
	EOM
}

function show_tags_paths { # @test
  run_dodder show -format tags-path :e
  assert_success
  assert_output_unsorted - <<-EOM
	EOM
}

function show_tags_exact { # @test
  run_dodder show =tag:e
  assert_success
  assert_output_unsorted - <<-EOM
	EOM

  run_dodder show =tag
  assert_success
  assert_output_unsorted ''
}

function show_inventory_lists { # @test
  run_dodder show :b
  assert_success
  assert_output --regexp - <<-'EOM'
		\[[0-9]+\.[0-9]+ @blake2b256-.+ !inventory_list-v2]
		\[[0-9]+\.[0-9]+ @blake2b256-.+ !inventory_list-v2]
		\[[0-9]+\.[0-9]+ @blake2b256-.+ !inventory_list-v2]
		\[[0-9]+\.[0-9]+ @blake2b256-.+ !inventory_list-v2]
	EOM
}

function show_inventory_list_blob_sort_correct { # @test
  function assert_sorted_tais() {
    echo -n "$1" | run sort -n -c -
    assert_success
  }

  run_dodder show -format tai :b
  assert_success
  assert_sorted_tais "$output"
  mapfile -t tais <<<"$output"

  for tai in "${tais[@]}"; do
    run_dodder show -format blob "$tai:b"
    assert_success
    listTais="$(echo -n "$output" | grep -o '[0-9]\+\.[0-9]\+')"
    assert_sorted_tais "$listTais"
  done
}

# bats test_tags=user_story:builtin_types
function show_builtin_type_md { # @test
  run_dodder show -format text !toml-type-v2:t
  assert_success
  assert_output - <<-EOM
		---
		! toml-type-v2
		---

		file-extension = "md"
		vim-syntax-type = "markdown"
	EOM
}

# bats file_tags=user_story:workspace

function show_workspace_default { # @test
  run_dodder organize -mode commit-directly one/uno <<-EOM
		- [one/uno !md tag-3 tag-4 tag-5] wow the first
	EOM
  assert_success
  assert_output_unsorted - <<-EOM
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4 tag-5]
	EOM

  run_dodder init-workspace -experimental-repo=false -query tag-5
  assert_success

  run_dodder show :
  assert_success
  assert_output_unsorted - <<-EOM
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4 tag-5]
	EOM
}

function show_workspace_exactly_one_zettel { # @test
  skip
  run_dodder organize -mode commit-directly one/uno <<-EOM
		- [one/uno !md tag-3 tag-4 tag-5] wow the first
	EOM
  assert_success
  assert_output_unsorted - <<-EOM
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4 tag-5]
		[tag-5]
	EOM

  run_dodder init-workspace -experimental-repo=false -query tag-3
  assert_success

  run_dodder show one/dos
  assert_success
  assert_output_unsorted - <<-EOM
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4 tag-5]
	EOM
}

# bats test_tags=user_story:referenced_objects
function show_zettel_with_referenced_object_lock { # @test
  run_dodder init-workspace -experimental-repo=false
  assert_success

  run_dodder new -edit=false - <<-EOM
		---
		# referencing zettel
		- one/dos
		! md
		---

		some content
	EOM
  assert_success

  run_dodder show -format text two/uno:
  assert_success
  assert_output --regexp - <<-'EOM'
		---
		# referencing zettel
		@ blake2b256-.+
		! md@.+
		- one/dos@ed25519_sig-.+
		---
	EOM
}

# bats test_tags=user_story:referenced_objects
function show_zettel_with_discovered_references { # @test
  run_dodder init-workspace -experimental-repo=false
  assert_success

  # Create a type with reference discovery script
  cat >ref-md.type <<-'TYPEFILE'
		---
		! toml-type-v2
		---

		file-extension = "md"
		vim-syntax-type = "markdown"

		[references]
		shell = ["bash", "-c"]
		script = "grep -oP '\\[\\[(.+?)\\]\\]' | sed 's/\\[\\[//;s/\\]\\]//'"
	TYPEFILE

  run_dodder checkin -delete ref-md.type
  assert_success

  # Create a zettel of type ref-md with a wiki-link to one/dos
  run_dodder new -edit=false - <<-EOM
		---
		# zettel with wiki link
		! ref-md
		---

		Check out [[one/dos]] for more info.
	EOM
  assert_success

  # Show the new zettel and verify the reference lock was auto-discovered
  run_dodder show -format text two/uno:
  assert_success
  assert_output --regexp - <<-'EOM'
		---
		# zettel with wiki link
		@ blake2b256-.+
		! ref-md@.+
		- one/dos@ed25519_sig-.+
		---
	EOM
}

# bats test_tags=user_story:referenced_objects
function show_zettel_with_pandoc_discovered_references { # @test
  run_dodder init-workspace -experimental-repo=false
  assert_success

  # Create a type with pandoc-based reference discovery
  cat >ref-pandoc-md.type <<-TYPEFILE
		---
		! toml-type-v2
		---

		file-extension = "md"
		vim-syntax-type = "markdown"

		[references]
		shell = ["pandoc", "--from", "markdown+wikilinks_title_after_pipe", "--to"]
		script = "$DIR/../zz-pandoc-refs/discover-refs.lua"
	TYPEFILE

  run_dodder checkin -delete ref-pandoc-md.type
  assert_success

  # Create a zettel of type ref-pandoc-md with a wiki-link to one/dos
  run_dodder new -edit=false - <<-EOM
		---
		# zettel with pandoc wiki link
		! ref-pandoc-md
		---

		Check out [[one/dos]] for more info.
	EOM
  assert_success

  # Show the new zettel and verify the reference was auto-discovered via pandoc
  run_dodder show -format text two/uno:
  assert_success
  assert_output --regexp - <<-'EOM'
		---
		# zettel with pandoc wiki link
		@ blake2b256-.+
		! ref-pandoc-md@.+
		- one/dos@ed25519_sig-.+
		---
	EOM
}

# bats test_tags=user_story:referenced_objects
function show_zettel_with_pandoc_discovered_code_block_type_references { # @test
  run_dodder init-workspace -experimental-repo=false
  assert_success

  # Create a type with pandoc-based reference discovery
  cat >ref-pandoc-cb.type <<-TYPEFILE
		---
		! toml-type-v2
		---

		file-extension = "md"
		vim-syntax-type = "markdown"

		[references]
		shell = ["pandoc", "--from", "markdown+wikilinks_title_after_pipe", "--to"]
		script = "$DIR/../zz-pandoc-refs/discover-refs.lua"
	TYPEFILE

  run_dodder checkin -delete ref-pandoc-cb.type
  assert_success

  # Create a zettel with a fenced code block using a !md type prefix
  run_dodder new -edit=false - <<-EOM
		---
		# zettel with code block type ref
		! ref-pandoc-cb
		---

		Some text before.

		\`\`\`!md
		# Hello World
		This is embedded markdown.
		\`\`\`

		Some text after.
	EOM
  assert_success

  # Show the new zettel and verify the code block type reference was discovered
  run_dodder show -format text two/uno:
  assert_success
  assert_output --regexp - <<-'EOM'
		---
		# zettel with code block type ref
		@ blake2b256-.+
		! ref-pandoc-cb@.+
		- !md@ed25519_sig-.+
		---
	EOM
}

# bats test_tags=user_story:format_stdin
function format_blob_stdin_resolves_type_with_and_without_lock { # @test
  run_dodder init-workspace -experimental-repo=false
  assert_success

  # Create a type with a pandoc formatter (markdown → plain text)
  cat >fmt-test.type <<-'TYPEFILE'
		---
		! toml-type-v2
		---

		file-extension = "txt"

		[formatters.text]
		shell = ["pandoc", "-f", "markdown", "-t", "plain", "--wrap=none"]
	TYPEFILE

  run_dodder checkin -delete fmt-test.type
  assert_success

  # Create a zettel with that type to capture the type lock
  run_dodder new -edit=false - <<-EOM
		---
		# test zettel
		! fmt-test
		---

		test content
	EOM
  assert_success

  # Get the type sig from the zettel text output
  run_dodder show -format text two/uno:
  assert_success
  type_sig=$(echo "$output" | grep '! fmt-test@' | sed 's/.*@//')
  [[ -n $type_sig ]]

  # Test format-blob -stdin with unlocked type (no lock)
  run_dodder format-blob -stdin !fmt-test <<<"hello world"
  assert_success
  assert_output "hello world"

  # Test format-blob -stdin with locked type (pinned version)
  run_dodder format-blob -stdin "!fmt-test@${type_sig}" <<<"hello world"
  assert_success
  assert_output "hello world"
}

# bats test_tags=user_story:format_stdin
function format_blob_stdin_selects_formatter_via_uti_group { # @test
  run_dodder init-workspace -experimental-repo=false
  assert_success

  # Create a type with text-edit and text-render UTI groups pointing to
  # different pandoc output formats, mirroring the real !md type's pattern.
  # Using **bold** input, each format produces clearly different output:
  #   markdown preserves it, html wraps in tags, plain strips formatting
  cat >uti-test.type <<-'TYPEFILE'
		---
		! toml-type-v2
		---

		file-extension = "txt"

		[uti-groups.text-edit]
		"public.utf8-plain-text" = "edit-fmt"

		[uti-groups.text-render]
		"public.utf8-plain-text" = "render-fmt"

		[formatters.text]
		shell = ["pandoc", "-f", "markdown", "-t", "markdown", "--wrap=none"]

		[formatters.edit-fmt]
		shell = ["pandoc", "-f", "markdown", "-t", "html", "--wrap=none"]

		[formatters.render-fmt]
		shell = ["pandoc", "-f", "markdown", "-t", "plain", "--wrap=none"]
	TYPEFILE

  run_dodder checkin -delete uti-test.type
  assert_success

  # Default format (text) preserves markdown formatting
  run_dodder format-blob -stdin !uti-test <<<"**hello**"
  assert_success
  assert_output "**hello**"

  # text-edit UTI group selects the html formatter
  run_dodder format-blob -stdin -uti-group text-edit public.utf8-plain-text !uti-test <<<"**hello**"
  assert_success
  assert_output "<p><strong>hello</strong></p>"

  # text-render UTI group selects the plain text formatter
  run_dodder format-blob -stdin -uti-group text-render public.utf8-plain-text !uti-test <<<"**hello**"
  assert_success
  assert_output "hello"
}

# bats test_tags=user_story:format_stdin
function format_blob_prefers_text_edit_over_text { # @test
  run_dodder init-workspace -experimental-repo=false
  assert_success

  # Create a type where text-edit produces html and text produces plain —
  # when no format is specified, format-blob should prefer text-edit
  cat >edit-pref.type <<-'TYPEFILE'
		---
		! toml-type-v2
		---

		file-extension = "txt"

		[formatters.text]
		shell = ["pandoc", "-f", "markdown", "-t", "plain", "--wrap=none"]

		[formatters.text-edit]
		shell = ["pandoc", "-f", "markdown", "-t", "html", "--wrap=none"]
	TYPEFILE

  run_dodder checkin -delete edit-pref.type
  assert_success

  run_dodder new -edit=false - <<-EOM
		---
		# pref test
		! edit-pref
		---

		some content
	EOM
  assert_success

  # Non-stdin with no explicit format should prefer text-edit over text
  run_dodder format-blob two/uno
  assert_success
  assert_output "<p>some content</p>"
}

# bats test_tags=user_story:referenced_objects
function show_zettel_with_discovered_blob_references { # @test
  run_dodder init-workspace -experimental-repo=false
  assert_success

  # Create a type with reference discovery that outputs typed blob refs
  cat >ref-blob.type <<-'TYPEFILE'
		---
		! toml-type-v2
		---

		file-extension = "md"
		vim-syntax-type = "markdown"

		[references]
		shell = ["bash", "-c"]
		script = "grep -oP '(@blake2b256-[a-z0-9]+|\\[\\[(.+?)\\]\\])' | sed 's/\\[\\[//;s/\\]\\]//' | sed 's/^@\\(blake2b256-[a-z0-9]*\\)/@\\1 !md/'"
	TYPEFILE

  run_dodder checkin -delete ref-blob.type
  assert_success

  # Create a zettel with both a wiki-link and a blob reference
  run_dodder new -edit=false - <<-EOM
		---
		# zettel with blob ref
		! ref-blob
		---

		See [[one/dos]] and blob @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd for details.
	EOM
  assert_success

  # Show the new zettel and verify both reference types appear with type lock
  run_dodder show -format text two/uno:
  assert_success
  assert_output --regexp - <<-'EOM'
		---
		# zettel with blob ref
		@ blake2b256-.+
		! ref-blob@.+
		- one/dos@ed25519_sig-.+
		- @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md@ed25519_sig-.+
		---
	EOM
}

# bats test_tags=user_story:referenced_objects
function blob_reference_without_type_fails { # @test
  run_dodder init-workspace -experimental-repo=false
  assert_success

  # Create a type whose reference discovery outputs untyped blob refs
  cat >ref-untyped.type <<-'TYPEFILE'
		---
		! toml-type-v2
		---

		file-extension = "md"
		vim-syntax-type = "markdown"

		[references]
		shell = ["bash", "-c"]
		script = "grep -oP '@blake2b256-[a-z0-9]+'"
	TYPEFILE

  run_dodder checkin -delete ref-untyped.type
  assert_success

  # Create a zettel with a blob reference — the discovery script outputs
  # @digest WITHOUT a type, so finalization should fail
  run_dodder new -edit=false - <<-EOM
		---
		# zettel with untyped blob ref
		! ref-untyped
		---

		See blob @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd for details.
	EOM
  assert_failure
}

# bats test_tags=user_story:referenced_objects
function discovery_script_crash_required_fails { # @test
  run_dodder init-workspace -experimental-repo=false
  assert_success

  # Type with required discovery script that exits non-zero
  cat >crashy.type <<-'TYPEFILE'
		---
		! toml-type-v2
		---

		file-extension = "md"

		[references]
		shell = ["bash", "-c"]
		script = "exit 1"
	TYPEFILE

  run_dodder checkin -delete crashy.type
  assert_success

  run_dodder new -edit=false - <<-EOM
		---
		# zettel with crashy type
		! crashy
		---

		content here
	EOM

  # Required script crash should cause commit failure
  assert_failure
}

# bats test_tags=user_story:referenced_objects
function discovery_script_crash_optional_succeeds { # @test
  run_dodder init-workspace -experimental-repo=false
  assert_success

  # Type with optional discovery script that exits non-zero
  cat >crashy-opt.type <<-'TYPEFILE'
		---
		! toml-type-v2
		---

		file-extension = "md"

		[references]
		optional = true
		shell = ["bash", "-c"]
		script = "exit 1"
	TYPEFILE

  run_dodder checkin -delete crashy-opt.type
  assert_success

  run_dodder new -edit=false - <<-EOM
		---
		# zettel with optional crashy type
		! crashy-opt
		---

		content here
	EOM

  # Optional script crash should succeed silently
  assert_success
}

# bats test_tags=user_story:referenced_objects
function show_box_format_includes_blob_references { # @test
  run_dodder init-workspace -experimental-repo=false
  assert_success

  # Create a type with reference discovery that outputs typed blob refs
  cat >ref-blob.type <<-'TYPEFILE'
		---
		! toml-type-v2
		---

		file-extension = "md"
		vim-syntax-type = "markdown"

		[references]
		shell = ["bash", "-c"]
		script = "grep -oP '(@blake2b256-[a-z0-9]+|\\[\\[(.+?)\\]\\])' | sed 's/\\[\\[//;s/\\]\\]//' | sed 's/^@\\(blake2b256-[a-z0-9]*\\)/@\\1 !md/'"
	TYPEFILE

  run_dodder checkin -delete ref-blob.type
  assert_success

  # Create a zettel with both a wiki-link and a blob reference
  run_dodder new -edit=false - <<-EOM
		---
		# zettel with refs
		! ref-blob
		---

		See [[one/dos]] and blob @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd for details.
	EOM
  assert_success

  # Verify box format includes blob reference with type lock
  run_dodder show two/uno
  assert_success
  assert_output --regexp '"<@blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md@ed25519_sig-.+"'
}

# bats test_tags=user_story:referenced_objects
function show_blob_references_sorted_in_hyphence { # @test
  run_dodder init-workspace -experimental-repo=false
  assert_success

  # Create a type whose reference discovery outputs multiple typed blob refs
  cat >ref-multi.type <<-'TYPEFILE'
		---
		! toml-type-v2
		---

		file-extension = "md"
		vim-syntax-type = "markdown"

		[references]
		shell = ["bash", "-c"]
		script = "grep -oP '@blake2b256-[a-z0-9]+' | sed 's/^@\\(blake2b256-[a-z0-9]*\\)/@\\1 !md/'"
	TYPEFILE

  run_dodder checkin -delete ref-multi.type
  assert_success

  # Content order: qyqs... then 9ft3... — but 9ft3 sorts before qyqs
  run_dodder new -edit=false - <<-EOM
		---
		# multi blob refs
		! ref-multi
		---

		First @blake2b256-qyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqsk2yde5 and second @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd here.
	EOM
  assert_success

  # Show in hyphence (text) format — blob refs must appear sorted
  run_dodder show -format text two/uno:
  assert_success

  # 9ft3 sorts before qyqs lexicographically — verify sorted order
  local line_9ft3 line_qyqs
  line_9ft3=$(echo "$output" | grep -n '@blake2b256-9ft3' | head -1 | cut -d: -f1)
  line_qyqs=$(echo "$output" | grep -n '@blake2b256-qyqs' | head -1 | cut -d: -f1)

  [[ -n $line_9ft3 ]] || fail "blob ref 9ft3 not found in output"
  [[ -n $line_qyqs ]] || fail "blob ref qyqs not found in output"
  [[ $line_9ft3 -lt $line_qyqs ]] || fail "blob refs not sorted: 9ft3 (line $line_9ft3) should appear before qyqs (line $line_qyqs)"
}

# bats test_tags=user_story:referenced_objects
function show_blob_references_sorted_in_inventory_list { # @test
  run_dodder init-workspace -experimental-repo=false
  assert_success

  # Create a type whose reference discovery outputs multiple typed blob refs
  cat >ref-multi.type <<-'TYPEFILE'
		---
		! toml-type-v2
		---

		file-extension = "md"
		vim-syntax-type = "markdown"

		[references]
		shell = ["bash", "-c"]
		script = "grep -oP '@blake2b256-[a-z0-9]+' | sed 's/^@\\(blake2b256-[a-z0-9]*\\)/@\\1 !md/'"
	TYPEFILE

  run_dodder checkin -delete ref-multi.type
  assert_success

  # Content order: qyqs... then 9ft3... — but 9ft3 sorts before qyqs
  run_dodder new -edit=false - <<-EOM
		---
		# multi blob refs
		! ref-multi
		---

		First @blake2b256-qyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqsk2yde5 and second @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd here.
	EOM
  assert_success

  # Show in box format (default) — blob refs appear as "<@digest !type@sig"
  run_dodder show two/uno
  assert_success

  # Verify 9ft3 appears before qyqs in the box format output
  local pos_9ft3 pos_qyqs
  pos_9ft3=$(echo "$output" | grep -boP '<@blake2b256-9ft3' | head -1 | cut -d: -f1)
  pos_qyqs=$(echo "$output" | grep -boP '<@blake2b256-qyqs' | head -1 | cut -d: -f1)

  [[ -n $pos_9ft3 ]] || fail "blob ref 9ft3 not found in box output"
  [[ -n $pos_qyqs ]] || fail "blob ref qyqs not found in box output"
  [[ $pos_9ft3 -lt $pos_qyqs ]] || fail "blob refs not sorted in box format: 9ft3 (pos $pos_9ft3) should appear before qyqs (pos $pos_qyqs)"
}

# bats test_tags=user_story:referenced_objects
function blob_ref_type_lock_succeeds_when_type_matches_zettel { # @test
  testdir="$BATS_TEST_TMPDIR/custom-blobref"
  mkdir -p "$testdir"
  pushd "$testdir" || exit 1

  run_dodder_init_disable_age

  # Create a type with discovery that emits blob refs typed as itself
  cat >img.type <<-'TYPEFILE'
		---
		! toml-type-v2
		---

		file-extension = "md"

		[references]
		shell = ["bash", "-c"]
		script = "grep -oP '@blake2b256-[a-z0-9]+' | sed 's/^@\\(blake2b256-[a-z0-9]*\\)/@\\1 !img/'"
	TYPEFILE

  run_dodder checkin -delete img.type
  assert_success

  # Blob ref type == zettel type, so !img type object is created as a side
  # effect of committing the zettel. Type lock resolution succeeds.
  run_dodder new -edit=false - <<-EOM
		---
		# same-type blob ref
		! img
		---

		See @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd here.
	EOM
  assert_success

  run_dodder show -format text one/uno:
  assert_success
  assert_output --regexp '@blake2b256-9ft3.+ !img'
}

# bats test_tags=user_story:referenced_objects
# https://github.com/amarbel-llc/dodder/issues/40
function blob_ref_type_lock_resolves_heterogeneous_types { # @test
  testdir="$BATS_TEST_TMPDIR/diff-blobref"
  mkdir -p "$testdir"
  pushd "$testdir" || exit 1

  run_dodder_init_disable_age

  # Create a custom type for blob references
  cat >img.type <<-'TYPEFILE'
		---
		! toml-type-v2
		---

		file-extension = "png"
	TYPEFILE

  run_dodder checkin -delete img.type
  assert_success

  # Create a discovery type that emits blob refs typed as !img
  cat >ref-img.type <<-'TYPEFILE'
		---
		! toml-type-v2
		---

		file-extension = "md"

		[references]
		shell = ["bash", "-c"]
		script = "grep -oP '@blake2b256-[a-z0-9]+' | sed 's/^@\\(blake2b256-[a-z0-9]*\\)/@\\1 !img/'"
	TYPEFILE

  run_dodder checkin -delete ref-img.type
  assert_success

  # Blob ref type (!img) differs from zettel type (!ref-img). Both were
  # created via checkin -delete, so !img must exist as a type object for
  # type lock resolution to succeed.
  run_dodder new -edit=false - <<-EOM
		---
		# diff-type blob ref
		! ref-img
		---

		See @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd here.
	EOM
  assert_success

  run_dodder show -format text one/uno:
  assert_success
  assert_output --regexp '@blake2b256-9ft3.+ !img'
}

# bats test_tags=user_story:referenced_objects
function show_box_format_includes_object_references { # @test
  run_dodder init-workspace -experimental-repo=false
  assert_success

  # Create a zettel that references another zettel
  run_dodder new -edit=false - <<-EOM
		---
		# referencing zettel
		- one/dos
		! md
		---

		references one/dos
	EOM
  assert_success

  # Verify box format includes object reference
  run_dodder show two/uno
  assert_success
  assert_output --regexp '<one/dos@ed25519_sig-.+'
}

# bats test_tags=user_story:referenced_objects
function object_reference_alias_with_quotes_round_trips { # @test
  run_dodder init-workspace -experimental-repo=false
  assert_success

  # Create a zettel with an aliased object reference containing a double quote
  run_dodder new -edit=false - <<-'EOM'
		---
		# alias with quotes
		- "say \"hello\"" < one/dos
		! md
		---

		test content
	EOM
  assert_success

  # Show the zettel in text format and verify the alias round-trips
  run_dodder show -format text two/uno:
  assert_success

  # The alias should contain the literal double quotes, properly escaped
  echo "$output" | grep -F 'say \"hello\"'
}

# bats test_tags=user_story:referenced_objects
function blob_reference_alias_with_quotes_round_trips { # @test
  run_dodder init-workspace -experimental-repo=false
  assert_success

  # Create a zettel with an aliased blob reference containing double quotes
  run_dodder new -edit=false - <<-'EOM'
		---
		# blob alias test
		- "say \"hello\"" < @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md
		! md
		---

		test content
	EOM
  assert_success

  # Show the zettel in text format and verify the alias round-trips
  run_dodder show -format text two/uno:
  assert_success

  # The alias should contain the literal double quotes, properly escaped
  echo "$output" | grep -F 'say \"hello\"'
}

# bats test_tags=user_story:referenced_objects
function blob_reference_alias_round_trips_through_box_format { # @test
  run_dodder init-workspace -experimental-repo=false
  assert_success

  # Create a zettel with an aliased blob reference
  run_dodder new -edit=false - <<-'EOM'
		---
		# alias box test
		- hero-image < @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md
		! md
		---

		content
	EOM
  assert_success

  # Box format should include the alias
  run_dodder show two/uno
  assert_success
  assert_output --regexp 'hero-image<@blake2b256-9ft3'
}

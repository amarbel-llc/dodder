#! /usr/bin/env bats

setup() {
	load "$(dirname "$BATS_TEST_FILE")/common.bash"

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
	assert_output --regexp - <<-EOM
		---
		# wow the first
		- tag-3
		- tag-4
		! md@.*
		---

		last time
	EOM
}

function show_simple_one_zettel_with_description_with_quotes { # @test
	run_dodder init-workspace
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
	assert_output --regexp - <<-EOM
		---
		# see these "quotes"
		! md@.*
		---

		last time
	EOM
}

function show_simple_one_zettel_with_sigil { # @test
	run_dodder show -format text one/uno:
	assert_success
	assert_output --regexp - <<-EOM
		---
		# wow the first
		- tag-3
		- tag-4
		! md@.*
		---

		last time
	EOM
}

function show_simple_one_zettel_with_sigil_and_genre { # @test
	run_dodder show -format text one/uno:zettel
	assert_success
	assert_output --regexp - <<-EOM
		---
		# wow the first
		- tag-3
		- tag-4
		! md@.*
		---

		last time
	EOM
}

function show_simple_one_zettel_checked_out { # @test
	run_dodder init-workspace
	assert_success

	run_dodder checkout one/uno
	assert_success
	assert_output - <<-EOM
		      checked out [one/uno.zettel @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM

	run_dodder show one/uno.zettel
	assert_success
	assert_output - <<-EOM
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
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
		[!bin !toml-type-v1]
		[two/uno @blake2b256-w9l3z9c2w8lhr42fwekmhrxeqtmzw40s9p46vt88ydgwux4rxxuqnfqsmk !bin "file"]
	EOM

	cat >bin.type <<-EOM
		---
		! toml-type-v1
		---

		binary = true
	EOM

	run_dodder checkin -delete bin.type
	assert_success
	assert_output_unsorted - <<-EOM
		          deleted [bin.type]
		[!bin @blake2b256-zhvux7vmpch9f44kvnua7n69f8jzgk5s7p9k2s3kuvkrcpjh07lse493jl !toml-type-v1]
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
	assert_output_unsorted --regexp - <<-EOM
		---
		# wow ok
		- tag-1
		- tag-2
		! md@.*
		---

		this is the body aiiiiight
		---
		# wow the first
		- tag-3
		- tag-4
		! md@.*
		---

		last time
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
	run_dodder init-workspace
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
		[!md @$(get_type_blob_sha) !toml-type-v1]
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM

	run_dodder show -format blob :z,t
	assert_success
	assert_output_unsorted - <<-EOM
		file-extension = 'md'
		last time
		not another one
		vim-syntax-type = 'markdown'
	EOM

	run_dodder show -format sku-metadata-sans-tai :z,t
	assert_success
	assert_output_unsorted - <<-EOM
		Type !md blake2b256-3kj7xgch6rjkq64aa36pnjtn9mdnl89k8pdhtlh33cjfpzy8ek4qnufx0m !toml-type-v1
		Zettel one/dos blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md tag-3 tag-4 "wow ok again"
		Zettel one/uno blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md tag-3 tag-4 "wow the first"
	EOM
}

function show_simple_type_one { # @test
	run_dodder show !md:t
	assert_output_unsorted - <<-EOM
		[!md @$(get_type_blob_sha) !toml-type-v1]
	EOM
}

function show_simple_type_one_history { # @test
	run_dodder show !md+t
	assert_output_unsorted - <<-EOM
		[!md @$(get_type_blob_sha) !toml-type-v1]
	EOM
}

function show_simple_type_tail { # @test
	run_dodder show :t
	assert_output_unsorted - <<-EOM
		[!md @$(get_type_blob_sha) !toml-type-v1]
	EOM
}

function show_simple_type_history { # @test
	run_dodder show +t
	assert_output_unsorted - <<-EOM
		[!md @$(get_type_blob_sha) !toml-type-v1]
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
		---
		! toml-config-v2
		---

		default-blob_store = '.default'

		[defaults]
		type = '!md'
		tags = []

		[file-extensions]
		config = 'konfig'
		conflict = 'conflict'
		lockfile = 'object-lockfile'
		organize = 'md'
		repo = 'repo'
		tag = 'tag'
		type = 'type'
		zettel = 'zettel'

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
		merge = ['vimdiff']
	EOM
}

function show_history_all { # @test
	run_dodder show +konfig,kasten,typ,etikett,zettel
	assert_output_unsorted - <<-EOM
		[konfig @$(get_konfig_sha) !toml-config-v2]
		[!md @$(get_type_blob_sha) !toml-type-v1]
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
	run_dodder show -format text !toml-type-v1:t
	assert_success
	assert_output - <<-EOM
		---
		! toml-type-v1
		---

		file-extension = 'md'
		vim-syntax-type = 'markdown'
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

	run_dodder init-workspace -query tag-5
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

	run_dodder init-workspace -query tag-3
	assert_success

	run_dodder show one/dos
	assert_success
	assert_output_unsorted - <<-EOM
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4 tag-5]
	EOM
}

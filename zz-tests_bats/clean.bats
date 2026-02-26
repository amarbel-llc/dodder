#! /usr/bin/env bats

setup() {
	load "$(dirname "$BATS_TEST_FILE")/common.bash"

	# for shellcheck SC2154
	export output

	copy_from_version "$DIR"
}

teardown() {
	chflags_and_rm
}

function prepare_checkouts() {
	run_dodder_init_workspace
	run_dodder checkout :z,t,e
	assert_success
	assert_output_unsorted - <<-EOM
		      checked out [md.type @blake2b256-3kj7xgch6rjkq64aa36pnjtn9mdnl89k8pdhtlh33cjfpzy8ek4qnufx0m !toml-type-v1]
		      checked out [alpha/hotel.zettel @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		      checked out [alpha/golf.zettel @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM
}

# bats file_tags=user_story:clean

# bats test_tags=user_story:workspace
function clean_fails_outside_workspace { # @test
	run_dodder clean .
	assert_failure
}

# bats file_tags=user_story:workspace

function clean_all { # @test
	prepare_checkouts
	run_dodder clean .
	assert_success
	assert_output_unsorted - <<-EOM
		          deleted [md.type]
		          deleted [alpha/]
		          deleted [alpha/hotel.zettel]
		          deleted [alpha/golf.zettel]
	EOM

	run_find
	assert_output '.'
}

function clean_zettels { # @test
	prepare_checkouts
	run_dodder clean .z
	assert_success
	assert_output_unsorted - <<-EOM
		          deleted [alpha/hotel.zettel]
		          deleted [alpha/golf.zettel]
		          deleted [alpha/]
	EOM

	run_find
	assert_success
	assert_output_unsorted - <<-EOM
		.
		./md.type
	EOM
}

function clean_all_dirty_wd { # @test
	prepare_checkouts
	cat >alpha/golf.zettel <<-EOM
		---
		# wildly different
		- etikett-one
		! md
		---

		newest body
	EOM

	cat >alpha/hotel.zettel <<-EOM
		---
		# dos wildly different
		- etikett-two
		! md
		---

		dos newest body
	EOM

	cat >md.type <<-EOM
		inline-akte = false
		vim-syntax-type = "test"
	EOM

	cat >da-new.type <<-EOM
		inline-akte = true
		vim-syntax-type = "da-new"
	EOM

	cat >zz-archive.tag <<-EOM
		hide = true
	EOM

	run_dodder clean .
	assert_success
	assert_output_unsorted - <<-EOM
	EOM

	run_find
	assert_success
	assert_output_unsorted - <<-EOM
		.
		./md.type
		./alpha
		./alpha/golf.zettel
		./alpha/hotel.zettel
		./da-new.type
		./zz-archive.tag
	EOM
}

function clean_all_force_dirty_wd { # @test
	prepare_checkouts
	cat >alpha/golf.zettel <<-EOM
		---
		# wildly different
		- etikett-one
		! md
		---

		newest body
	EOM

	cat >alpha/hotel.zettel <<-EOM
		---
		# dos wildly different
		- tag-two
		! md
		---

		dos newest body
	EOM

	cat >md.type <<-EOM
		inline-akte = false
		vim-syntax-type = "test"
	EOM

	cat >da-new.type <<-EOM
		inline-akte = true
		vim-syntax-type = "da-new"
	EOM

	cat >zz-archive.tag <<-EOM
		hide = true
	EOM

	run_dodder clean -force .
	assert_success
	assert_output_unsorted - <<-EOM
		          deleted [da-new.type]
		          deleted [md.type]
		          deleted [alpha/hotel.zettel]
		          deleted [alpha/golf.zettel]
		          deleted [alpha/]
		          deleted [zz-archive.tag]
	EOM

	run_find
	assert_success
	assert_output '.'
}

function clean_hidden { # @test
	prepare_checkouts
	run_dodder show alpha/golf
	assert_success
	assert_output - <<-EOM
		[alpha/golf @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM
	run_dodder organize -mode commit-directly :z <<-EOM
		- [alpha/golf  !md zz-archive tag-3 tag-4] wow the first
	EOM
	assert_success
	assert_output - <<-EOM
		[alpha/golf @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4 zz-archive]
	EOM

	run_dodder dormant-add zz-archive
	assert_success
	assert_output ''

	run_dodder show :z
	assert_success
	assert_output - <<-EOM
		[alpha/hotel @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
	EOM

	run_dodder show :?z
	assert_success
	assert_output_unsorted - <<-EOM
		[alpha/golf @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4 zz-archive]
		[alpha/hotel @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
	EOM

	run_dodder checkout -force alpha/golf
	assert_success
	assert_output - <<-EOM
		      checked out [alpha/golf.zettel @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4 zz-archive]
	EOM

	run_dodder clean !md.z
	assert_success
	assert_output_unsorted - <<-EOM
		          deleted [alpha/]
		          deleted [alpha/hotel.zettel]
		          deleted [alpha/golf.zettel]
	EOM
}

function clean_mode_blob_hidden { # @test
	prepare_checkouts
	run_dodder organize -mode commit-directly :z <<-EOM
		- [alpha/golf  !md zz-archive tag-3 tag-4] wow the first
	EOM
	assert_success
	assert_output - <<-EOM
		[alpha/golf @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4 zz-archive]
	EOM

	run_dodder dormant-add zz-archive
	assert_success
	assert_output ''

	run_dodder checkout -force -mode blob alpha/golf
	assert_success
	assert_output - <<-EOM
		      checked out [alpha/golf @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4 zz-archive
		                   alpha/golf.md]
	EOM

	run_dodder clean !md.z
	assert_success
	assert_output_unsorted - <<-EOM
		          deleted [alpha/golf.md]
		          deleted [alpha/hotel.zettel]
		          deleted [alpha/]
	EOM
}

function clean_mode_blob { # @test
	run_dodder_init_workspace
	run_dodder checkout -mode blob alpha/golf
	assert_success
	assert_output - <<-EOM
		      checked out [alpha/golf @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4
		                   alpha/golf.md]
	EOM

	run_dodder clean .
	assert_success
	assert_output_unsorted - <<-EOM
		          deleted [alpha/golf.md]
		          deleted [alpha/]
	EOM
}

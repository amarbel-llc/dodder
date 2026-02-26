#! /usr/bin/env bats

setup() {
	load "$(dirname "$BATS_TEST_FILE")/common.bash"
}

teardown() {
	chflags_and_rm
}

function add { # @test
	run_dodder_init_disable_age

	f=to_add.md
	{
		echo test file
	} >"$f"

	run_dodder add \
		-delete \
		-tags zz-inbox-2022-11-14 \
		"$f"

	assert_success
	assert_output - <<-EOM
		[one/uno @blake2b256-kzc0sp5p2ftddtjqgtusdwrsc33fs8h6xwdhp8shhgp0r92uln9q5mkl08 !md "to_add" zz-inbox-2022-11-14]
		          deleted [to_add.md]
	EOM

	run_dodder show -format text one/uno
	assert_success
	assert_output --regexp - <<-EOM
		---
		# to_add
		- zz-inbox-2022-11-14
		! md@.*
		---

		test file
	EOM
}

function add_with_dupe_added { # @test
	run_dodder_init_disable_age

	f=to_add.md
	{
		echo test file
	} >"$f"

	f2=to_add2.md
	{
		echo test file
	} >"$f2"

	run_dodder add \
		-delete \
		-tags zz-inbox-2022-11-14 \
		"$f" "$f2"

	assert_success
	assert_output_unsorted - <<-EOM
		          deleted [to_add.md]
		          deleted [to_add2.md]
		[one/uno @blake2b256-kzc0sp5p2ftddtjqgtusdwrsc33fs8h6xwdhp8shhgp0r92uln9q5mkl08 !md "to_add to_add2" zz-inbox-2022-11-14]
	EOM

	run_dodder show -format text one/uno
	assert_success
	assert_output --regexp - <<-EOM
		---
		# to_add to_add2
		- zz-inbox-2022-11-14
		! md@.*
		---

		test file
	EOM
}

function add_not_md { # @test
	run_dodder_init_disable_age

	f=to_add.pdf
	{
		echo test file
	} >"$f"

	run_dodder add \
		-delete \
		-tags zz-inbox-2022-11-14 \
		-each-blob "bash -c 'basename \$0'" \
		"$f"

	assert_success
	assert_output - <<-EOM
		[!pdf !toml-type-v1]
		[one/uno @blake2b256-kzc0sp5p2ftddtjqgtusdwrsc33fs8h6xwdhp8shhgp0r92uln9q5mkl08 !pdf "to_add" zz-inbox-2022-11-14]
		      checked out [one/uno @blake2b256-kzc0sp5p2ftddtjqgtusdwrsc33fs8h6xwdhp8shhgp0r92uln9q5mkl08 !pdf "to_add" zz-inbox-2022-11-14
		                   one/uno.pdf]
		uno.pdf
		          deleted [to_add.pdf]
	EOM

	run_dodder show -format text one/uno
	assert_success
	assert_output --regexp - <<-EOM
		---
		# to_add
		- zz-inbox-2022-11-14
		! pdf@.*
		---

		test file
	EOM
}

function add_1 { # @test
	run_dodder_init_disable_age

	f=to_add.md
	{
		echo test file
	} >"$f"

	run_dodder add \
		-delete \
		-tags zz-inbox-2022-11-14 \
		"$f"

	assert_success
	assert_output - <<-EOM
		[one/uno @blake2b256-kzc0sp5p2ftddtjqgtusdwrsc33fs8h6xwdhp8shhgp0r92uln9q5mkl08 !md "to_add" zz-inbox-2022-11-14]
		          deleted [to_add.md]
	EOM
}

function add_2 { # @test
	wd="$(mktemp -d)"
	cd "$wd" || exit 1

	run_dodder_init_disable_age
	assert_success

	f=to_add.md
	{
		echo test file
	} >"$f"

	f2=to_add2.md
	{
		echo test file 2
	} >"$f2"

	run_dodder add \
		-delete \
		-tags zz-inbox-2022-11-14 \
		"$f" "$f2"

	assert_success
	assert_output_unsorted - <<-EOM
		          deleted [to_add.md]
		          deleted [to_add2.md]
		[one/dos @blake2b256-exrtq04pdgc72mh7ufctgak45f085u6rzdfwlw0yhxrnxdfmnmqs6du6ya !md "to_add2" zz-inbox-2022-11-14]
		[one/uno @blake2b256-kzc0sp5p2ftddtjqgtusdwrsc33fs8h6xwdhp8shhgp0r92uln9q5mkl08 !md "to_add" zz-inbox-2022-11-14]
	EOM
}

function add_dot { # @test
	wd="$(mktemp -d)"
	cd "$wd" || exit 1

	run_dodder_init_disable_age
	assert_success

	f=to_add.md
	{
		echo test file
	} >"$f"

	f2=to_add2.md
	{
		echo test file 2
	} >"$f2"

	run_dodder add \
		-delete \
		-tags zz-inbox-2022-11-14 \
		.

	assert_success
	assert_output_unsorted - <<-EOM
		          deleted [to_add.md]
		          deleted [to_add2.md]
		[one/dos @blake2b256-exrtq04pdgc72mh7ufctgak45f085u6rzdfwlw0yhxrnxdfmnmqs6du6ya !md "to_add2" zz-inbox-2022-11-14]
		[one/uno @blake2b256-kzc0sp5p2ftddtjqgtusdwrsc33fs8h6xwdhp8shhgp0r92uln9q5mkl08 !md "to_add" zz-inbox-2022-11-14]
	EOM
}

#TODO-P2 add_dedupe: define expected behavior when re-adding a blob that already
# has a zettel. Current behavior: creates a new zettel (one/dos) inheriting
# metadata from the existing zettel, but -tags flag is silently ignored.
# Should -tags be merged? Should it reuse the existing zettel ID instead?

function add_several_with_spaces_in_filename { # @test
	run_dodder_init_disable_age

	f="to add.md"
	{
		echo test file
	} >"$f"

	f2="to add2.md"
	{
		echo test file
		echo two!!!!
	} >"$f2"

	run_dodder add \
		-delete \
		-tags zz-inbox-2022-11-14 \
		"$f" "$f2"

	assert_success
	assert_output_unsorted - <<-EOM
		          deleted [to add.md]
		          deleted [to add2.md]
		[one/dos @blake2b256-wg3yq6ymccg8mrm3zrsdym3u9dvyq6pfdt2d463e53nw9vxypxeqttduac !md "to add2" zz-inbox-2022-11-14]
		[one/uno @blake2b256-kzc0sp5p2ftddtjqgtusdwrsc33fs8h6xwdhp8shhgp0r92uln9q5mkl08 !md "to add" zz-inbox-2022-11-14]
	EOM

	run_dodder show -format text one/uno
	assert_success
	assert_output --regexp - <<-EOM
		---
		# to add
		- zz-inbox-2022-11-14
		! md@.*
		---

		test file
	EOM
}

function add_each_blob { # @test
	run_dodder_init_disable_age

	f="to add.md"
	{
		echo test file
	} >"$f"

	run_dodder add \
		-each-blob "cat" \
		-delete \
		-tags zz-inbox-2022-11-14 \
		"$f"

	assert_success
	assert_output_unsorted - <<-EOM
		                   one/uno.md]
		          deleted [to add.md]
		      checked out [one/uno @blake2b256-kzc0sp5p2ftddtjqgtusdwrsc33fs8h6xwdhp8shhgp0r92uln9q5mkl08 !md "to add" zz-inbox-2022-11-14
		[one/uno @blake2b256-kzc0sp5p2ftddtjqgtusdwrsc33fs8h6xwdhp8shhgp0r92uln9q5mkl08 !md "to add" zz-inbox-2022-11-14]
		test file
	EOM
}

function add_organize { # @test
	run_dodder_init_disable_age

	function editor() {
		# shellcheck disable=SC2317
		cp "$1" organize.md
	}

	export -f editor

	# shellcheck disable=SC2016
	export EDITOR='bash -c "editor $0"'

	f="to add.md"
	{
		echo test file
	} >"$f"

	run_dodder add \
		-each-blob "cat" \
		-organize \
		-delete \
		-tags zz-inbox-2022-11-14 \
		"$f"

	assert_success
	assert_output - <<-EOM
		[one/uno @blake2b256-kzc0sp5p2ftddtjqgtusdwrsc33fs8h6xwdhp8shhgp0r92uln9q5mkl08 !md "to add" zz-inbox-2022-11-14]
		      checked out [one/uno @blake2b256-kzc0sp5p2ftddtjqgtusdwrsc33fs8h6xwdhp8shhgp0r92uln9q5mkl08 !md "to add" zz-inbox-2022-11-14
		                   one/uno.md]
		test file
		          deleted [to add.md]
	EOM

	run cat organize.md
	assert_success
	assert_output - <<-EOM
		---
		% instructions: to prevent an object from being checked in, delete it entirely
		% delete:true delete once checked in
		- zz-inbox-2022-11-14
		---

		- ["to add.md"]
	EOM
}

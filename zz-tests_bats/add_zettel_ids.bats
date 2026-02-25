#! /usr/bin/env bats

setup() {
	load "$(dirname "$BATS_TEST_FILE")/common.bash"

	# for shellcheck SC2154
	export output

	copy_from_version "$DIR"

	run_dodder_init_workspace

	# create the object id log via migration
	run_dodder migrate-zettel-ids
	assert_success
}

teardown() {
	chflags_and_rm
}

function add_zettel_ids_yin_success { # @test
	input="$(mktemp)"
	{
		echo "a sentence about ceroplastes"
		echo "another line about midtown"
		echo "something about harbor"
	} >"$input"

	run_dodder add-zettel-ids-yin <"$input"
	assert_success
	assert_output --partial "added 3 words to Yin"
	assert_output --partial "pool size: 54"
}

function add_zettel_ids_yin_dedup { # @test
	input="$(mktemp)"
	{
		echo "a sentence about ceroplastes"
		echo "another line about midtown"
		echo "something about harbor"
	} >"$input"

	run_dodder add-zettel-ids-yin <"$input"
	assert_success
	assert_output --partial "added 3 words"

	# same input again should be a no-op since words already exist
	run_dodder add-zettel-ids-yin <"$input"
	assert_success
	assert_output --partial "no new words to add"
}

function add_zettel_ids_yin_cross_side_rejection { # @test
	input="$(mktemp)"
	{
		echo "something about quatro"
		echo "another about newword"
	} >"$input"

	# quatro is in Yang, should be rejected; newword is new
	run_dodder add-zettel-ids-yin <"$input"
	assert_success
	assert_output --partial "added 1 words to Yin"
	assert_output --partial "pool size: 42"
}

function add_zettel_ids_yin_no_new_words { # @test
	input="$(mktemp)"
	{
		echo "something about three"
		echo "another about four"
	} >"$input"

	run_dodder add-zettel-ids-yin <"$input"
	assert_success
	assert_output --partial "no new words to add"
}

function add_zettel_ids_yang_success { # @test
	input="$(mktemp)"
	{
		echo "a sentence about ceroplastes"
		echo "another line about midtown"
		echo "something about harbor"
	} >"$input"

	run_dodder add-zettel-ids-yang <"$input"
	assert_success
	assert_output --partial "added 3 words to Yang"
	assert_output --partial "pool size: 54"
}

function add_zettel_ids_yang_cross_side_rejection { # @test
	input="$(mktemp)"
	{
		echo "something about three"
		echo "another about newword"
	} >"$input"

	# three is in Yin, should be rejected; newword is new
	run_dodder add-zettel-ids-yang <"$input"
	assert_success
	assert_output --partial "added 1 words to Yang"
	assert_output --partial "pool size: 42"
}

function add_zettel_ids_peek_shows_larger_pool_after_reindex { # @test
	input="$(mktemp)"
	{
		echo "a sentence about ceroplastes"
		echo "another line about midtown"
		echo "something about harbor"
	} >"$input"

	run_dodder add-zettel-ids-yin <"$input"
	assert_success

	# reindex rebuilds the zettel ID availability index from flat files
	run_dodder reindex
	assert_success

	run_dodder peek-zettel-ids 100
	assert_success

	# 9 yin words x 6 yang words = 54 possible, minus some used
	# original was 6x6=36, so available count should be higher
	after_count="$(echo "$output" | wc -l)"
	[[ "$after_count" -gt 34 ]]
}

function add_zettel_ids_new_still_works { # @test
	input="$(mktemp)"
	{
		echo "a sentence about ceroplastes"
		echo "another line about midtown"
		echo "something about harbor"
	} >"$input"

	run_dodder add-zettel-ids-yin <"$input"
	assert_success

	run_dodder new -edit=false
	assert_success
	assert_output --regexp '\[.+/.+ !md\]'
}

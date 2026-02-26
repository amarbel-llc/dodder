#! /usr/bin/env bats

cat_yin_nato() {
	cat <<'EOM'
the alpha
the bravo
the charlie
the delta
an echo
the foxtrot
EOM
}

cat_yang_nato() {
	cat <<'EOM'
the golf
the hotel
the india
the juliet
the kilo
the lima
EOM
}

setup() {
	load "$(dirname "$BATS_TEST_FILE")/common.bash"

	# for shellcheck SC2154
	export output

	run_dodder init \
		-yin <(cat_yin_nato) \
		-yang <(cat_yang_nato) \
		-lock-internal-files=false \
		-override-xdg-with-cwd \
		test

	assert_success

	run_dodder_init_workspace
}

teardown() {
	chflags_and_rm
}

function add_zettel_ids_yin_success { # @test
	run_dodder add-zettel-ids-yin <<'EOM'
a sentence about ceroplastes
another line about midtown
something about harbor
EOM
	assert_success
	assert_output "added 3 words to Yin (pool size: 54)"
}

function add_zettel_ids_yin_dedup { # @test
	input="$(mktemp)"
	cat >"$input" <<'EOM'
a sentence about ceroplastes
another line about midtown
something about harbor
EOM

	run_dodder add-zettel-ids-yin <"$input"
	assert_success
	assert_output "added 3 words to Yin (pool size: 54)"

	# same input again should be a no-op since words already exist
	run_dodder add-zettel-ids-yin <"$input"
	assert_success
	assert_output "no new words to add"
}

function add_zettel_ids_yin_cross_side_rejection { # @test
	# golf is in Yang, should be rejected; newword is new
	run_dodder add-zettel-ids-yin <<'EOM'
something about golf
another about newword
EOM
	assert_success
	assert_output "added 1 words to Yin (pool size: 42)"
}

function add_zettel_ids_yin_no_new_words { # @test
	run_dodder add-zettel-ids-yin <<'EOM'
something about alpha
another about bravo
EOM
	assert_success
	assert_output "no new words to add"
}

function add_zettel_ids_yang_success { # @test
	run_dodder add-zettel-ids-yang <<'EOM'
a sentence about ceroplastes
another line about midtown
something about harbor
EOM
	assert_success
	assert_output "added 3 words to Yang (pool size: 54)"
}

function add_zettel_ids_yang_cross_side_rejection { # @test
	# alpha is in Yin, should be rejected; newword is new
	run_dodder add-zettel-ids-yang <<'EOM'
something about alpha
another about newword
EOM
	assert_success
	assert_output "added 1 words to Yang (pool size: 42)"
}

function add_zettel_ids_peek_shows_larger_pool_after_reindex { # @test
	run_dodder add-zettel-ids-yin <<'EOM'
a sentence about ceroplastes
another line about midtown
something about harbor
EOM
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
	run_dodder add-zettel-ids-yin <<'EOM'
a sentence about ceroplastes
another line about midtown
something about harbor
EOM
	assert_success

	run_dodder new -edit=false
	assert_success
	assert_output --regexp '\[.+/.+ !md\]'
}

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

function migrate_zettel_ids_success { # @test
	run_dodder migrate-zettel-ids
	assert_success
	assert_output --partial "migrated zettel ids"
}

function migrate_zettel_ids_new_still_works { # @test
	run_dodder migrate-zettel-ids
	assert_success

	run_dodder new -edit=false
	assert_success
	assert_output --regexp '\[.+/.+ !md\]'
}

function migrate_zettel_ids_idempotent { # @test
	run_dodder migrate-zettel-ids
	assert_success
	assert_output --partial "migrated zettel ids"

	run_dodder migrate-zettel-ids
	assert_success
	assert_output --partial "already contains entries"
}

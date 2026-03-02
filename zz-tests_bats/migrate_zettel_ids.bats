#! /usr/bin/env bats

setup() {
	load "$(dirname "$BATS_TEST_FILE")/common.bash"

	# for shellcheck SC2154
	export output
}

teardown() {
	chflags_and_rm
}

# bats file_tags=user_story:migration

function migrate_zettel_ids { # @test
	wd="$(mktemp -d)"
	cd "$wd" || exit 1

	run_dodder_init_disable_age

	run_dodder migrate-zettel-ids
	assert_success
	assert_output --partial "migrated Yin"
	assert_output --partial "migrated Yang"
}

function migrate_zettel_ids_idempotent { # @test
	wd="$(mktemp -d)"
	cd "$wd" || exit 1

	run_dodder_init_disable_age

	run_dodder migrate-zettel-ids
	assert_success

	run_dodder migrate-zettel-ids
	assert_success
	assert_output --partial "already contains entries"
}

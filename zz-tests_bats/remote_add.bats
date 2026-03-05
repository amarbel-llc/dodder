#! /usr/bin/env bats

setup() {
	load "$(dirname "$BATS_TEST_FILE")/common.bash"

	# for shellcheck SC2154
	export output

	copy_from_version "$DIR"
	run_dodder_init_workspace
}

teardown() {
	chflags_nouchg
}

# bats file_tags=user_story:remote

function remote_add_dotenv_xdg { # @test
	mkdir -p them
	pushd them || exit 1
	run_dodder_init
	popd || exit 1

	run_dodder \
		remote-add \
		toml-repo-local_override_path-v0 \
		"$(realpath them)" \
		test-repo-id-them

	assert_success
	assert_output_unsorted --regexp - <<-'EOM'
		\[/test-repo-id-them @blake2b256-.+ !toml-repo-local_override_path-v0]
	EOM

	run_dodder show /test-repo-id-them:k
	assert_success
	assert_output_unsorted --regexp - <<-'EOM'
		\[/test-repo-id-them @blake2b256-.+ !toml-repo-local_override_path-v0]
	EOM

	run_dodder show -format text /test-repo-id-them:k
	assert_success
	assert_output --regexp - <<-'EOM'
		---
		! toml-repo-local_override_path-v0
		---

		public-key = 'dodder-repo-public_key-v1.*'
		override-path = '.+/them'
	EOM
}

function remote_add_local_path { # @test
	{
		mkdir -p them
		pushd them || exit 1
		run_dodder_init -override-xdg-with-cwd test-repo-remote
		popd || exit 1
	}

	run_dodder \
		remote-add \
		toml-repo-local_override_path-v0 \
		"$(realpath them)" \
		test-repo-id-them

	assert_success
	assert_output_unsorted --regexp - <<-'EOM'
		\[/test-repo-id-them @blake2b256-.+ !toml-repo-local_override_path-v0]
	EOM

	run_dodder show /test-repo-id-them:k
	assert_success
	assert_output_unsorted --regexp - <<-'EOM'
		\[/test-repo-id-them @blake2b256-.+ !toml-repo-local_override_path-v0]
	EOM

	run_dodder show -format text /test-repo-id-them:k
	assert_success
	assert_output --regexp - <<-'EOM'
		---
		! toml-repo-local_override_path-v0
		---

		public-key = 'dodder-repo-public_key-v1.*'
		override-path = '.+/them'
	EOM
}

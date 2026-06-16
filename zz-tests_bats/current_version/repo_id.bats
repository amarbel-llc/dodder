#! /usr/bin/env bats

setup() {
	load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

	# for shellcheck SC2154
	export output
}

teardown() {
	chflags_nouchg
}

# bats test_tags=repo_id
function repo_id_cwd_selects_cwd_repo { # @test
	run_dodder_init test-repo-id

	run test -d ".dodder"
	assert_success
}

# bats test_tags=repo_id
function repo_id_cwd_errors_when_no_repo_exists { # @test
	run_dodder show -repo_id .default
	assert_failure
}

# bats test_tags=repo_id
function repo_id_system_rejected_not_yet_resolvable { # @test
	run_dodder_stderr_unified init \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-repo_id / \
		test-repo-id

	assert_failure
	assert_output --regexp 'system scope is not yet resolvable'
}

# bats test_tags=repo_id
function dodder_repo_id_env_var_selects_cwd_repo { # @test
	DODDER_REPO_ID=.default run_dodder init \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		test-repo-id

	assert_success

	run test -d ".dodder"
	assert_success
}

# bats test_tags=repo_id
function repo_id_flag_overrides_env_var { # @test
	DODDER_REPO_ID=/ run_dodder init \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-repo_id .default \
		test-repo-id

	assert_success

	run test -d ".dodder"
	assert_success
}

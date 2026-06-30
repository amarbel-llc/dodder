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
	run_dodder_init

	run test -d ".dodder"
	assert_success
}

# bats test_tags=repo_id
function repo_id_cwd_errors_when_no_repo_exists { # @test
	run_dodder show -repo_id .default
	assert_failure
}

# bats test_tags=repo_id
function repo_id_remote_first_rejected_no_transport { # @test
	# `/name` is the remote-first system spelling (consult the repo's
	# remotes, fall back to the system-scoped name). dodder has no remote
	# transport and can't tell whether `name` is a defined remote before
	# opening, so it rejects rather than silently treating it as the system
	# repo. The forced-system `//name` spelling resolves (#280) — see
	# info_repo's system-scope test.
	run_dodder_stderr_unified init \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		/backup

	assert_failure
	assert_output --regexp 'remote-first.*not yet resolvable'
}

# bats test_tags=repo_id
function dodder_repo_id_env_var_rejected_on_init { # @test
	# FDR-0021 T3-C: DODDER_REPO_ID addresses an EXISTING repo; it no longer
	# names a new one on init. Setting it (making config.RepoId non-auto) is
	# rejected, pointing the caller at the location positional. The location
	# is named by the required positional instead.
	DODDER_REPO_ID=.default run_dodder init \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		.default

	assert_failure
	assert_output --regexp 'cannot name a new one'
}

# bats test_tags=repo_id
function init_rejects_non_auto_repo_id { # @test
	# FDR-0021 T3-C: -repo_id addresses an existing repo and cannot name a new
	# one on init. Supplying it alongside the required location positional is
	# rejected.
	run_dodder init \
		-yin <(cat_yin) \
		-yang <(cat_yang) \
		-repo_id other \
		.default

	assert_failure
	assert_output --regexp 'cannot name a new one'
}

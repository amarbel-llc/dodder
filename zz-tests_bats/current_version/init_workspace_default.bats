#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

  # for shellcheck SC2154
  export output
}

teardown() {
  chflags_nouchg
}

# bats file_tags=init

# init-workspace-default creates a workspace (over an init-default repo)
# whose external-item scan skips the default non-dot dirs (node_modules,
# target, dist, build, vendor). A file under node_modules/ is excluded
# from `status`; a normal top-level file is reported untracked. The blob
# digest is over the file content (deterministic).
function init_workspace_default_ignores_default_dirs { # @test
  run_dodder init-default test-iwd-repo
  assert_success

  run_dodder init-workspace-default
  assert_success

  mkdir -p node_modules
  cat >node_modules/dep.md <<-EOM
		ignored dependency
	EOM
  cat >keep.md <<-EOM
		a normal note
	EOM

  run_dodder status .
  assert_success
  assert_output '        untracked [keep.md @blake2b256-j2fcqsg8an9w7kxe03dsuhack393syupxwquqxeypevvgkvcu6nst7ww5n]'
}

# A second init-workspace-default is a no-op (a workspace already
# exists), so the bootstrap is safe to re-run.
function init_workspace_default_idempotent { # @test
  run_dodder init-default test-iwd-repo
  assert_success

  run_dodder init-workspace-default
  assert_success

  run_dodder init-workspace-default
  assert_success
  assert_output ''
}

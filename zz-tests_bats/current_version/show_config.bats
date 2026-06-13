setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"
  export output
}

# bats file_tags=user_story:config

# Covers `show-config`, the read surface over the repo-local config log
# (config_log package, FDR 0020). The functional path requires a
# populated config log, which only exists after init seeds the root
# entry (Task 6) or after migration converts the old konfig history
# (Task 9). Until then there is nothing to assert against, so the one
# test below is skipped.

function show_config_head { # @test
  skip # TODO(Task 6): unskip once init seeds the config log
}

# Drives the full write/read round-trip through the new surfaces: a fresh
# repo (no init-seeded config log yet), one real `edit-config`, then
# `show-config` streams the edited blob and `show-config -history` lists
# the single appended entry. Reuses edit_config.bats's $EDITOR-script
# mechanism for the non-interactive edit. Signatures are
# non-deterministic on a fresh key, so the history line is matched with a
# regexp; the streamed blob is the raw TOML so its appended marker line is
# asserted exactly.
function show_config_after_edit_config_roundtrips { # @test
  mkdir -p "$BATS_TEST_TMPDIR/fresh"
  cd "$BATS_TEST_TMPDIR/fresh"
  run_dodder_init_disable_age test-show-config

  export EDITOR="bash -c 'echo \"# this is the body 2\" >> \"\$0\"'"
  run_dodder edit-config
  assert_success

  run_dodder show-config
  assert_success
  assert_line '# this is the body 2'

  run_dodder show-config -history
  assert_success
  assert_output --regexp '^\[konfig @blake2b256-[a-z0-9]+ [0-9.]+ dodder-repo-public_key-v1@ed25519_pub-[a-z0-9]+ dodder-object-sig-v2@ed25519_sig-[a-z0-9]+ !toml-config-v2\]$'
}

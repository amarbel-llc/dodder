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

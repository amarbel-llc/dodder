# TODO

- [ ] Debug `just test-bats-update-fixtures` failure: bats succeeds when run directly from `zz-tests_bats/` but fails when invoked through the justfile recipe chain. Likely a working directory or environment variable propagation issue. The `cp` command can't find `.dodder` in the bats temp dir, suggesting the fixture generation test silently fails or the temp dir path extraction breaks.

#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

  # for shellcheck SC2154
  export output
}

teardown() {
  chflags_nouchg
}

# bats file_tags=user_story:zettel_ids

function add_zettel_ids_yin { # @test
  wd="$(mktemp -d)"
  cd "$wd" || exit 1

  run_dodder_init_disable_age

  run_dodder migrate-zettel-ids
  assert_success

  run bash -c 'echo -e "alpha\nbravo\ncharlie" | '"$DODDER_BIN"' add-zettel-ids-yin'
  assert_success
  assert_output --partial "added 3 words"
  assert_output --partial "pool size"
}

function add_zettel_ids_yang { # @test
  wd="$(mktemp -d)"
  cd "$wd" || exit 1

  run_dodder_init_disable_age

  run_dodder migrate-zettel-ids
  assert_success

  run bash -c 'echo -e "golf\nhotel\nindia" | '"$DODDER_BIN"' add-zettel-ids-yang'
  assert_success
  assert_output --partial "added 3 words"
}

function add_zettel_ids_dedup { # @test
  wd="$(mktemp -d)"
  cd "$wd" || exit 1

  run_dodder_init_disable_age

  run_dodder migrate-zettel-ids
  assert_success

  run bash -c 'echo -e "alpha\nbravo" | '"$DODDER_BIN"' add-zettel-ids-yin '
  assert_success

  run bash -c 'echo -e "alpha\ncharlie" | '"$DODDER_BIN"' add-zettel-ids-yin '
  assert_success
  assert_output --partial "added 1 words"
}

function add_zettel_ids_cross_side_rejection { # @test
  wd="$(mktemp -d)"
  cd "$wd" || exit 1

  run_dodder_init_disable_age

  run_dodder migrate-zettel-ids
  assert_success

  run bash -c 'echo -e "alpha" | '"$DODDER_BIN"' add-zettel-ids-yin '
  assert_success

  # alpha is already in yin, should be rejected from yang
  run bash -c 'echo -e "alpha" | '"$DODDER_BIN"' add-zettel-ids-yang '
  assert_success
  assert_output --partial "no new words"
}

function add_zettel_ids_bootstrap_from_nothing { # @test
  wd="$(mktemp -d)"
  cd "$wd" || exit 1

  # init WITHOUT -yin/-yang: no zettel-id pool is seeded, so the
  # object_ids/Yin and object_ids/Yang flat files do not exist.
  run_dodder init -encryption none .default
  assert_success

  run_dodder init-workspace -experimental-repo=false
  assert_success

  # yin side: must bootstrap the pool from nothing (no flat files yet).
  # yang count is still 0, so pool size is 0.
  run bash -c 'echo -e "alpha\nbravo\ncharlie" | '"$DODDER_BIN"' add-zettel-ids-yin'
  assert_success
  assert_output "added 3 words to Yin (pool size: 0)"

  # yang side completes the pool: 3 yin * 3 yang = 9.
  run bash -c 'echo -e "golf\nhotel\nindia" | '"$DODDER_BIN"' add-zettel-ids-yang'
  assert_success
  assert_output "added 3 words to Yang (pool size: 9)"
}

function add_zettel_ids_no_new_words { # @test
  wd="$(mktemp -d)"
  cd "$wd" || exit 1

  run_dodder_init_disable_age

  run_dodder migrate-zettel-ids
  assert_success

  # "one" is already in the yin provider from init
  run bash -c 'echo -e "one" | '"$DODDER_BIN"' add-zettel-ids-yin '
  assert_success
  assert_output --partial "no new words"
}

#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"
  load "$(dirname "$BATS_TEST_FILE")/../lib/sftp.bash"
  export output

  set_xdg "$BATS_TEST_TMPDIR"
  start_sftp_server
}

teardown() {
  chflags_nouchg
  stop_sftp_server
}

function sftp_single_hash_remote_blake2b256_blob_unreadable { # @test
  # Reproduces a real-world mystery: a repo genesis'd with
  # -hash_type-id blake2b256 against an SFTP blob store whose remote
  # is legacy single-hash sha256 (as a real rsync.net-backed store,
  # set up before madder's multi-hash SFTP support existed, actually
  # is). amarbel-llc/madder#261/#262 fixed MakeBlobWriter silently
  # substituting the wrong hash type on write -- but re-verifying
  # against the fixed madder showed init still succeeds silently here,
  # and the written blob is unreadable afterward via the same digest.
  # Root cause not yet confirmed; this test pins the current (buggy)
  # behavior end-to-end so a real fix has something concrete to turn
  # green, and so the mechanism can be iterated on directly instead of
  # via a live personal repo.

  # Harvest a real sha256 blob_store-config by creating a plain local
  # store with that hash type, rather than hand-writing the whole
  # TOML: init-sftp-explicit's fresh-remote path hardcodes blake2b256
  # + multi-hash=true (madder#149's fix), and there is no CLI flag for
  # -hash_type-id sha256 -single-hash either -- TomlV3's SingleHash
  # field (single_hash in TOML) has no flag at all, it's only ever set
  # by the sftp-analyze-and-suggest-configs discover tooling. Append
  # it by hand onto the harvested config to reproduce a legacy
  # single-hash remote like the real rsync.net-backed store.
  run_madder init -hash_type-id sha256 -encryption none harvest-sha256
  assert_success

  local remote_root="$BATS_TEST_TMPDIR/sftp-remote-legacy"
  mkdir -p "$remote_root"
  # Strip the embedded `@ <digest>` metadata line rather than
  # recomputing it: DecodeAndVerify (madder's delta/blob_store_configs
  # package) trusts a config with no `@` line silently (pre-FDR-0008
  # back-compat), which is the simplest way to hand-edit a harvested
  # config without needing madder's internal digest/purpose-tag
  # construction.
  grep -v '^@ ' "$XDG_DATA_HOME/madder/blob_stores/harvest-sha256/blob_store-config" \
    >"$remote_root/blob_store-config"
  echo 'single_hash = true' >>"$remote_root/blob_store-config"

  # init-sftp-explicit detects the pre-existing remote config and
  # adopts it instead of writing a fresh (multi-hash) one -- mirrors
  # how a real pre-existing remote (rsync.net, in the original report)
  # gets linked in without dodder ever choosing its hash type.
  run_madder init-sftp-explicit \
    -host 127.0.0.1 \
    -port "$SFTP_PORT" \
    -user testuser \
    -password anything \
    -remote-path "$remote_root" \
    -known-hosts-file "$SFTP_KNOWN_HOSTS" \
    rsync-mimic
  assert_success
  assert_output --partial 'remote blob store config already present'

  run_madder info-repo rsync-mimic hash_type-id
  assert_success
  assert_line 'sha256'

  run_madder info-repo rsync-mimic supports-multi-hash
  assert_success
  assert_line 'false'

  # The actual repro: genesis a dodder repo requesting blake2b256
  # against this single-hash sha256 remote.
  run_dodder init \
    -yin-default -yang-default \
    -hash_type-id blake2b256 \
    -blob_store-id rsync-mimic \
    legacy-single-hash-repo
  assert_success

  # Deterministic across fresh inits (content-addressed, no
  # repo-specific data) -- pinning the exact digest so a future fix
  # (or an unrelated change to the pandoc defaults bundle) surfaces as
  # a clear diff rather than a silently-passing test.
  local pandoc_defaults_digest='blake2b256-zcfmrghzp36r4r4qxtrh4t8xcd5g0f3mkpm8f3swac0vr5x503msyfsu3d'

  run_dodder cat-object -repo_id legacy-single-hash-repo "$pandoc_defaults_digest"
  assert_failure
  assert_output --partial 'not found'
}

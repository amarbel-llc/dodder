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

function sftp_single_hash_remote_blob_store_id_pins_default_store { # @test
  # Regression test for amarbel-llc/dodder#365. Traced through several
  # wrong theories before landing on the real mechanism -- see the
  # issue thread for the full history (hash-type mismatch, then a
  # theorized SFTP mover rename bug, both ruled out by direct
  # instrumentation). The actual bug: env_repo.Genesis() validates
  # that -blob_store-id's named store exists but never pins it as the
  # *default* store via madder's BlobStoreEnv.SetBlobStoreOrder.
  # Without that, madder's own default-selection (an alphabetical sort
  # of every blob store discovered in the XDG scope) silently wins, so
  # genesis writes land in whatever store happens to sort first rather
  # than the one actually named.
  #
  # This test reproduces that precisely: two stores coexist in the
  # same XDG namespace -- "harvest-sha256" (created first, purely to
  # harvest a config template) and "rsync-mimic" (an SFTP store, the
  # one actually passed via -blob_store-id). "harvest-sha256" sorts
  # before "rsync-mimic" alphabetically, so pre-fix it silently won
  # and absorbed every genesis write; post-fix, everything must land
  # in rsync-mimic specifically, verified by direct filesystem
  # inspection of both stores.
  #
  # Deliberately stops at write-routing and does NOT attempt to read
  # the repo back (no `dodder last`/`cat-object`): a repo whose only
  # configured blob store is remote can't currently read its own
  # bootstrap config back at all (config-bootstrap reads are
  # restricted to local-typed stores by design -- see
  # store_config/persist.go's loadMutableConfigBlob comment). That is
  # a separate, much larger, already-tracked gap: amarbel-llc/dodder#223
  # (FDR-0016 D1, multi-default with a local write-store). Not this
  # test's concern -- see #223 for a concrete repro of that gap.
  #
  # The SFTP flavor here isn't essential to the #365 bug -- it's what
  # the original real-world report happened to hit -- but it's kept as
  # the reproduction shape since it's already fully worked out, matches
  # the original incident precisely, and exercises the single-hash
  # remote-store path along the way.

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

  # The actual repro: genesis a dodder repo against the single-hash
  # sha256 remote, with "harvest-sha256" (alphabetically earlier)
  # still present as a competing store in the same XDG namespace.
  run_dodder init \
    -yin-default -yang-default \
    -hash_type-id blake2b256 \
    -blob_store-id rsync-mimic \
    legacy-single-hash-repo
  assert_success

  # Decisive check #1: harvest-sha256 (alphabetically earlier) must
  # receive NO genesis writes -- pre-fix it silently absorbed
  # everything via madder's alphabetical default-store sort. Its blob
  # store directory should contain only its own blob_store-config, no
  # hash-bucket subdirectories.
  run find "$XDG_DATA_HOME/madder/blob_stores/harvest-sha256" \
    -mindepth 1 -not -name 'blob_store-config'
  assert_success
  assert_output ""

  # Decisive check #2: genesis writes DID land in rsync-mimic (the
  # store actually named via -blob_store-id), not nowhere -- real
  # committed blob content beyond the remote's own blob_store-config
  # and any leaked SFTP mover temp files (amarbel-llc/dodder#366).
  run find "$remote_root" -mindepth 1 -type f \
    -not -name 'blob_store-config' -not -name 'tmp_*'
  assert_success
  refute_output ""
}

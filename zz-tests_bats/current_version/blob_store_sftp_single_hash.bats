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
  # Also proves #223 (FDR-0016 D1) is fixed: genesis now always wraps
  # -blob_store-id in a write_through multi whose write-store is the
  # shared local store and whose read-stores are the caller-named
  # store(s), so the bootstrap config read (store_config/persist.go's
  # loadMutableConfigBlob, restricted to local-typed stores by design)
  # can always find the local write-store -- it never needs to dial the
  # remote just to read the repo's own config back. See sibling test
  # sftp_single_hash_remote_blob_store_id_read_back_works below for the
  # decisive read-back check.
  #
  # write_through (rather than mirror) is FDR-0016 D1's originally
  # written design; it also has no cross-member hash-type-agreement
  # requirement (unlike mirror, madder#268), so a legacy single-hash
  # remote whose native hash type differs from the local default (as
  # here: sha256 vs. blake2b256) is fully supported. The tradeoff:
  # genesis writes land ONLY in the local write-store -- rsync-mimic,
  # as a read-only fallback, receives no new content from genesis.
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

  # Decisive check #2: rsync-mimic (the store named via -blob_store-id)
  # must receive NO genesis writes either -- under the write_through
  # design (FDR-0016 D1), a caller-named remote is a read-only
  # fallback, not a mirror target; only the shared local write-store
  # receives new content. No committed blob content beyond the
  # remote's own blob_store-config and any leaked SFTP mover temp
  # files (amarbel-llc/dodder#366).
  run find "$remote_root" -mindepth 1 -type f \
    -not -name 'blob_store-config' -not -name 'tmp_*'
  assert_success
  assert_output ""

  # Decisive check #3: genesis writes DID land somewhere real -- the
  # shared local write-store ("default-local"), not nowhere and not
  # either of the two competing stores above.
  run find "$XDG_DATA_HOME/madder/blob_stores/default-local" \
    -mindepth 1 -not -name 'blob_store-config'
  assert_success
  refute_output ""
}

function sftp_single_hash_remote_blob_store_id_read_back_works { # @test
  # The decisive #223 (FDR-0016 D1) check: a repo genesis'd with
  # -blob_store-id pointing at a remote-only store must be able to read
  # its own bootstrap config back, with no local fallback store
  # configured by the caller. Before #223 landed, this failed outright --
  # store_config/persist.go's loadMutableConfigBlob only reads via
  # GetLocalReadBlobStore(), which categorically excludes remote-typed
  # stores, and the repo's only configured store WAS the remote. Setup
  # mirrors sftp_single_hash_remote_blob_store_id_pins_default_store
  # above (a single-hash sha256 SFTP remote), minus the competing
  # harvest-sha256 store, which isn't needed for this check.
  run_madder init -hash_type-id sha256 -encryption none harvest-sha256
  assert_success

  local remote_root="$BATS_TEST_TMPDIR/sftp-remote-legacy"
  mkdir -p "$remote_root"
  grep -v '^@ ' "$XDG_DATA_HOME/madder/blob_stores/harvest-sha256/blob_store-config" \
    >"$remote_root/blob_store-config"
  echo 'single_hash = true' >>"$remote_root/blob_store-config"

  run_madder init-sftp-explicit \
    -host 127.0.0.1 \
    -port "$SFTP_PORT" \
    -user testuser \
    -password anything \
    -remote-path "$remote_root" \
    -known-hosts-file "$SFTP_KNOWN_HOSTS" \
    rsync-mimic-readback
  assert_success

  # CWD-scoped, and specifically named "default": a bare (XDG
  # user-scoped) repo-id has no CWD binding, so a later bare `dodder
  # last`/`show` (no -repo_id) can't auto-discover it. Every bare
  # subsequent-command example across the whole bats suite uses
  # `.default` specifically -- bare commands assume that name for
  # CWD-scoped discovery rather than reading back whatever name init
  # was actually given, so this test follows the same convention
  # instead of introducing untested territory.
  run_dodder init \
    -yin-default -yang-default \
    -hash_type-id blake2b256 \
    -blob_store-id rsync-mimic-readback \
    .default
  assert_success

  # The read-back itself: `last` decodes the config it just bootstrapped
  # from (persist.go's loadMutableConfigBlob), and `show` proves an
  # actual object round-trips through the store, not just an in-memory
  # default. Neither dials the remote -- both are served by the multi's
  # local write-store.
  run_dodder last
  assert_success

  run_dodder show :z
  assert_success
}

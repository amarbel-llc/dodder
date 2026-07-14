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
  # substituting the wrong hash type on write -- re-verifying against
  # the fixed madder showed init still succeeds silently here, and
  # reads still fail afterward.
  #
  # Instrumenting this test revealed the real shape of the bug: it is
  # NOT a hash-type mismatch. Genesis's blob-write calls
  # (writeRawBlob / SaveBlobText in
  # go/internal/romeo/local_working_copy/genesis_pandoc_tools.go and
  # golf/type_blobs/coder.go) pass a nil requested hash type, which
  # resolveWriteHashFormat (madder's fix) resolves to the store's own
  # default silently and correctly -- no mismatch, no error. `last`
  # fails trying to read genesis's own inventory-list-log entry, and
  # that entry's digest IS sha256 (matching the single-hash remote's
  # native format exactly). So a *correctly* sha256-digested blob,
  # written to a store natively configured for sha256, still cannot be
  # read back afterward. Root cause is still open below that point --
  # this test pins the current (buggy) behavior with direct filesystem
  # instrumentation so it can be iterated on further without needing a
  # live personal repo.

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

  # Capture the digest genesis ACTUALLY assigned, rather than trusting
  # a hardcoded value from an earlier report: a stale/wrong hardcoded
  # digest would make the cat-object check below pass trivially (any
  # nonexistent digest "fails to be found"), proving nothing about the
  # real bug. Pinning the previously-observed value as a same-line
  # assertion so a genuine change (e.g. genesis starting to record a
  # sha256 digest instead) surfaces as a clear, informative diff.
  # Reframing: `last` itself already fails trying to read genesis's own
  # inventory-list-log entry, NOT the pandoc-defaults type blob -- and
  # that entry's digest is sha256, matching the single-hash remote's
  # native format exactly as resolveWriteHashFormat's nil-request path
  # predicts (see remote_hash_format.go). So this is not a hash-type
  # mismatch at all: a *correctly* sha256-digested blob still can't be
  # read back. Capture the exact digest from the failure itself.
  run_dodder last -format inventory_list-sans-tai -repo_id legacy-single-hash-repo
  assert_success
  assert_output --partial 'does not exist locally'

  local inventory_log_digest
  inventory_log_digest="$(echo "$output" | grep -o 'sha256-[a-z0-9]*' | head -n1)"
  [ -n "$inventory_log_digest" ]

  # Decisive instrumentation: inspect the SFTP remote's actual
  # filesystem directly (madder-test-sftp-server serves a plain local
  # directory, so remote_root IS the on-disk store) rather than
  # inferring from error messages. This settles, definitively, whether
  # any blob was physically written at all, whether the write path
  # treated the store as single-hash (flat <bucket>/<rest> layout, no
  # <hash-type-id>/ parent directory) vs multi-hash, and gives a real
  # written path to cross-reference against the digest dodder expects.
  run bash -c "find '$remote_root' -mindepth 1 -not -name 'blob_store-config' | sort"
  assert_success
  # This is the actual root cause: what's present is a stray,
  # never-renamed upload temp file (sftpMover's tmp_<random-hex>
  # naming, store_remote_sftp.go), not a properly-bucketed blob.
  # sftpMover.Close() writes to this temp file then renames it to the
  # final bucket path via sftpClient.Rename(tempPath, finalPath) --
  # that rename either fails or never runs, and the failure doesn't
  # propagate back to the caller as an error (genesis reports success
  # throughout). No properly-bucketed blob file is ever created, so
  # every subsequent read correctly reports "not found" -- this was
  # never a read-side or hash-type bug at all.
  assert_line --regexp '/tmp_[0-9a-f]+$'
  refute_line --regexp '^[^ ]*/[0-9a-f]{2}/[0-9a-f]+$'

  run_dodder cat-object -repo_id legacy-single-hash-repo "$inventory_log_digest"
  assert_failure
  assert_output --partial 'not found'
}

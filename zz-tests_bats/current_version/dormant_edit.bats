#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

  # for shellcheck SC2154
  export output

  copy_from_version "$DIR"
}

teardown() {
  chflags_nouchg
}

# bats file_tags=user_story:config

# Config mutation is log-only (FDR 0020): dormant-edit no longer writes a
# konfig object and is silent on success. Dormant tags live inside the
# config blob, so the change is observed through the config surface.
function dormant_edit_and_change { # @test
  # EDITOR appends a TOML comment — semantically a no-op but the
  # blob bytes change, so a new config state is appended to the log.
  export EDITOR="bash -c 'echo \"# dormant-edit smoke comment\" >> \"\$0\"'"
  run_dodder dormant-edit
  assert_success
  assert_output '[konfig @blake2b256-7yf9a4qdhfc7a4alp0ywdd82e2wuarw23muggjffcpuxp6uyqr0se5a3hj !toml-config-v2]'

  # The change is observed through show-config (the config read
  # surface); the legacy `show :konfig` query is removed in a later task.
  run_dodder show-config
  assert_success
  assert_line '# dormant-edit smoke comment'

  # dormant-edit appends the new config state to the config log
  # (FDR 0020). The regenerated fixture seeds the config log root entry
  # at init, so show-config -history lists two entries oldest->newest:
  # the seeded root (object-sig, no mother) and the dormant-edit entry
  # (chained via mother-sig). The blob digest is content-addressed; tai
  # and ed25519 signatures are not.
  run_dodder show-config -history
  assert_success
  assert_equal "${#lines[@]}" 2
  # #294/FDR-0021 T4: the konfig is SELF provenance, so show-config -history
  # (empty handle) renders the bare `ed25519_pub-...` self form.
  assert_line --index 0 --regexp '^\[konfig @blake2b256-[a-z0-9]+ [0-9.]+ ed25519_pub-[a-z0-9]+ dodder-object-sig-v3@ed25519_sig-[a-z0-9]+ !toml-config-v2\]$'
  assert_line --index 1 --regexp '^\[konfig @blake2b256-[a-z0-9]+ [0-9.]+ ed25519_pub-[a-z0-9]+ dodder-object-mother-sig-v3@ed25519_sig-[a-z0-9]+ dodder-object-sig-v3@ed25519_sig-[a-z0-9]+ !toml-config-v2\]$'
}

function dormant_edit_and_dont_change { # @test
  export EDITOR=true
  run_dodder dormant-edit
  assert_success
  assert_output ''

  # No edit was made, so no config state is appended to the log (FDR 0020):
  # config mutation is log-only and `show :konfig` no longer queries. The
  # regenerated fixture seeds the config log root entry at init, so with no
  # mutation the log holds exactly that one seeded root entry (object-sig,
  # no mother); contrast the change case, which appends a second entry.
  run_dodder show-config -history
  assert_success
  assert_equal "${#lines[@]}" 1
  assert_line --index 0 --regexp '^\[konfig @blake2b256-[a-z0-9]+ [0-9.]+ ed25519_pub-[a-z0-9]+ dodder-object-sig-v3@ed25519_sig-[a-z0-9]+ !toml-config-v2\]$'
}

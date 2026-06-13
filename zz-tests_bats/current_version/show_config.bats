setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"
  export output
}

# bats file_tags=user_story:config

# Covers `show-config`, the read surface over the repo-local config log
# (config_log package, FDR 0020). init seeds the config log root entry
# (Task 6), so a freshly init-ed repo's `show-config` streams the default
# config TOML straight from the log head.

# On a fresh init the config log head is the seeded root entry, so
# `show-config` (no args) streams the deterministic default config TOML.
function show_config_head { # @test
  mkdir -p "$BATS_TEST_TMPDIR/fresh"
  cd "$BATS_TEST_TMPDIR/fresh"
  run_dodder_init_disable_age test-show-config-head

  run_dodder show-config
  assert_success
  assert_output 'blob-stores = [".default"]

[defaults]
type = "!md"
tags = []

[file-extensions]
config = "konfig"
conflict = "conflict"
lockfile = "object-lockfile"
organize = "md"
repo = "repo"
tag = "tag"
type = "type"
zettel = "zettel"

[cli-output]
print-blob_digests = true
print-colors = true
print-empty-blob_digests = false
print-flush = true
print-include-description = true
print-include-types = true
print-inventory_lists = true
print-matched-dormant = false
print-tags-always = true
print-time = true
print-unchanged = true

[cli-output.abbreviations]
zettel_ids = true
merkle_ids = true

[tools]
merge = ["vimdiff"]'
}

# Drives the full write/read round-trip through the new surfaces: a fresh
# repo (whose config log already holds the init-seeded root entry), one
# real `edit-config`, then `show-config` streams the edited blob and
# `show-config -history` lists both entries oldest->newest: the seeded
# root (object-sig, no mother) and the edited entry (chained via
# mother-sig). Reuses edit_config.bats's $EDITOR-script mechanism for the
# non-interactive edit. Signatures are non-deterministic on a fresh key,
# so the history lines are matched with regexps; the streamed blob is the
# raw TOML so its appended marker line is asserted exactly.
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
  assert_line --index 0 --regexp '^\[konfig @blake2b256-[a-z0-9]+ [0-9.]+ dodder-repo-public_key-v1@ed25519_pub-[a-z0-9]+ dodder-object-sig-v2@ed25519_sig-[a-z0-9]+ !toml-config-v2\]$'
  assert_line --index 1 --regexp '^\[konfig @blake2b256-[a-z0-9]+ [0-9.]+ dodder-repo-public_key-v1@ed25519_pub-[a-z0-9]+ dodder-object-mother-sig-v2@ed25519_sig-[a-z0-9]+ dodder-object-sig-v2@ed25519_sig-[a-z0-9]+ !toml-config-v2\]$'
}

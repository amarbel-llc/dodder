#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

  # for shellcheck SC2154
  export output
}

teardown() {
  chflags_nouchg
}

# bats file_tags=registry

# The per-host repo registry index (RFC-0007 registry v1, twin of madder's
# blob-store registry) lives under $XDG_STATE_HOME, which lib/common.bash
# redirects into $BATS_TEST_TMPDIR — so every test gets an isolated index
# for free. No walk-up, no ceiling.
index_dir() { echo "$XDG_STATE_HOME/dodder/index"; }

# index_link_count echoes the number of symlink entries in the index dir
# (0 when the dir does not exist yet).
index_link_count() {
  local d
  d="$(index_dir)"
  [[ -d $d ]] || {
    echo 0
    return
  }
  find "$d" -maxdepth 1 -type l | wc -l | tr -d ' '
}

# index_only_link echoes the path of the sole index symlink (fails the test
# if there is not exactly one).
index_only_link() {
  local links
  links="$(find "$(index_dir)" -maxdepth 1 -type l)"
  [[ $(echo "$links" | wc -l) -eq 1 ]] || fail "expected exactly one index link, got: $links"
  echo "$links"
}

function init_registers_cwd_repo_as_symlink { # @test
  run_dodder_init_disable_age

  assert_equal "$(index_link_count)" 1

  local link target base
  link="$(index_only_link)"
  # The entry is a symlink whose target is the repo's config-seed.
  target="$(readlink "$link")"
  assert_equal "$(basename "$target")" "config-seed"
  # Key is 16 lowercase hex chars, no extension: sha256(repo-dir)[:8].
  base="$(basename "$link")"
  [[ $base =~ ^[0-9a-f]{16}$ ]] || fail "index key not 16 hex chars: $base"
}

function init_registers_xdg_user_repo_too { # @test
  # All scopes register uniformly (RFC-0007). An undotted repo id is
  # XDG-user-scoped.
  run_dodder init \
    -yin <(cat_yin) \
    -yang <(cat_yang) \
    -encryption none \
    default
  assert_success

  assert_equal "$(index_link_count)" 1

  run_dodder repos-list -format=ndjson
  assert_success
  assert_output --regexp - <<-'EOM'
		^\{"name":"default","pubkey":"dodder-repo-public_key-v1[^"]+","id":"uuidv7-[^"]+","location":"~/\.xdg/data/dodder/repos/default"\}$
	EOM
}

function repos_list_shows_live_repo_with_identity { # @test
  run_dodder_init_disable_age

  # One row: the live cwd repo, deduped against its own registry entry,
  # carrying pubkey + uuidv7 instance id decoded from its config-seed.
  run_dodder repos-list -format=ndjson
  assert_success
  assert_output --regexp - <<-'EOM'
		^\{"name":"\.default","pubkey":"dodder-repo-public_key-v1[^"]+","id":"uuidv7-[^"]+","location":"~/\.dodder/local/share/repos/default"\}$
	EOM
}

function repos_list_marks_deleted_repo_stale { # @test
  run_dodder_init_disable_age

  # Delete the repo out from under the index; its symlink now dangles. The
  # scoped `.default` spelling is inferred from the dead target path.
  rm -rf .dodder

  run_dodder repos-list -format=ndjson
  assert_success
  assert_output --regexp - <<-'EOM'
		^\{"name":"\.default","location":"~/\.dodder/local/share/repos/default","stale":true\}$
	EOM
}

function info_repo_repos_is_unchanged_by_registry { # @test
  # The current-scope listing stays purely discovery-driven — the registry
  # feeds only repos-list, never info-repo repos (advisory-only contract).
  run_dodder_init_disable_age

  run_dodder info-repo repos
  assert_success
  assert_output '.default'
}

function registry_gc_keeps_fresh_prunes_aged { # @test
  run_dodder_init_disable_age
  rm -rf .dodder
  assert_equal "$(index_link_count)" 1

  # A just-registered dangling entry is within the grace window — kept.
  run_dodder registry-gc -retention=720h
  assert_success
  assert_output - <<-EOM
		TAP version 14
		ok 1 - registry-gc pruned 0 stale entries (retention 720h0m0s)
		1..1
	EOM
  assert_equal "$(index_link_count)" 1

  # Age the dangling symlink's own mtime past a short retention (touch -h
  # sets the link, not its missing target), then prune.
  local link
  link="$(index_only_link)"
  touch -h -d '25 hours ago' "$link"

  run_dodder registry-gc -retention=1h
  assert_success
  assert_output - <<-EOM
		TAP version 14
		ok 1 - registry-gc pruned 1 stale entry (retention 1h0m0s)
		1..1
	EOM
  assert_equal "$(index_link_count)" 0
}

function registry_gc_zero_retention_is_noop { # @test
  run_dodder_init_disable_age
  rm -rf .dodder

  run_dodder registry-gc -retention=0
  assert_success
  assert_output - <<-EOM
		TAP version 14
		ok 1 - registry-gc pruned 0 stale entries (retention 0s)
		1..1
	EOM
  assert_equal "$(index_link_count)" 1
}

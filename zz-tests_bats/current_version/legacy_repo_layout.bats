#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

  # for shellcheck SC2154
  export output
}

teardown() {
  chflags_nouchg
}

# bats file_tags=repo_id

# FDR-0019 (#363) decided explicit-migrate-only, not a read-in-place
# fallback for pre-repos/<name>/ legacy trees. These tests pin the
# resulting distinct-error-state behavior: a legacy flat-layout tree
# produces a message naming `migrate-repo-layout`, distinguishable from a
# location that is genuinely not a dodder directory at all.

# Bootstrap a legacy (pre-FDR-0019) flat-layout tree directly on disk: a
# bare config-seed under .dodder/local/share/, with no repos/<name>/
# nesting. Never produced by the current `dodder init` -- this simulates
# what a repo created before the repos/<name>/ layout landed looks like.
function bootstrap_legacy_layout_at {
  mkdir -p "$1/.dodder/local/share"
  echo "legacy config-seed placeholder" >"$1/.dodder/local/share/config-seed"
}

# bats test_tags=repo_id
function legacy_repo_layout_names_migrate_repo_layout { # @test
  bootstrap_legacy_layout_at legacy

  legacy_path="$(realpath legacy)"

  mkdir -p elsewhere
  pushd elsewhere || exit 1

  run_dodder show -dir-dodder "$legacy_path" :z
  assert_failure
  assert_output --regexp - <<-'EOM'
		Error: .*/legacy/\.dodder/local/share is a legacy \(pre-FDR-0019\) repo layout, not readable in place; migrate it first

		Cause:
		.*/legacy/\.dodder/local/share holds a legacy config-seed directly at the scope root, from before FDR-0019's `repos/<name>/` nesting\.
		The current binary only reads repos nested under repos/<name>/ -- it does not recognize a legacy flat tree in place \(a deliberate scope decision, not a bug\)\.

		Recovery:
		Migrate it: `dodder migrate-repo-layout -source ".*/legacy/\.dodder/local/share" -dest <new-path> -name default` \(never modifies -source\)\.
		Then point at the migrated tree \(e\.g\. `-dir-dodder <new-path>`\) and verify with a read-only command \(`dodder show`\) before treating the old tree as disposable\.
	EOM
}

# Negative case: a location with no config-seed at all (never a dodder
# directory, legacy or otherwise) keeps the original generic error and
# does NOT mention migrate-repo-layout.
# bats test_tags=repo_id
function nonexistent_repo_layout_does_not_name_migrate_repo_layout { # @test
  mkdir -p empty
  empty_path="$(realpath empty)"

  mkdir -p elsewhere
  pushd elsewhere || exit 1

  run_dodder show -dir-dodder "$empty_path" :z
  assert_failure
  assert_output --regexp - <<-'EOM'
		not in a dodder directory\. Looking for .*/empty/\.dodder/local/share/repos/default/config-seed
	EOM
}

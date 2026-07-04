#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

  # for shellcheck SC2154
  export output
}

teardown() {
  chflags_nouchg
}

# bats file_tags=user_story:scoped_repos

# FDR-0019 prototype: several user-scoped named repos coexist under
# $XDG_DATA_HOME/dodder/repos/<name>/ and are addressed by the name
# portion of -repo_id. Each repo has an independent metadata tree, so
# an object committed to one is invisible to the other.

# bats test_tags=user_story:scoped_repos,isolation
function scoped_user_repos_are_isolated { # @test
  run_dodder init -yin <(cat_yin) -yang <(cat_yang) work
  assert_success

  run_dodder init -yin <(cat_yin) -yang <(cat_yang) personal
  assert_success

  # a zettel committed to "work"
  run_dodder new -repo_id work -edit=false
  assert_success
  assert_output - <<-EOM
		[one/uno !md]
	EOM

  # ...is visible in "work"
  run_dodder show -repo_id work :z
  assert_success
  assert_output - <<-EOM
		[one/uno !md]
	EOM

  # ...and absent from "personal"
  run_dodder show -repo_id personal :z
  assert_success
  assert_output ''

  # each repo reports its own <handle>@<pubkey> identity (#294): the
  # location handle differs per repo and the pubkey is freshly generated,
  # so match the handle exactly and the pubkey by format.
  run_dodder info-repo -repo_id work id
  assert_success
  assert_output --regexp '^work@ed25519_pub-'

  run_dodder info-repo -repo_id personal id
  assert_success
  assert_output --regexp '^personal@ed25519_pub-'
}

# bats test_tags=user_story:scoped_repos,env_var
function scoped_repo_addressed_via_env_var { # @test
  run_dodder init -yin <(cat_yin) -yang <(cat_yang) work
  assert_success

  DODDER_REPO_ID=work run_dodder info-repo id
  assert_success
  assert_output --regexp '^work@ed25519_pub-'
}

# bats test_tags=user_story:scoped_repos,repo_id
function scoped_repo_multi_dot_resolves_nth_ancestor { # @test
  # FDR-0019 #281: nested same-named cwd repos. The operate path resolves
  # `.notes` (depth 0) to the nearest `.dodder/` hosting a `notes` repo, and
  # `..notes` (depth 1) to the next such ancestor up — store-aware via
  # directory_layout.ResolveNthAncestorMatch. The walk-up must reach above
  # the nesting, so raise the ceiling past $BATS_TEST_TMPDIR.
  export DODDER_TEST_CEILING="$(dirname "$BATS_TEST_TMPDIR")"
  export MADDER_TEST_CEILING="$(dirname "$BATS_TEST_TMPDIR")"

  # outer `notes` repo at $BATS_TEST_TMPDIR, with one zettel as its marker.
  run_dodder init -yin <(cat_yin) -yang <(cat_yang) .notes
  assert_success
  run_dodder new -repo_id .notes -edit=false
  assert_success

  # inner `notes` repo one level down, left empty — so the two repos are
  # distinguishable by content (inner empty, outer has the zettel) without
  # committing into the nested repo. (A nested-cwd-repo commit currently
  # trips a separate, pre-existing blob-store bug; see #283.)
  mkdir -p inner
  pushd inner || exit 1
  run_dodder init -yin <(cat_yin) -yang <(cat_yang) .notes
  assert_success

  # from inner: `.notes` (nearest match) is the inner repo — empty.
  run_dodder show -repo_id .notes :z
  assert_success
  assert_output ''

  # `..notes` (one matching ancestor up) is the outer repo — its zettel.
  run_dodder show -repo_id ..notes :z
  assert_success
  assert_output - <<-EOM
		[one/uno !md]
	EOM

  # `...notes` overflows — only two matching ancestors exist.
  run_dodder show -repo_id ...notes :z
  assert_failure
}

# bats test_tags=user_story:scoped_repos,repo_id
function explicit_user_repo_pins_to_user_scope_from_inside_workspace { # @test
  # FDR-0019: a bare `name` is XDG-user scope UNCONDITIONALLY. From inside a
  # .dodder/ workspace, `-repo_id default` must reach the user repo, not the
  # workspace-local default. Regression for the cwd-walk-up leak on the
  # explicit-XDGUser id (operate seam = MakeOperateEnvDir; init/info seam =
  # MakeEnvRepo). Pre-fix, the explicit bare name was hijacked to the
  # workspace repo by MakeDefault's walk-up.

  # user-scope `default`, populated, BEFORE any workspace exists in cwd.
  run_dodder init -yin <(cat_yin) -yang <(cat_yang) default
  assert_success
  run_dodder new -repo_id default -edit=false
  assert_success
  assert_output - <<-EOM
		[one/uno !md]
	EOM

  # workspace-local `.default` in cwd, left empty — content distinguishes the
  # two repos (user has one/uno, workspace empty), avoiding the nested-cwd
  # commit bug (#283).
  run_dodder init -yin <(cat_yin) -yang <(cat_yang) .default
  assert_success

  # (a) explicit bare `default` from inside the workspace -> the USER repo.
  run_dodder show -repo_id default :z # operate seam
  assert_success
  assert_output - <<-EOM
		[one/uno !md]
	EOM
  run_dodder info-repo -repo_id default id # MakeEnvRepo seam
  assert_success
  assert_output --regexp '^default@ed25519_pub-'

  # (b) the cwd spelling `.default` still reaches the workspace repo.
  run_dodder show -repo_id .default :z
  assert_success
  assert_output ''
  run_dodder info-repo -repo_id .default id
  assert_success
  assert_output --regexp '^\.default@ed25519_pub-'

  # (c) auto/empty still walks up to the workspace repo. The empty `show
  # :z` above proves it resolved to the (empty) workspace repo; with no
  # selector the identity carries no handle, so `info-repo id` renders the
  # bare pubkey (#294: empty handle -> bare pubkey fallback).
  run_dodder show :z
  assert_success
  assert_output ''
  run_dodder info-repo id
  assert_success
  assert_output --regexp '^ed25519_pub-'
}

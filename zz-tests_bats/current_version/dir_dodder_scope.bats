#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

  # for shellcheck SC2154
  export output
}

teardown() {
  chflags_nouchg
}

# bats file_tags=user_story:workspace,user_story:noworkspace

# These tests probe whether the -dir-dodder base-path override composes with
# FDR-0019 scope resolution for an OPERATE command (show). If it does, an
# arbitrary filesystem path is reproducible as `-dir-dodder <path>` + a scope —
# which would let ParentBackedWorkspace's `-parent` path branch collapse onto
# the repo-id resolver (#343). If it does NOT, the path branch must stay.
#
# Structured as: init a repo at a path, then address it from a DIFFERENT cwd
# purely via -dir-dodder + scope, and check the operate command reads it.

# Bootstrap: init a cwd-scoped repo rooted at $1, with one zettel.
function bootstrap_repo_at {
  (
    mkdir -p "$1"
    pushd "$1" || exit 1
    run_dodder_init

    run_dodder new -edit=false - <<-EOM
			---
			# probe zettel
			! md
			---

			probe body
		EOM
    assert_success
  )
}

# Baseline: from inside the repo dir, show :z lists the zettel. Confirms the
# bootstrap shape and gives the expected output the -dir-dodder cases compare
# against.
function dir_dodder_baseline_from_cwd { # @test
  bootstrap_repo_at repo

  pushd repo || exit 1
  run_dodder show :z
  assert_success
  assert_output --regexp - <<-EOM
		\[one/uno @blake2b256-.* !md "probe zettel"\]
	EOM
}

# The probe (#343): from a DIFFERENT cwd (no .dodder here), address the repo
# purely via -dir-dodder <path> for an OPERATE command. This is the property
# that lets a filesystem path be reproduced as `-dir-dodder <path>` + scope —
# so ParentBackedWorkspace's `-parent` path branch can collapse onto the
# repo-id resolver. Until MakeOperateEnvDir honors config.BasePath this reads
# the default XDG repo instead ("not in a dodder directory"); once it composes,
# it reads the repo at <path>.
function dir_dodder_operate_from_elsewhere { # @test
  bootstrap_repo_at repo
  repo_path="$(realpath repo)"

  mkdir -p elsewhere
  pushd elsewhere || exit 1

  run_dodder show -dir-dodder "$repo_path" :z
  assert_success
  assert_output --regexp - <<-EOM
		\[one/uno @blake2b256-.* !md "probe zettel"\]
	EOM
}

#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

  # for shellcheck SC2154
  export output
}

teardown() {
  chflags_nouchg
}

# bats file_tags=user_story:pull,user_story:repo,user_store:xdg,user_story:remote

function bootstrap_repo {
  (
    mkdir -p "$1"
    pushd "$1" >/dev/null || exit 1
    run_dodder_init
    bootstrap_content
  )
}

function bootstrap_repo_at_dir_with_name {
  (
    mkdir -p "$1"
    pushd "$1" || exit 1
    run_dodder_init
    bootstrap_content
  )
}

function bootstrap_content {
  {
    echo "---"
    echo "# wow"
    echo "- tag"
    echo "! md"
    echo "---"
    echo
    echo "body"
  } >to_add

  run_dodder new -edit=false to_add
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc !md "wow" tag]
	EOM

  run_dodder new -edit=false - <<-EOM
		---
		# zettel with multiple etiketten
		- this_is_the_first
		- this_is_the_second
		! md
		---

		zettel with multiple etiketten body
	EOM

  assert_success
  assert_output - <<-EOM
		[one/dos @blake2b256-fm7kce7793j3npevpm29spk04r6ycxv38dvx3hjxlzl8tcm5m3qq2mml86 !md "zettel with multiple etiketten" this_is_the_first this_is_the_second]
	EOM

  cat - >task.type <<-EOM
		binary = false
	EOM

  run_dodder checkin -delete task.type
  assert_success
  assert_output - <<-EOM
		[!task @blake2b256-qxzg22c3axe9m42tpwqd4usnfag4elp20q7zvnkgmyea4f4rwcwsurfp5e !toml-type-v2]
		          deleted [task.type]
	EOM
}

function try_add_new_after_pull {
  run_dodder new -edit=false - <<-EOM
		---
		# zettel after clone description
		! md
		---

		zettel after clone body
	EOM

  assert_success
  assert_output - <<-EOM
		[two/uno @blake2b256-kn7w3q7c3xvfa2p78wny0h79f7hd72nxtded0gvymu33wcnr2qmscl46ar !md "zettel after clone description"]
	EOM
}

function pull_history_zettel_type_tag_no_conflicts { # @test
  them="them"
  bootstrap_repo "$them"

  pushd "$BATS_TEST_TMPDIR" || exit 1

  run_dodder_init_disable_age

  run_dodder remote-add \
    toml-repo-local_override_path-v0 \
    "$(realpath them)" \
    them

  assert_success
  assert_output_unsorted --regexp - <<-'EOM'
		\[/them @blake2b256-.+ !toml-repo-local_override_path-v0]
	EOM

  run_dodder pull /them +zettel,typ,etikett

  assert_success
  assert_output_unsorted - <<-EOM
		copied Blob blake2b256-fm7kce7793j3npevpm29spk04r6ycxv38dvx3hjxlzl8tcm5m3qq2mml86 (36 B)
		copied Blob blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc (5 B)
		copied Blob blake2b256-qxzg22c3axe9m42tpwqd4usnfag4elp20q7zvnkgmyea4f4rwcwsurfp5e (15 B)
		[one/dos @blake2b256-fm7kce7793j3npevpm29spk04r6ycxv38dvx3hjxlzl8tcm5m3qq2mml86 !md "zettel with multiple etiketten" this_is_the_first this_is_the_second]
		[one/uno @blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc !md "wow" tag]
		[!task @blake2b256-qxzg22c3axe9m42tpwqd4usnfag4elp20q7zvnkgmyea4f4rwcwsurfp5e !toml-type-v2]
	EOM

  try_add_new_after_pull
}

function pull_history_zettel_type_tag_no_conflicts_stdio_local { # @test
  bootstrap_repo_at_dir_with_name them

  run_dodder_init_disable_age

  run_dodder remote-add \
    toml-repo-local_override_path-v0 \
    "$(realpath them)" \
    them

  assert_success
  assert_output_unsorted --regexp - <<-'EOM'
		\[/them @blake2b256-.+ !toml-repo-local_override_path-v0]
	EOM

  # TODO make this actually use a socket
  run_dodder pull /them +zettel,typ,etikett

  assert_success
  assert_output_unsorted --partial - <<-EOM
		copied Blob blake2b256-fm7kce7793j3npevpm29spk04r6ycxv38dvx3hjxlzl8tcm5m3qq2mml86 (36 B)
		copied Blob blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc (5 B)
		copied Blob blake2b256-qxzg22c3axe9m42tpwqd4usnfag4elp20q7zvnkgmyea4f4rwcwsurfp5e (15 B)
		[one/dos @blake2b256-fm7kce7793j3npevpm29spk04r6ycxv38dvx3hjxlzl8tcm5m3qq2mml86 !md "zettel with multiple etiketten" this_is_the_first this_is_the_second]
		[one/uno @blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc !md "wow" tag]
		[!task @blake2b256-qxzg22c3axe9m42tpwqd4usnfag4elp20q7zvnkgmyea4f4rwcwsurfp5e !toml-type-v2]
	EOM

  try_add_new_after_pull
}

# bats test_tags=timeout:long
function pull_history_zettel_type_tag_yes_conflicts_remote_second { # @test
  BATS_TEST_TIMEOUT=60
  them="them"
  bootstrap_repo "$them"

  pushd "$BATS_TEST_TMPDIR" || exit 1

  copy_from_version "$DIR"

  run_dodder show one/dos+
  assert_success
  assert_output - <<-EOM
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
	EOM

  run_dodder show +z
  assert_success
  assert_output_unsorted - <<-EOM
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
		[one/uno @blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 !md "wow ok" tag-1 tag-2]
	EOM

  run_dodder remote-add \
    toml-repo-local_override_path-v0 \
    "$(realpath them)" \
    them

  assert_success
  assert_output_unsorted --regexp - <<-'EOM'
		\[/them @blake2b256-.+ !toml-repo-local_override_path-v0]
	EOM

  run_dodder pull /them +zettel,typ,etikett

  assert_failure
  assert_output_unsorted --partial - <<-EOM
		copied Blob blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc (5 B)
		       conflicted [one/uno]
		copied Blob blake2b256-fm7kce7793j3npevpm29spk04r6ycxv38dvx3hjxlzl8tcm5m3qq2mml86 (36 B)
		       conflicted [one/dos]
		copied Blob blake2b256-qxzg22c3axe9m42tpwqd4usnfag4elp20q7zvnkgmyea4f4rwcwsurfp5e (15 B)
		[!task @blake2b256-qxzg22c3axe9m42tpwqd4usnfag4elp20q7zvnkgmyea4f4rwcwsurfp5e !toml-type-v2]
		import failed with conflicts, merging required
	EOM

  assert_output --partial - <<-EOM
		import failed with conflicts, merging required
	EOM

  run_dodder_init_workspace
  tree -n .

  run_dodder status
  assert_success
  assert_output_unsorted - <<-EOM
		       conflicted [one/dos]
		       conflicted [one/uno]
	EOM

  run_dodder show +z
  assert_success
  assert_output_unsorted - <<-EOM
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
		[one/uno @blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 !md "wow ok" tag-1 tag-2]
	EOM

  run_dodder merge-tool -merge-tool "bash -c 'cat \"\$2\" >\"\$3\"'" .
  assert_success
  assert_output_unsorted - <<-EOM
		[one/dos @blake2b256-fm7kce7793j3npevpm29spk04r6ycxv38dvx3hjxlzl8tcm5m3qq2mml86 !md "zettel with multiple etiketten" this_is_the_first this_is_the_second]
		[one/uno @blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc !md "wow" tag]
		          deleted [one/dos.conflict]
		          deleted [one/uno.conflict]
		          deleted [one/]
	EOM

  # TODO make sure merging includes the REMOTE in addition to the MERGED
  run_dodder show +z
  assert_success
  assert_output_unsorted - <<-EOM
		[one/dos @blake2b256-fm7kce7793j3npevpm29spk04r6ycxv38dvx3hjxlzl8tcm5m3qq2mml86 !md "zettel with multiple etiketten" this_is_the_first this_is_the_second]
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
		[one/uno @blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 !md "wow ok" tag-1 tag-2]
		[one/uno @blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc !md "wow" tag]
	EOM

  run_dodder show -format text one/dos
  assert_success
  assert_output --regexp - <<EOM
---
# zettel with multiple etiketten
- this_is_the_first
- this_is_the_second
! md@.*
---
EOM

  run_dodder show one/dos+
  assert_success
  assert_output - <<-EOM
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		[one/dos @blake2b256-fm7kce7793j3npevpm29spk04r6ycxv38dvx3hjxlzl8tcm5m3qq2mml86 !md "zettel with multiple etiketten" this_is_the_first this_is_the_second]
	EOM

  try_add_new_after_pull
}

# bats test_tags=timeout:long
function pull_history_zettel_type_tag_yes_conflicts_allowed_remote_first { # @test
  BATS_TEST_TIMEOUT=30
  pushd "$BATS_TEST_TMPDIR" || exit 1
  run_dodder_init_disable_age

  run_dodder new -edit=false - <<-EOM
		---
		# zettel after clone description
		! md
		---

		zettel after clone body
	EOM

  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-kn7w3q7c3xvfa2p78wny0h79f7hd72nxtded0gvymu33wcnr2qmscl46ar !md "zettel after clone description"]
	EOM

  them="them"
  bootstrap_repo "$them"
  assert_success

  pushd "$BATS_TEST_TMPDIR" || exit 1

  run_dodder remote-add \
    toml-repo-local_override_path-v0 \
    "$(realpath them)" \
    them

  assert_success
  assert_output_unsorted --regexp - <<-'EOM'
		\[/them @blake2b256-.+ !toml-repo-local_override_path-v0]
	EOM

  run_dodder pull -allow-merge-conflicts /them +zettel,typ,etikett
  assert_success
  # TODO address the bandaid of two `[tag]` objects
  assert_output_unsorted - <<-EOM
		copied Blob blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc (5 B)
		[one/uno @blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc !md "wow" tag]
		copied Blob blake2b256-fm7kce7793j3npevpm29spk04r6ycxv38dvx3hjxlzl8tcm5m3qq2mml86 (36 B)
		[one/dos @blake2b256-fm7kce7793j3npevpm29spk04r6ycxv38dvx3hjxlzl8tcm5m3qq2mml86 !md "zettel with multiple etiketten" this_is_the_first this_is_the_second]
		copied Blob blake2b256-qxzg22c3axe9m42tpwqd4usnfag4elp20q7zvnkgmyea4f4rwcwsurfp5e (15 B)
		[!task @blake2b256-qxzg22c3axe9m42tpwqd4usnfag4elp20q7zvnkgmyea4f4rwcwsurfp5e !toml-type-v2]
	EOM

  run_dodder status
  assert_success
  assert_output_unsorted ''

  run_dodder show -format text one/dos
  assert_success
  assert_output --regexp - <<EOM
---
# zettel with multiple etiketten
- this_is_the_first
- this_is_the_second
! md@.*
---
EOM

  run_dodder show one/uno+
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-kn7w3q7c3xvfa2p78wny0h79f7hd72nxtded0gvymu33wcnr2qmscl46ar !md "zettel after clone description"]
		[one/uno @blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc !md "wow" tag]
	EOM
}

# bats test_tags=timeout:long
function pull_history_zettel_type_tag_yes_conflicts_remote_first { # @test
  BATS_TEST_TIMEOUT=30
  pushd "$BATS_TEST_TMPDIR" || exit 1
  run_dodder_init_disable_age

  run_dodder new -edit=false - <<-EOM
		---
		# zettel after clone description
		! md
		---

		zettel after clone body
	EOM

  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-kn7w3q7c3xvfa2p78wny0h79f7hd72nxtded0gvymu33wcnr2qmscl46ar !md "zettel after clone description"]
	EOM

  them="them"
  bootstrap_repo "$them"
  assert_success

  pushd "$BATS_TEST_TMPDIR" || exit 1

  run_dodder remote-add \
    toml-repo-local_override_path-v0 \
    "$(realpath them)" \
    them

  assert_success
  assert_output_unsorted --regexp - <<-'EOM'
		\[/them @blake2b256-.+ !toml-repo-local_override_path-v0]
	EOM

  run_dodder pull /them +zettel,typ,etikett

  assert_failure
  assert_output_unsorted --partial - <<-EOM
		       conflicted [one/uno]
		[!task @blake2b256-qxzg22c3axe9m42tpwqd4usnfag4elp20q7zvnkgmyea4f4rwcwsurfp5e !toml-type-v2]
		[one/dos @blake2b256-fm7kce7793j3npevpm29spk04r6ycxv38dvx3hjxlzl8tcm5m3qq2mml86 !md "zettel with multiple etiketten" this_is_the_first this_is_the_second]
		copied Blob blake2b256-fm7kce7793j3npevpm29spk04r6ycxv38dvx3hjxlzl8tcm5m3qq2mml86 (36 B)
		copied Blob blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc (5 B)
		copied Blob blake2b256-qxzg22c3axe9m42tpwqd4usnfag4elp20q7zvnkgmyea4f4rwcwsurfp5e (15 B)
		import failed with conflicts, merging required
	EOM

  assert_output --partial - <<-EOM
		import failed with conflicts, merging required
	EOM

  run_dodder status
  assert_success
  assert_output_unsorted - <<-EOM
		       conflicted [one/uno]
	EOM

  run_dodder merge-tool -merge-tool "bash -c 'cat \"\$2\" >\"\$3\"'" .
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc !md "wow" tag]
		          deleted [one/uno.conflict]
		          deleted [one/]
	EOM

  run_dodder show -format text one/dos
  assert_success
  assert_output --regexp - <<EOM
---
# zettel with multiple etiketten
- this_is_the_first
- this_is_the_second
! md@.*
---
EOM

  run_dodder show one/uno+
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-kn7w3q7c3xvfa2p78wny0h79f7hd72nxtded0gvymu33wcnr2qmscl46ar !md "zettel after clone description"]
		[one/uno @blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc !md "wow" tag]
	EOM
}

function pull_history_default_no_conflict { # @test
  them="them"
  bootstrap_repo "$them"

  pushd "$BATS_TEST_TMPDIR" || exit 1

  run_dodder_init_disable_age

  run_dodder remote-add \
    toml-repo-local_override_path-v0 \
    "$(realpath them)" \
    them

  assert_success
  assert_output_unsorted --regexp - <<-'EOM'
		\[/them @blake2b256-.+ !toml-repo-local_override_path-v0]
	EOM

  run_dodder pull /them
  assert_success

  run_dodder show +?z,t,e
  assert_success
  assert_golden_unsorted pull_history_zettel_type_tag

  run_dodder show one/dos+
  assert_success
  assert_output - <<-EOM
		[one/dos @blake2b256-fm7kce7793j3npevpm29spk04r6ycxv38dvx3hjxlzl8tcm5m3qq2mml86 !md "zettel with multiple etiketten" this_is_the_first this_is_the_second]
	EOM

  run_dodder show !md:t
  assert_success
  assert_golden pull_show_md_type

  run_dodder show !task:t
  assert_success
  assert_output - <<-EOM
		[!task @blake2b256-qxzg22c3axe9m42tpwqd4usnfag4elp20q7zvnkgmyea4f4rwcwsurfp5e !toml-type-v2]
	EOM

  try_add_new_after_pull
}

function pull_history_zettel_one_abbr { # @test
  # TODO add support for abbreviations in remote transfers
  skip
  them="them"
  bootstrap_repo "$them"
  assert_success

  pushd "$BATS_TEST_TMPDIR" || exit 1

  run_dodder_init_disable_age

  run_dodder remote-add \
    "$(realpath them)" \
    them
  assert_success
  assert_output_unsorted --regexp - <<-'EOM'
		\[/them @[0-9a-z]+ !toml-repo-dotenv_xdg-v0]
	EOM

  run_dodder pull -include-blobs=false /them o/u+

  assert_success
  assert_output_unsorted - <<-EOM
		[one/uno @blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc !md "wow" tag]
	EOM

  run_dodder show one/uno+
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc !md "wow" tag]
	EOM
}

function pull_history_zettels_no_conflict_no_blobs { # @test
  them="them"
  bootstrap_repo "$them"

  pushd "$BATS_TEST_TMPDIR" || exit 1

  run_dodder_init_disable_age

  run_dodder remote-add \
    toml-repo-local_override_path-v0 \
    "$(realpath them)" \
    them

  assert_success
  assert_output_unsorted --regexp - <<-'EOM'
		\[/them @blake2b256-.+ !toml-repo-local_override_path-v0]
	EOM

  run_dodder pull -exclude-blobs /them +zettel

  assert_success
  assert_output_unsorted - <<-EOM
		[one/dos @blake2b256-fm7kce7793j3npevpm29spk04r6ycxv38dvx3hjxlzl8tcm5m3qq2mml86 !md "zettel with multiple etiketten" this_is_the_first this_is_the_second]
		[one/uno @blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc !md "wow" tag]
	EOM

  run_dodder show one/dos+
  assert_success
  assert_output - <<-EOM
		[one/dos @blake2b256-fm7kce7793j3npevpm29spk04r6ycxv38dvx3hjxlzl8tcm5m3qq2mml86 !md "zettel with multiple etiketten" this_is_the_first this_is_the_second]
	EOM

  run_dodder show -format blob one/dos
  assert_failure

  try_add_new_after_pull
}

# pull_does_not_seed_config_from_source is the RFC 0005 conformance test for
# "Client Seeding, MUST NOT seed on pull": config is repo-local and a pull
# must never adopt the remote's config. The source edits its config to a
# distinctive marker; the puller then pulls objects from it and asserts its
# own show-config does NOT contain the marker. The config log is repo-local
# (.dodder/config_log), so the source and puller have independent config logs
# even sharing one XDG home. Transport-independent (local override path here):
# the client discards any config descriptor and never fetches config on pull.
function pull_does_not_seed_config_from_source { # @test
  them="them"
  bootstrap_repo "$them"

  # Edit the SOURCE's config to a distinctive marker so a wrongful seed
  # would be detectable in the puller's config.
  (
    pushd "$them" >/dev/null || exit 1
    export EDITOR="bash -c 'echo \"# pull-should-not-seed\" >> \"\$0\"'"
    run_dodder edit-config
    assert_success

    run_dodder show-config
    assert_success
    assert_line '# pull-should-not-seed'
  )

  pushd "$BATS_TEST_TMPDIR" || exit 1

  run_dodder_init_disable_age

  run_dodder remote-add \
    toml-repo-local_override_path-v0 \
    "$(realpath them)" \
    them

  assert_success
  assert_output_unsorted --regexp - <<-'EOM'
		\[/them @blake2b256-.+ !toml-repo-local_override_path-v0]
	EOM

  run_dodder pull /them +zettel,typ,etikett
  assert_success

  # The pull moved objects but MUST NOT have adopted the source's config:
  # the marker the source appended is absent from the puller's config.
  run_dodder show-config
  assert_success
  refute_line '# pull-should-not-seed'
}

function pull_direct_local_path_no_conflicts { # @test
  them="them"
  bootstrap_repo "$them"

  pushd "$BATS_TEST_TMPDIR" || exit 1

  run_dodder_init_disable_age

  run_dodder pull -direct "$(realpath them)" +zettel,typ,etikett

  assert_success
  assert_output_unsorted - <<-EOM
		copied Blob blake2b256-fm7kce7793j3npevpm29spk04r6ycxv38dvx3hjxlzl8tcm5m3qq2mml86 (36 B)
		copied Blob blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc (5 B)
		copied Blob blake2b256-qxzg22c3axe9m42tpwqd4usnfag4elp20q7zvnkgmyea4f4rwcwsurfp5e (15 B)
		[one/dos @blake2b256-fm7kce7793j3npevpm29spk04r6ycxv38dvx3hjxlzl8tcm5m3qq2mml86 !md "zettel with multiple etiketten" this_is_the_first this_is_the_second]
		[one/uno @blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc !md "wow" tag]
		[!task @blake2b256-qxzg22c3axe9m42tpwqd4usnfag4elp20q7zvnkgmyea4f4rwcwsurfp5e !toml-type-v2]
	EOM

  try_add_new_after_pull
}

# bats test_tags=user_story:pull,user_story:referenced_objects
function pull_direct_blob_references_transferred { # @test
  them="$BATS_TEST_TMPDIR/them"
  mkdir -p "$them"

  pushd "$them" || exit 1

  run_dodder_init_disable_age

  # Write a standalone blob to the source store
  run_madder write -format tap <(echo "referenced content")
  assert_success
  ref_blob_sha="$(echo "$output" | grep -oP 'blake2b256-\S+' | head -1)"

  # Create a type with reference discovery for blob refs
  cat >refblob.type <<-'TYPEFILE'
		---
		! toml-type-v2
		---

		file-extension = "md"
		vim-syntax-type = "markdown"

		[references]
		shell = ["bash", "-c"]
		script = "grep -oP '(@blake2b256-[a-z0-9]+|\\[\\[(.+?)\\]\\])' | sed 's/\\[\\[//;s/\\]\\]//' | sed 's/^@\\(blake2b256-[a-z0-9]*\\)/@\\1 !refblob/'"
	TYPEFILE

  run_dodder checkin -delete refblob.type
  assert_success

  # Create a zettel whose body references the standalone blob
  run_dodder new -edit=false - <<-EOM
		---
		# zettel with blob ref
		! refblob
		---

		See blob @${ref_blob_sha} for details.
	EOM
  assert_success

  popd || exit 1

  # Set up destination repo
  us="$BATS_TEST_TMPDIR/us"
  mkdir -p "$us"
  pushd "$us" || exit 1

  run_dodder_init_disable_age

  # Pull from source
  run_dodder pull -direct "$(realpath "$them")" +zettel,typ,etikett
  assert_success

  # Verify the referenced blob was transferred to the destination
  run_madder cat "$ref_blob_sha"
  assert_success
  assert_output "$(printf "%s\n" "referenced content")"
}

# bats test_tags=user_story:pull,user_story:referenced_objects
function pull_direct_hyphenated_type_name_no_phantom { # @test
  them="$BATS_TEST_TMPDIR/them"
  mkdir -p "$them"

  pushd "$them" || exit 1

  run_dodder_init_disable_age

  # Write a standalone blob to the source store
  run_madder write -format tap <(echo "referenced content")
  assert_success
  ref_blob_sha="$(echo "$output" | grep -oP 'blake2b256-\S+' | head -1)"

  # Create a type with hyphen in name and reference discovery
  cat >ref-blob.type <<-'TYPEFILE'
		---
		! toml-type-v2
		---

		file-extension = "md"
		vim-syntax-type = "markdown"

		[references]
		shell = ["bash", "-c"]
		script = "grep -oP '(@blake2b256-[a-z0-9]+|\\[\\[(.+?)\\]\\])' | sed 's/\\[\\[//;s/\\]\\]//' | sed 's/^@\\(blake2b256-[a-z0-9]*\\)/@\\1 !ref-blob/'"
	TYPEFILE

  run_dodder checkin -delete ref-blob.type
  assert_success

  # Create a zettel whose body references the standalone blob
  run_dodder new -edit=false - <<-EOM
		---
		# zettel with blob ref
		! ref-blob
		---

		See blob @${ref_blob_sha} for details.
	EOM
  assert_success

  popd || exit 1

  # Set up destination repo
  us="$BATS_TEST_TMPDIR/us"
  mkdir -p "$us"
  pushd "$us" || exit 1

  run_dodder_init_disable_age

  # Pull from source — should NOT produce a phantom !ref type
  run_dodder pull -direct "$(realpath "$them")" +zettel,typ,etikett
  assert_success

  # Verify the referenced blob was transferred to the destination
  run_madder cat "$ref_blob_sha"
  assert_success
  assert_output "$(printf "%s\n" "referenced content")"
}

# bats test_tags=user_story:pull,user_story:referenced_objects
function pull_direct_blob_reference_alias_survives { # @test
  them="$BATS_TEST_TMPDIR/them"
  mkdir -p "$them"

  pushd "$them" || exit 1

  run_dodder_init_disable_age

  # Create a zettel with an aliased blob reference
  run_dodder new -edit=false - <<-'EOM'
		---
		# aliased blob ref
		- hero-image < @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md
		! md
		---

		content
	EOM
  assert_success

  # Verify alias exists in source
  run_dodder show -format text one/uno:
  assert_success
  assert_output --partial 'hero-image'

  popd || exit 1

  # Set up destination repo
  us="$BATS_TEST_TMPDIR/us"
  mkdir -p "$us"
  pushd "$us" || exit 1

  run_dodder_init_disable_age

  # Pull from source
  run_dodder pull -direct "$(realpath "$them")" +zettel,typ,etikett
  assert_success

  # Verify alias survived the pull (binary stream index round-trip)
  run_dodder show -format text one/uno:
  assert_success
  assert_output --partial 'hero-image'
}

# bats test_tags=user_story:pull,user_story:referenced_objects
function pull_direct_multiple_blob_references_transferred { # @test
  them="$BATS_TEST_TMPDIR/them"
  mkdir -p "$them"

  pushd "$them" || exit 1

  run_dodder_init_disable_age

  # Write two standalone blobs to the source store
  run_madder write -format tap <(echo "first referenced blob")
  assert_success
  blob_sha_1="$(echo "$output" | grep -oP 'blake2b256-\S+' | head -1)"

  run_madder write -format tap <(echo "second referenced blob")
  assert_success
  blob_sha_2="$(echo "$output" | grep -oP 'blake2b256-\S+' | head -1)"

  # Create a type with reference discovery for blob refs
  cat >refblob.type <<-'TYPEFILE'
		---
		! toml-type-v2
		---

		file-extension = "md"
		vim-syntax-type = "markdown"

		[references]
		shell = ["bash", "-c"]
		script = "grep -oP '(@blake2b256-[a-z0-9]+|\\[\\[(.+?)\\]\\])' | sed 's/\\[\\[//;s/\\]\\]//' | sed 's/^@\\(blake2b256-[a-z0-9]*\\)/@\\1 !refblob/'"
	TYPEFILE

  run_dodder checkin -delete refblob.type
  assert_success

  # Create a zettel whose body references both standalone blobs
  run_dodder new -edit=false - <<-EOM
		---
		# zettel with two blob refs
		! refblob
		---

		First: @${blob_sha_1}
		Second: @${blob_sha_2}
	EOM
  assert_success

  popd || exit 1

  # Set up destination repo
  us="$BATS_TEST_TMPDIR/us"
  mkdir -p "$us"
  pushd "$us" || exit 1

  run_dodder_init_disable_age

  # Pull from source
  run_dodder pull -direct "$(realpath "$them")" +zettel,typ,etikett
  assert_success

  # Verify both referenced blobs were transferred
  run_madder cat "$blob_sha_1"
  assert_success
  assert_output "$(printf "%s\n" "first referenced blob")"

  run_madder cat "$blob_sha_2"
  assert_success
  assert_output "$(printf "%s\n" "second referenced blob")"
}

# bats test_tags=user_story:pull,user_story:referenced_objects
function pull_direct_transitive_blob_references_transferred { # @test
  them="$BATS_TEST_TMPDIR/them"
  mkdir -p "$them"

  pushd "$them" || exit 1

  run_dodder_init_disable_age

  # Write a leaf blob (no further references)
  run_madder write -format tap <(echo "leaf blob content")
  assert_success
  leaf_blob_sha="$(echo "$output" | grep -oP 'blake2b256-\S+' | head -1)"

  # Write an intermediate blob whose content references the leaf blob
  run_madder write -format tap <(echo "tree entry: @${leaf_blob_sha}")
  assert_success
  tree_blob_sha="$(echo "$output" | grep -oP 'blake2b256-\S+' | head -1)"

  # Create treeblob type: its discovery script emits blob refs from content
  cat >treeblob.type <<-'TYPEFILE'
		---
		! toml-type-v2
		---

		file-extension = "tree"

		[references]
		shell = ["bash", "-c"]
		script = "grep -oP '@blake2b256-[a-z0-9]+' | sed 's/^@\\(blake2b256-[a-z0-9]*\\)/@\\1 !treeblob/'"
	TYPEFILE

  run_dodder checkin -delete treeblob.type
  assert_success

  # Create refblob type: its discovery script emits blob refs with treeblob type
  cat >refblob.type <<-'TYPEFILE'
		---
		! toml-type-v2
		---

		file-extension = "md"
		vim-syntax-type = "markdown"

		[references]
		shell = ["bash", "-c"]
		script = "grep -oP '@blake2b256-[a-z0-9]+' | sed 's/^@\\(blake2b256-[a-z0-9]*\\)/@\\1 !treeblob/'"
	TYPEFILE

  run_dodder checkin -delete refblob.type
  assert_success

  # Create a zettel whose body references the intermediate (tree) blob
  run_dodder new -edit=false - <<-EOM
		---
		# zettel with transitive blob refs
		! refblob
		---

		See tree @${tree_blob_sha} for details.
	EOM
  assert_success

  popd || exit 1

  # Set up destination repo
  us="$BATS_TEST_TMPDIR/us"
  mkdir -p "$us"
  pushd "$us" || exit 1

  run_dodder_init_disable_age

  # Pull from source — should transfer the zettel, both blobs transitively
  run_dodder pull -direct "$(realpath "$them")" +zettel,typ,etikett
  assert_success

  # Verify the intermediate tree blob was transferred
  run_madder cat "$tree_blob_sha"
  assert_success
  assert_output "$(printf "%s\n" "tree entry: @${leaf_blob_sha}")"

  # Verify the leaf blob was transferred transitively
  run_madder cat "$leaf_blob_sha"
  assert_success
  assert_output "$(printf "%s\n" "leaf blob content")"
}

function pull_direct_no_repo_at_path { # @test
  pushd "$BATS_TEST_TMPDIR" || exit 1
  run_dodder_init_disable_age

  mkdir -p empty_dir

  run_dodder pull -direct "$(realpath empty_dir)" +zettel

  assert_failure
  assert_output --partial 'not in a dodder directory'
}

# Regression coverage for a real-world failure: pulling a Tag object with a
# rich description (multi-paragraph, embedded literal double-quotes, an
# em-dash) from a long-lived repo aborted with `invalid signature: ed25519:
# invalid signature` during re-verification after inventory_list decode --
# even though `fsck` on the source repo passed clean and `show -format
# inventory_list` rendered identical output in both `dodder` and `der`.
#
# This test alone does NOT reproduce the failure (confirmed): a freshly
# created tag with the same description shape pulls cleanly. It's kept as a
# regression guard for the description-shape half of the investigation, and
# a base for pull_tag_with_mother_sig_history_signature_survives below,
# which adds the other half (a real mother-sig chain via organize).
function pull_tag_with_rich_description_signature_survives { # @test
  them="them"

  (
    mkdir -p "$them"
    pushd "$them" || exit 1
    run_dodder_init

    # Mirrors a real tag's shape: multi-paragraph description (blank-line
    # separated), embedded literal double-quotes, an em-dash, and two
    # meta-tags.
    run_dodder new -edit=false -object-id widget-throughput_lag \
      -description 'Widget queue throughput/lag investigation: theory that the widget'"'"'s queue consumer apps hit a throughput ceiling before backlog climbs. Dashboard: Grafana "QueueMetrics" (uid AbCdEfGhI) — personal working copy.

FINDING (revised): widget_queue_legacy_backlog_metric no longer exists — use widget_queue_consumer_fetch_manager_metrics_records_lag_avg instead, confirmed live.

A narrow 3h sample showed only small isolated per-partition lag spikes — arguing AGAINST a throughput ceiling. But a wider window surfaced a bigger episode where rate climbed steadily with no plateau — the consumer was working harder and still could not outpace incoming volume.' \
      -tags active,priority-2_want
    assert_success
  )

  pushd "$BATS_TEST_TMPDIR" || exit 1
  run_dodder_init_disable_age

  run_dodder pull -direct "$(realpath "$them")" +etikett
  assert_success
  refute_output --partial "invalid signature"

  run_dodder show 'widget-throughput_lag:e'
  assert_success
}

# Same rich-description tag as above, but with a SECOND version created via
# `organize -mode commit-directly` before the pull, so the pulled object
# carries a real (non-null) mother-sig -- the one property the real failing
# object had that the fresh single-version tag above does not.
function pull_tag_with_mother_sig_history_signature_survives { # @test
  them="them"

  (
    mkdir -p "$them"
    pushd "$them" || exit 1
    run_dodder_init

    run_dodder new -edit=false -object-id widget-throughput_lag \
      -description 'Widget queue throughput/lag investigation: theory that the widget'"'"'s queue consumer apps hit a throughput ceiling before backlog climbs. Dashboard: Grafana "QueueMetrics" (uid AbCdEfGhI) — personal working copy.

FINDING (revised): widget_queue_legacy_backlog_metric no longer exists — use widget_queue_consumer_fetch_manager_metrics_records_lag_avg instead, confirmed live.

A narrow 3h sample showed only small isolated per-partition lag spikes — arguing AGAINST a throughput ceiling. But a wider window surfaced a bigger episode where rate climbed steadily with no plateau — the consumer was working harder and still could not outpace incoming volume.' \
      -tags active
    assert_success

    # Second version: add a meta-tag via organize, giving the tag a real
    # mother-sig chain (v1 -> v2) rather than a fresh/null one.
    run_dodder organize -mode commit-directly :e <<-EOM
			# priority-2_want
			- [widget-throughput_lag]
		EOM
    assert_success
  )

  pushd "$BATS_TEST_TMPDIR" || exit 1
  run_dodder_init_disable_age

  run_dodder pull -direct "$(realpath "$them")" +etikett
  assert_success
  refute_output --partial "invalid signature"

  run_dodder show 'widget-throughput_lag+:e'
  assert_success
}

# Regression test for the actual root cause behind the two tests above: a
# description with an embedded blank-line paragraph break (a real "\n\n",
# not just long single-line text) was silently corrupted by two
# independent bugs on its way through export -> import:
#
#   1. WRITE-SIDE (box_format/transacted.go, object_metadata_box_builder):
#      the inventory_list/archive wire-format encoder called
#      Description.StringWithoutNewlines() -- a DISPLAY-ONLY helper meant
#      for collapsing multi-paragraph text onto one terminal line --
#      silently replacing every embedded "\n" with a space before writing
#      the signed wire-format bytes.
#   2. READ-SIDE (doddish's Scanner.consumeLiteralOrFieldValue): even once
#      the writer correctly `%q`-escapes a real newline as the two-byte
#      sequence backslash+'n', the scanner's escape handling only stripped
#      the backslash -- it did not interpret standard escape sequences, so
#      reading `\n` back produced the literal letter 'n' glued onto the
#      surrounding text instead of a real newline.
#
# Either bug alone changes the digest computed from the object's fields
# (object_fmt_digest.WriteDigest hashes one "Description <line>" key per
# newline-separated paragraph), so a description round-tripped through
# export/import would recompute to a DIFFERENT digest than the one
# originally signed -- triggering "invalid signature: ed25519: invalid
# signature" on pull/import re-verification, even though `fsck` (which
# trusts the stored digest without recomputing it) reported no problem.
#
# This test proves both are fixed for newly-created objects: it asserts
# not just "no signature error" (the earlier two tests already covered
# that with a real-world-shaped description) but that the description's
# blank-line paragraph break survives byte-for-byte -- the exact property
# that was silently lost before.
function pull_tag_with_embedded_blank_line_description_round_trips { # @test
  them="them"

  (
    mkdir -p "$them"
    pushd "$them" || exit 1
    run_dodder_init

    run_dodder new -edit=false -object-id blank-line-description-tag \
      -description "$(printf 'paragraph one text here.\n\nparagraph two text here.')"
    assert_success
  )

  pushd "$BATS_TEST_TMPDIR" || exit 1
  run_dodder_init_disable_age

  run_dodder pull -direct "$(realpath "$them")" +etikett
  assert_success
  refute_output --partial "invalid signature"

  run_dodder show -format text 'blank-line-description-tag:e'
  assert_success
  # The blank-paragraph-separator line renders as "# " (hash + trailing
  # space, from writeCommonMetadataFormat's `# %s` with an empty line) --
  # not a bare "#". Built with printf (not a heredoc) so the trailing
  # space survives editor/whitespace-trimming tooling.
  expected="$(printf -- '---\n# paragraph one text here.\n# \n# paragraph two text here.\n! toml-tag-v1\n---')"
  assert_output "$expected"
}

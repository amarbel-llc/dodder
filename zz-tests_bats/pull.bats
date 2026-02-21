#! /usr/bin/env bats

setup() {
	load "$(dirname "$BATS_TEST_FILE")/common.bash"

	# for shellcheck SC2154
	export output
}

teardown() {
	chflags_and_rm
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
		run_dodder_init -override-xdg-with-cwd "$1"
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
		[!task @blake2b256-qxzg22c3axe9m42tpwqd4usnfag4elp20q7zvnkgmyea4f4rwcwsurfp5e !toml-type-v1]
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
		[!task @blake2b256-qxzg22c3axe9m42tpwqd4usnfag4elp20q7zvnkgmyea4f4rwcwsurfp5e !toml-type-v1]
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
		[!task @blake2b256-qxzg22c3axe9m42tpwqd4usnfag4elp20q7zvnkgmyea4f4rwcwsurfp5e !toml-type-v1]
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
		[!task @blake2b256-qxzg22c3axe9m42tpwqd4usnfag4elp20q7zvnkgmyea4f4rwcwsurfp5e !toml-type-v1]
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

	run_dodder merge-tool -merge-tool "/bin/bash -c 'cat \"\$2\" >\"\$3\"'" .
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
	assert_output --regexp - <<-EOM
		---
		# zettel with multiple etiketten
		- this_is_the_first
		- this_is_the_second
		! md@.*
		---

		zettel with multiple etiketten body
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
		[!task @blake2b256-qxzg22c3axe9m42tpwqd4usnfag4elp20q7zvnkgmyea4f4rwcwsurfp5e !toml-type-v1]
	EOM

	run_dodder status
	assert_success
	assert_output_unsorted ''

	run_dodder show -format text one/dos
	assert_success
	assert_output --regexp - <<-EOM
		---
		# zettel with multiple etiketten
		- this_is_the_first
		- this_is_the_second
		! md@.*
		---

		zettel with multiple etiketten body
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
		[!task @blake2b256-qxzg22c3axe9m42tpwqd4usnfag4elp20q7zvnkgmyea4f4rwcwsurfp5e !toml-type-v1]
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

	run_dodder merge-tool -merge-tool "/bin/bash -c 'cat \"\$2\" >\"\$3\"'" .
	assert_success
	assert_output - <<-EOM
		[one/uno @blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc !md "wow" tag]
		          deleted [one/uno.conflict]
		          deleted [one/]
	EOM

	run_dodder show -format text one/dos
	assert_success
	assert_output --regexp - <<-EOM
		---
		# zettel with multiple etiketten
		- this_is_the_first
		- this_is_the_second
		! md@.*
		---

		zettel with multiple etiketten body
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
	assert_output_unsorted - <<-EOM
		[!md @blake2b256-3kj7xgch6rjkq64aa36pnjtn9mdnl89k8pdhtlh33cjfpzy8ek4qnufx0m !toml-type-v1]
		[one/dos @blake2b256-fm7kce7793j3npevpm29spk04r6ycxv38dvx3hjxlzl8tcm5m3qq2mml86 !md "zettel with multiple etiketten" this_is_the_first this_is_the_second]
		[one/uno @blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc !md "wow" tag]
		[!task @blake2b256-qxzg22c3axe9m42tpwqd4usnfag4elp20q7zvnkgmyea4f4rwcwsurfp5e !toml-type-v1]
	EOM

	run_dodder show one/dos+
	assert_success
	assert_output - <<-EOM
		[one/dos @blake2b256-fm7kce7793j3npevpm29spk04r6ycxv38dvx3hjxlzl8tcm5m3qq2mml86 !md "zettel with multiple etiketten" this_is_the_first this_is_the_second]
	EOM

	run_dodder show !md:t
	assert_success
	assert_output - <<-EOM
		[!md @blake2b256-3kj7xgch6rjkq64aa36pnjtn9mdnl89k8pdhtlh33cjfpzy8ek4qnufx0m !toml-type-v1]
	EOM

	run_dodder show !task:t
	assert_success
	assert_output - <<-EOM
		[!task @blake2b256-qxzg22c3axe9m42tpwqd4usnfag4elp20q7zvnkgmyea4f4rwcwsurfp5e !toml-type-v1]
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

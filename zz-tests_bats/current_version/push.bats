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

# bats file_tags=user_story:pull,user_story:repo,user_store:xdg,user_story:remote

function bootstrap_with_content {
  (
    mkdir -p "$1"
    pushd "$1" || exit 1
    run_dodder_init

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
  )
}

function bootstrap_without_content_xdg {
  mkdir -p them || exit 1
  pushd them || exit 1
  run_dodder_init
  assert_success
  popd || exit 1
}

function bootstrap_without_content {
  mkdir -p them || exit 1

  pushd them || exit 1
  run_dodder_init
  assert_success
  popd || exit 1
}

function bootstrap_archive {
  mkdir -p them || exit 1

  pushd them || exit 1
  run_dodder init \
    -repo-type archive \
    .default

  run_dodder info-repo type
  assert_success
  assert_output 'archive'

  assert_success
  popd || exit 1
}

function push_history_zettel_type_tag_no_conflicts { # @test
  (
    mkdir -p them
    pushd them || exit 1
    run_dodder_init
  )

  run_dodder remote-add \
    toml-repo-local_override_path-v0 \
    "$(realpath them)" \
    them
  assert_success
  assert_output_unsorted --regexp - <<-'EOM'
		\[/them @blake2b256-.+ !toml-repo-local_override_path-v0]
	EOM

  run_dodder push /them:k +zettel,typ,etikett

  assert_success
  assert_output_unsorted - <<-EOM
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
		[one/uno @blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 !md "wow ok" tag-1 tag-2]
		copied Blob blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd (10 B)
		copied Blob blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd (16 B)
		copied Blob blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 (27 B)
	EOM

  (
    pushd them || exit 1
    run_dodder show +zettel,typ,etikett,repo
    assert_golden_unsorted push_no_conflicts_them
  )
}

function push_history_zettel_type_tag_yes_conflicts { # @test
  bootstrap_with_content them

  run_dodder remote-add \
    toml-repo-local_override_path-v0 \
    "$(realpath them)" \
    them

  run_dodder push /them +zettel,typ,etikett

  assert_failure
  assert_output_unsorted - <<-EOM
		       conflicted [one/dos]
		       conflicted [one/uno]
		       conflicted [one/uno]
		copied Blob blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd (10 B)
		copied Blob blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd (16 B)
		copied Blob blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 (27 B)
		import failed with conflicts, merging required
	EOM

  (
    pushd them || exit 1
    run_dodder_init_workspace

    run_dodder status .
    assert_output_unsorted - <<-EOM
			       conflicted [one/dos]
			       conflicted [one/uno]
			        untracked [to_add @blake2b256-45lpe4rm9mjvdx8pt04kp5gh04uy77h0m0xtw2fhr0q7vl98g0vqls6hxe]
		EOM
  )

  run_dodder show +zettel,typ,etikett
  assert_golden_unsorted push_yes_conflicts_show
}

function push_history_default { # @test
  bootstrap_without_content_xdg

  run_dodder remote-add \
    toml-repo-local_override_path-v0 \
    "$(realpath them)" \
    them

  assert_success
  assert_output_unsorted --regexp - <<-'EOM'
		\[/them @blake2b256-.+ !toml-repo-local_override_path-v0]
	EOM

  run_dodder push /them

  assert_success

  run_dodder show +?z,e,t
  assert_golden_unsorted push_history_default_local

  pushd them || exit 1
  run_dodder show +zettel,typ,etikett #,repo
  assert_golden_unsorted push_history_default_them
}

function push_history_default_only_blobs { # @test
  bootstrap_without_content_xdg

  run_dodder remote-add \
    toml-repo-local_override_path-v0 \
    "$(realpath them)" \
    them

  assert_success
  assert_output_unsorted --regexp - <<-'EOM'
		\[/them @blake2b256-.+ !toml-repo-local_override_path-v0]
	EOM

  run_dodder push -exclude-objects /them

  assert_success
  assert_output_unsorted --regexp - <<-'EOM'
		copied Blob .* \(.*)
		copied Blob .* \(.*)
		copied Blob .* \(.*)
		copied Blob .* \(.*)
		copied Blob .* \(.*)
		copied Blob .* \(.*)
		copied Blob .* \(.*)
	EOM

  run_dodder show +?z,e,t
  assert_golden_unsorted push_default_only_blobs_local

  pushd them || exit 1
  run_dodder show +zettel,typ,etikett,repo
  assert_golden_unsorted push_default_only_blobs_them
}

function push_default_stdio_local_once { # @test
  bootstrap_without_content

  run_dodder remote-add \
    toml-repo-local_override_path-v0 \
    them \
    them
  assert_success
  assert_output_unsorted --regexp - <<-'EOM'
		\[/them @blake2b256-.+ !toml-repo-local_override_path-v0]
	EOM

  # 	run_dodder show -format text /them
  # 	assert_success
  # 	assert_output ''

  run_dodder push /them
  assert_success
  assert_output_unsorted --regexp - <<-'EOM'
		\[/them @blake2b256-.+ !toml-repo-local_override_path-v0]
	EOM

  pushd them || exit 1
  run_dodder show :zettel
  assert_success
  assert_output_unsorted - <<-EOM
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM
}

function push_history_default_stdio_local_twice { # @test
  bootstrap_without_content

  run_dodder remote-add \
    toml-repo-local_override_path-v0 \
    them \
    them

  assert_success
  assert_output_unsorted --regexp - <<-'EOM'
		\[/them @blake2b256-.+ !toml-repo-local_override_path-v0]
	EOM

  run_dodder push /them :z
  assert_success
  assert_output_unsorted --partial - <<-EOM
		copied Blob blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd (10 B)
		copied Blob blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd (16 B)
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM

  pushd them || exit 1
  run_dodder show :zettel
  assert_success
  assert_output_unsorted - <<-EOM
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM
  popd || exit 1

  run_dodder push /them :z

  assert_success
  assert_output_unsorted - <<-EOM
	EOM
}

# bats test_tags=user_story:integrity
function push_validates_blob_digest { # @test
  # Verify that the push flow includes blob digest validation by
  # confirming a normal push succeeds (the client computes and sends
  # the digest, the server validates it).
  bootstrap_without_content

  run_dodder remote-add \
    toml-repo-local_override_path-v0 \
    them \
    them
  assert_success

  run_dodder push /them
  assert_success

  pushd them || exit 1
  run_dodder show :zettel
  assert_success
  assert_output_unsorted - <<-EOM
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM
}

function push_history_default_stdio_twice { # @test
  bootstrap_without_content

  run_dodder remote-add \
    toml-repo-local_override_path-v0 \
    them \
    them

  assert_success
  assert_output_unsorted --regexp - <<-'EOM'
		\[/them @blake2b256-.+ !toml-repo-local_override_path-v0]
	EOM

  run_dodder push /them

  assert_success
  assert_line --regexp '\[/them @blake2b256-.+ !toml-repo-local_override_path-v0]'
  assert_line --regexp '\[[0-9]+\.[0-9]+ @blake2b256-.+ !inventory_list-v2]'
  assert_line --partial '[one/dos @blake2b256-'
  assert_line --partial '[one/uno @blake2b256-'
  assert_line --regexp 'copied Blob blake2b256-.+ \(.*\)'

  pushd them || exit 1
  run_dodder show +z,e,t,b
  assert_success
  assert_output_unsorted --regexp - <<-'EOM'
		\[[0-9]+\.[0-9]+ @blake2b256-.+ !inventory_list-v2]
		\[[0-9]+\.[0-9]+ @blake2b256-.+ !inventory_list-v2]
		\[[0-9]+\.[0-9]+ @blake2b256-.+ !inventory_list-v2]
		\[[0-9]+\.[0-9]+ @blake2b256-.+ !inventory_list-v2]
		\[[0-9]+\.[0-9]+ @blake2b256-.+ !inventory_list-v2]
		\[[0-9]+\.[0-9]+ @blake2b256-.+ !inventory_list-v2]
		\[[0-9]+\.[0-9]+ @blake2b256-.+ !inventory_list-v2]
		\[!md @blake2b256-e3ew5ma0s399rmk3akms90ah2kdmr88l4jluckmdqylnlqtzu7dq60533j !toml-type-v2 .+]
		\[!pandoc-defaults @blake2b256-zcfmrghzp36r4r4qxtrh4t8xcd5g0f3mkpm8f3swac0vr5x503msyfsu3d !toml-type-v2]
		\[!pandoc-lua_filter @blake2b256-afnd989ttt3vmeunlj2asss5hjtkqe75vhupupuz2y9uv8wfx8hs6q8szw !toml-type-v2]
		\[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		\[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
		\[one/uno @blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 !md "wow ok" tag-1 tag-2]
	EOM
  popd || exit 1

  run_dodder push /them
  assert_success
  assert_output_unsorted ''
}

function push_direct_local_path_no_conflicts { # @test
  (
    mkdir -p them
    pushd them || exit 1
    run_dodder_init
  )

  run_dodder push -direct "$(realpath them)" +zettel,typ,etikett

  assert_success
  assert_output_unsorted - <<-EOM
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
		[one/uno @blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 !md "wow ok" tag-1 tag-2]
		copied Blob blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd (10 B)
		copied Blob blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd (16 B)
		copied Blob blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 (27 B)
	EOM

  (
    pushd them || exit 1
    run_dodder show +zettel,typ,etikett
    assert_golden_unsorted push_direct_local_them
  )
}

function push_direct_no_repo_at_path { # @test
  mkdir -p empty_dir

  run_dodder push -direct "$(realpath empty_dir)" +zettel

  assert_failure
  assert_output --partial 'not in a dodder directory'
}

# #291: a blobless type definition (a type object with a null blob digest) is a
# documented "skip" during transfer, but the skip error was propagated as fatal
# on the default (continue-on-error=false) path, aborting the whole push. The
# transfer should succeed, skipping the blobless type and transferring the rest.
function push_skips_blobless_type_definition { # @test
  bootstrap_without_content_xdg

  # A normal zettel that must transfer.
  run_dodder new -edit=false - <<-EOM
		---
		# normal zettel
		- tag
		! md
		---

		body
	EOM
  assert_success

  # A blobless type definition: `new -object-id '!task'` with no -blob
  # commits a type object with a null blob digest (shown without @digest).
  run_dodder new -edit=false -object-id '!task'
  assert_success
  assert_output - <<-EOM
		[!task !toml-type-v2]
	EOM

  run_dodder remote-add \
    toml-repo-local_override_path-v0 \
    "$(realpath them)" \
    them
  assert_success

  # Default push (continue-on-error=false) must NOT abort on the blobless
  # type; it is skipped and a notice is emitted on stderr.
  run_dodder push /them
  assert_success

  # The parent received the fixture content + the normal zettel and the !md
  # type, but NOT the skipped blobless !task type. Content-addressed digests
  # are deterministic, so assert the exact set.
  pushd them || exit 1
  run_dodder show +zettel,typ,etikett
  assert_success
  assert_golden_unsorted push_fast_forward_them
  refute_output --partial '!task'
}

# #291: with -forbid-blobless-types, a blobless type definition is fatal again
# (opt-in strict mode), so the push aborts.
function push_forbid_blobless_type_definition_aborts { # @test
  bootstrap_without_content_xdg

  run_dodder new -edit=false -object-id '!task'
  assert_success

  run_dodder remote-add \
    toml-repo-local_override_path-v0 \
    "$(realpath them)" \
    them
  assert_success

  run_dodder push -forbid-blobless-types /them
  assert_failure
  assert_output --partial 'blobless type definition skipped'
}

# #298: pushing a clean, linear, single-author descendant to a parent that
# already holds an older ancestor of the local head must fast-forward, not
# raise a false "import failed with conflicts, merging required". The parent's
# older head is on the path to the local head, so there is no real divergence.
function push_fast_forward_linear_no_conflict { # @test
  run_dodder_init_workspace

  bootstrap_without_content_xdg

  run_dodder remote-add \
    toml-repo-local_override_path-v0 \
    "$(realpath them)" \
    them
  assert_success

  # First push syncs the parent to the current local head of one/uno.
  run_dodder push /them one/uno
  assert_success

  # Edit one/uno locally, producing a strict linear descendant (v_next).
  run_dodder checkout one/uno
  assert_success

  {
    echo "---"
    echo "# wow the second"
    echo "- tag-3"
    echo "- tag-4"
    echo "! md"
    echo "---"
    echo
    echo "edited locally"
  } >one/uno.zettel

  run_dodder checkin -delete one/uno.zettel
  assert_success

  # Second push must be a clean fast-forward, not a conflict.
  run_dodder push /them one/uno
  assert_success

  pushd them || exit 1
  run_dodder show one/uno
  assert_success
  assert_output --regexp '\[one/uno @blake2b256-.+ !md "wow the second" tag-3 tag-4\]'
  popd || exit 1
}

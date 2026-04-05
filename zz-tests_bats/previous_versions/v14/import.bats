#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

  # for shellcheck SC2154
  export output

  set_xdg "$BATS_TEST_TMPDIR"

  # Create a user-scoped blob store accessible by all repos
  run_dodder blob_store-init shared
  assert_success

  # Init outer repo using the shared user-scoped store
  run_dodder init \
    -yin <(cat_yin) \
    -yang <(cat_yang) \
    -lock-internal-files=false \
    -repo_id . \
    -encryption none \
    -blob_store-id shared \
    test
  assert_success

  run_dodder init-workspace -experimental-repo=false

  create_test_zettels
}

teardown() {
  chflags_nouchg
}

function import { # @test
  (
    mkdir inner
    pushd inner || exit 1
    run_dodder_init
    popd || exit 1
  )

  run_dodder export -print-time=true +z,e,t
  assert_success
  echo "$output" >list

  list="$(realpath list)"
  pushd inner || exit 1

  run_dodder info-repo pubkey
  assert_success
  new_pubkey="$output"

  run_dodder import \
    -blob_store-id shared \
    "$list"
  assert_success

  run_dodder show -format inventory_list +z,e,t
  assert_success
  assert_output_unsorted --regexp - <<-EOM
		\\[!md @$(get_type_blob_sha) .* !toml-type-v1]
		\\[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd .* $new_pubkey .* !md@.* "wow ok again" tag-3 tag-4]
		\\[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd .* $new_pubkey .* !md@.* "wow the first" tag-3 tag-4]
		\\[one/uno @blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 .* $new_pubkey .* !md@.* "wow ok" tag-1 tag-2]
	EOM

  run_dodder show one/uno
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM
}

function import_with_overwrite_sig { # @test
  (
    mkdir inner
    pushd inner || exit 1
    run_dodder_init
    popd || exit 1
  )

  cat >list <<-EOM
		---
		! inventory_list-v2
		---

		[!md @blake2b256-45v3c002j9xfjguu2a7ljxnf68tqglg8fa0csjgnn7d2n36ltp0snfjxgj 2135591162.342034946 dodder-repo-public_key-v1@ed25519_pub-vhhh5p6qfc9q5fpqm2xmjmetgnagmjpxxqlwlac4uvrhrvjvgevsv5z5q6 dodder-object-sig-v1@ed25519_sig-anhgqrkdqnn6uzvcaj93hr7epr72v8vefv0gkrhd7ktskl6pez2cr8kwe3krrndw8lefh8a7k5dzhete4pjk72zfp4vgf8f0srpksqsy6nn8g !toml-type-v1]
		[one/uno @blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 2135591162.520209927 dodder-repo-public_key-v1@ed25519_pub-vhhh5p6qfc9q5fpqm2xmjmetgnagmjpxxqlwlac4uvrhrvjvgevsv5z5q6 dodder-object-sig-v1@ed25519_sig-jr7jqjh6rq0zd42n03z5vcl2grqr3eg9eqwnuwxj809h3eaxqw58mm3garf4nzenptmu9mhamhtlt9uuxsrt5wl4dshsfsnak3zvgrcelwkhr !md "wow ok" tag-1 tag-2]
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd 2135591162.606407248 dodder-repo-public_key-v1@ed25519_pub-vhhh5p6qfc9q5fpqm2xmjmetgnagmjpxxqlwlac4uvrhrvjvgevsv5z5q6 dodder-object-sig-v1@ed25519_sig-3ya9fl5nlx7e77qk4vvx2ae7cez8uagywym8f2h5r6f4ern2fhslgtvqjge6fzxjwkkgfr9qjpt0kjjq6slzrm7phraq9jm4z42q2qqnnh2eu !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd 2135591162.697539117 dodder-repo-public_key-v1@ed25519_pub-vhhh5p6qfc9q5fpqm2xmjmetgnagmjpxxqlwlac4uvrhrvjvgevsv5z5q6 dodder-object-mother-sig-v1@ed25519_sig-jr7jqjh6rq0zd42n03z5vcl2grqr3eg9eqwnuwxj809h3eaxqw58mm3garf4nzenptmu9mhamhtlt9uuxsrt5wl4dshsfsnak3zvgrcelwkhr dodder-object-sig-v1@ed25519_sig-3ngs79lfywr6ewtdze0c9d3mwk824mymu8xjavzn3uc5s26fzwdy6mz487yasxhd2nqwefjuq3rtnfsj6a4p2u4dcj0wt2h4s2yl6qgm73qt6 !md "wow the first" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd 2135591162.697539118 dodder-repo-public_key-v1@ed25519_pub-vhhh5p6qfc9q5fpqm2xmjmetgnagmjpxxqlwlac4uvrhrvjvgevsv5z5q6 dodder-object-mother-sig-v1@ed25519_sig-jr7jqjh6rq0zd42n03z5vcl2grqr3eg9eqwnuwxj809h3eaxqw58mm3garf4nzenptmu9mhamhtlt9uuxsrt5wl4dshsfsnak3zvgrcelwkhr dodder-object-sig-v1@ed25519_sig-3ngs79lfywr6ewtdze0c9d3mwk824mymu8xjavzn3uc5s26fzwdy6mz487yasxhd2nqwefjuq3rtnfsj6a4p2u4dcj0wt2h4s2yl6qgm73qt6 !md "wow the first" tag-3 tag-4]
	EOM

  list="$(realpath list)"
  pushd inner || exit 1

  run_dodder info-repo pubkey
  assert_success
  new_pubkey="$output"

  run_dodder import \
    -overwrite-signatures=true \
    -blob_store-id shared \
    "$list"
  assert_success

  run_dodder show -format inventory_list +z,e,t
  assert_success
  assert_output_unsorted --regexp - <<-EOM
		\\[!md @$(get_type_blob_sha) .* !toml-type-v1]
		\\[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd .* $new_pubkey .* !md@.* "wow ok again" tag-3 tag-4]
		\\[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd .* $new_pubkey .* !md@.* "wow the first" tag-3 tag-4]
		\\[one/uno @blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 .* $new_pubkey .* !md@.* "wow ok" tag-1 tag-2]
	EOM

  run_dodder show one/uno
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM
}

# TODO add support for imports resolving signatures within the graph
function import_with_overwrite_sig_different_hash { # @test
  skip
  (
    mkdir inner
    pushd inner || exit 1
    run_dodder_init_sha256
  )

  (
    run_dodder_debug export -print-time=true +z,e,t >list
  )

  list="$(realpath list)"
  pushd inner || exit 1

  run_dodder info-repo pubkey
  assert_success
  new_pubkey="$output"

  run_dodder import \
    -overwrite-signatures=true \
    -blob_store-id shared \
    "$list"
  assert_success

  run_dodder show -format inventory_list +z,e,t
  assert_success
  assert_output_unsorted --regexp - <<-EOM
		\\[!md @sha256-.+ .* !toml-type-v1]
		\\[one/dos @sha256-95mv2p9mtaxxejqycc7fsvt55d3s8c0ptgazzgzgz4z7a3kvtujqa84qe8 .* $new_pubkey .* !md@.* "wow ok again" tag-3 tag-4]
		\\[one/uno @sha256-z8suqjv408y63y3x8dt83cwlexzusepm94aqa0wu7j7suq5ghsgs7dg4qc .* $new_pubkey .* !md@.* "wow the first" tag-3 tag-4]
		\\[one/uno @sha256-8259ya5jn9gmqvvy5quv5zkk0ja83tnzduhr2yzzdddp0ftdl92s6huu7d .* $new_pubkey .* !md@.* "wow ok" tag-1 tag-2]
	EOM

  run_dodder show one/uno
  assert_success
  assert_output - <<-EOM
		[one/uno @sha256-z8suqjv408y63y3x8dt83cwlexzusepm94aqa0wu7j7suq5ghsgs7dg4qc !md "wow the first" tag-3 tag-4]
	EOM

  run_dodder_debug show -format sig one/uno+

  run_dodder show -format sig-mother one/uno
  assert_success
  assert_output - <<-EOM
		[one/uno @sha256-8259ya5jn9gmqvvy5quv5zkk0ja83tnzduhr2yzzdddp0ftdl92s6huu7d !md "wow ok" tag-1 tag-2]
	EOM
}

function import_with_dupes_in_list { # @test
  (
    mkdir inner
    pushd inner || exit 1
    run_dodder_init
  )

  cat >list <<-EOM
		---
		! inventory_list-v2
		---

		[!md @blake2b256-45v3c002j9xfjguu2a7ljxnf68tqglg8fa0csjgnn7d2n36ltp0snfjxgj 2135591162.342034946 !toml-type-v1]
		[one/uno @blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 2135591162.520209927 !md "wow ok" tag-1 tag-2]
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd 2135591162.606407248 !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd 2135591162.697539117 !md "wow the first" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd 2135591162.697539118 !md "wow the first" tag-3 tag-4]
	EOM

  list="$(realpath list)"
  pushd inner || exit 1

  run_dodder info-repo pubkey
  assert_success
  new_pubkey="$output"

  run_dodder import \
    -overwrite-signatures=true \
    -blob_store-id shared \
    "$list"
  assert_success
  assert_output - <<-EOM
		copied Blob blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 (27 B)
		[one/uno @blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 !md "wow ok" tag-1 tag-2]
		copied Blob blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd (16 B)
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		copied Blob blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd (10 B)
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
		1 objects deduped during import
	EOM

  run_dodder show -format inventory_list +z,e,t
  assert_success
  assert_output_unsorted --regexp - <<-EOM
		\\[!md @$(get_type_blob_sha) .* !toml-type-v1]
		\\[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd .* $new_pubkey .* !md@.* "wow ok again" tag-3 tag-4]
		\\[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd .* $new_pubkey .* !md@.* "wow the first" tag-3 tag-4]
		\\[one/uno @blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 .* $new_pubkey .* !md@.* "wow ok" tag-1 tag-2]
	EOM

  run_dodder show one/uno
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM
}

function import_one_tai_same { # @test
  (
    mkdir inner
    pushd inner || exit 1
    run_dodder_init
  )

  run_dodder show -format tai one/uno
  tai="$output"

  run_dodder export -print-time=true one/uno [tag ^tag-1 ^tag-2]:e
  assert_success
  echo "$output" >list

  list="$(realpath list)"
  pushd inner || exit 1

  run_dodder import \
    -blob_store-id shared \
    "$list"

  assert_success
  assert_output_unsorted - <<-EOM
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
		copied Blob blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd (10 B)
	EOM

  run_dodder show one/uno
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM

  run_dodder show -format tai one/uno
  assert_success
  assert_output "$tai"
}

function import_twice_no_dupes_one_zettel { # @test
  (
    mkdir inner
    pushd inner || exit 1
    run_dodder_init
  )

  run_dodder export -print-time=true one/uno+
  assert_success
  echo "$output" >list

  list="$(realpath list)"
  pushd inner || exit 1

  run_dodder \
    import \
    -blob_store-id shared \
    "$list"
  assert_success
  assert_output_unsorted - <<-EOM
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
		[one/uno @blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 !md "wow ok" tag-1 tag-2]
		copied Blob blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd (10 B)
		copied Blob blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 (27 B)
	EOM

  run_dodder \
    import \
    -blob_store-id shared \
    "$list"
  assert_success
  assert_output - <<-EOM
	EOM

  run_dodder show :z,e,t
  assert_success
  assert_output_unsorted - <<-EOM
		[!md @$(get_type_blob_sha) !toml-type-v1]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM
}

# TODO add support for conflict resolution
function import_conflict { # @test
  (
    mkdir inner
    pushd inner || exit 1
    run_dodder_init
  )

  run_dodder export -print-time=true one/uno+
  assert_success
  echo "$output" >list

  list="$(realpath list)"
  pushd inner || exit 1

  run_dodder new -edit=false - <<-EOM
		---
		# get out of here!
		- scary
		! md
		---

		ouch a conflict!
	EOM
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-u20x7tfr58tc74p5y76xauwfrz382g96gfeenxvsaxaq6l3fnl2sntvzd5 !md "get out of here!" scary]
	EOM

  # Import accepts the remote versions directly when there is no parent
  # negotiator (imports are one-way; use pull for conflict detection).
  run_dodder import \
    -print-copies=false \
    -blob_store-id shared \
    "$list"
  assert_success
}

function import_twice_no_dupes { # @test
  (
    mkdir inner
    pushd inner || exit 1
    run_dodder_init
  )

  run_dodder export -print-time=true +z,e,t
  assert_success
  echo "$output" >list

  list="$(realpath list)"
  pushd inner || exit 1

  run_dodder import \
    -blob_store-id shared \
    "$list"
  assert_success
  assert_output_unsorted - <<-EOM
		copied Blob blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd (10 B)
		copied Blob blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd (16 B)
		copied Blob blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 (27 B)
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
		[one/uno @blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 !md "wow ok" tag-1 tag-2]
	EOM

  run_dodder show +z,e,t
  assert_success
  assert_output_unsorted - <<-EOM
		[!md @$(get_type_blob_sha) !toml-type-v1]
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
		[one/uno @blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 !md "wow ok" tag-1 tag-2]
	EOM

  run_dodder import \
    -blob_store-id shared \
    "$list"
  assert_success
  assert_output_unsorted - <<-EOM
	EOM

  run_dodder show :z,e,t
  assert_success
  assert_output_unsorted - <<-EOM
		[!md @$(get_type_blob_sha) !toml-type-v1]
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM

  run_dodder show +z,e,t
  assert_success
  assert_output_unsorted - <<-EOM
		[!md @blake2b256-45v3c002j9xfjguu2a7ljxnf68tqglg8fa0csjgnn7d2n36ltp0snfjxgj !toml-type-v1]
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
		[one/uno @blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 !md "wow ok" tag-1 tag-2]
	EOM
}

function import_reimport_no_local_checkout { # @test
  (
    mkdir inner
    pushd inner || exit 1
    run_dodder_init
    popd || exit 1
  )

  # Export the latest one/uno from outer
  run_dodder export -print-time=true one/uno
  assert_success
  echo "$output" >list1

  list1="$(realpath list1)"
  pushd inner || exit 1

  # First import: one/uno doesn't exist in inner, goes through importNewObject
  run_dodder import \
    -blob_store-id shared \
    "$list1"
  assert_success

  run_dodder show one/uno
  assert_success
  assert_output - <<-EOM
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
	EOM

  popd || exit 1

  # Modify one/uno in outer to create a new version
  run_dodder checkout one/uno
  assert_success

  cat >one/uno.zettel <<EOM
---
# updated for reimport
- tag-5
- tag-6
! md
---

reimported content
EOM

  run_dodder checkin -delete one/uno.zettel
  assert_success

  # Export the modified version
  run_dodder export -print-time=true one/uno
  assert_success
  echo "$output" >list2

  list2="$(realpath list2)"
  pushd inner || exit 1

  # Second import: one/uno exists in inner store but has no workspace checkout.
  # ReadOneInto finds the local copy, MergeCheckedOut is called on an object
  # with no workspace files — this triggers errInvalidCheckoutMode if the bug
  # is present.
  run_dodder import \
    -blob_store-id shared \
    "$list2"
  assert_success

  run_dodder show one/uno
  assert_success
  assert_output --regexp - <<-EOM
		\\[one/uno @blake2b256-.+ !md "updated for reimport" tag-5 tag-6]
	EOM

  popd || exit 1
}

function import_dry_run { # @test
  (
    mkdir inner
    pushd inner || exit 1
    run_dodder_init
  )

  run_dodder export -print-time=true +z,e,t
  assert_success
  echo "$output" >list

  list="$(realpath list)"
  pushd inner || exit 1

  run_dodder import \
    -dry-run \
    -blob_store-id shared \
    "$list"
  assert_success
  assert_output --partial "classification"
  assert_output --partial "committable"
  assert_output --partial "import"

  # Verify no objects were committed (inner should have only init objects)
  run_dodder show :z,e,t
  assert_success
  assert_output_unsorted - <<-EOM
		[!md @$(get_type_blob_sha) !toml-type-v1]
	EOM
}

function import_dry_run_objects_format { # @test
  (
    mkdir inner
    pushd inner || exit 1
    run_dodder_init
  )

  run_dodder export -print-time=true +z,e,t
  assert_success
  echo "$output" >list

  list="$(realpath list)"
  pushd inner || exit 1

  # First import to populate the inner repo
  run_dodder import \
    -blob_store-id shared \
    "$list"
  assert_success

  # Second dry-run to see skip-exists classifications
  run_dodder import \
    -dry-run \
    -plan-format objects \
    -blob_store-id shared \
    "$list"
  assert_success

  # Per-object output has tab-separated classification, genre, object, tai
  assert_output --partial "skip-exists"
  assert_output --partial "Type"
  assert_output --partial "Zettel"
}

function import_multiple_lists { # @test
  (
    mkdir inner
    pushd inner || exit 1
    run_dodder_init
  )

  # Export zettels into two separate lists
  run_dodder export -print-time=true one/uno+
  assert_success
  echo "$output" >list_a

  run_dodder export -print-time=true one/dos
  assert_success
  echo "$output" >list_b

  list_a="$(realpath list_a)"
  list_b="$(realpath list_b)"
  pushd inner || exit 1

  run_dodder import \
    -blob_store-id shared \
    "$list_a" "$list_b"
  assert_success

  # All objects from both lists should be present
  run_dodder show +z,e,t
  assert_success
  assert_output_unsorted - <<-EOM
		[!md @$(get_type_blob_sha) !toml-type-v1]
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
		[one/uno @blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 !md "wow ok" tag-1 tag-2]
	EOM
}

function import_multiple_lists_cross_dedup { # @test
  (
    mkdir inner
    pushd inner || exit 1
    run_dodder_init
  )

  # Export overlapping sets: both include one/uno (latest)
  run_dodder export -print-time=true one/uno one/dos
  assert_success
  echo "$output" >list_a

  run_dodder export -print-time=true one/uno
  assert_success
  echo "$output" >list_b

  list_a="$(realpath list_a)"
  list_b="$(realpath list_b)"
  pushd inner || exit 1

  run_dodder import \
    -blob_store-id shared \
    "$list_a" "$list_b"
  assert_success

  # The overlapping one/uno should be deduped
  assert_output --partial "deduped during import"
}

function import_tai_collision_resolution { # @test
  (
    mkdir inner
    pushd inner || exit 1
    run_dodder_init
  )

  # Craft a list with two objects sharing ObjectId+TAI but different blobs
  cat >list <<-EOM
		---
		! inventory_list-v2
		---

		[!md @blake2b256-45v3c002j9xfjguu2a7ljxnf68tqglg8fa0csjgnn7d2n36ltp0snfjxgj 2135591162.342034946 !toml-type-v1]
		[one/uno @blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 2135591162.520209927 !md "wow ok" tag-1 tag-2]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd 2135591162.520209927 !md "wow the first" tag-3 tag-4]
	EOM

  list="$(realpath list)"
  pushd inner || exit 1

  # Dry-run to verify collision is detected and resolved
  run_dodder import \
    -dry-run \
    -plan-format objects \
    -blob_store-id shared \
    "$list"
  assert_success
  assert_output --partial "resolve-tai-reassign"

  # Actual import: both objects should be committed
  run_dodder import \
    -overwrite-signatures=true \
    -blob_store-id shared \
    "$list"
  assert_success

  # Both versions of one/uno should exist with different TAIs
  run_dodder show -format inventory_list one/uno+
  assert_success

  line_count="$(echo "$output" | wc -l)"
  [[ "$line_count" -eq 2 ]]
}

function import_inventory_lists { # @test
  (
    mkdir inner
    pushd inner || exit 1
    run_dodder_init
  )

  run_dodder export -print-time=true
  assert_success
  echo "$output" >list

  list="$(realpath list)"
  pushd inner || exit 1

  export BATS_TEST_BODY=true
  run_dodder \
    import \
    -blob_store-id shared \
    "$list"

  assert_success

  run_dodder show +z,e,t
  assert_success
  assert_output_unsorted - <<-EOM
		[!md @$(get_type_blob_sha) !toml-type-v1]
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
		[one/uno @blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 !md "wow ok" tag-1 tag-2]
	EOM
}

function import_interactive_flag_accepted_non_tty { # @test
  (
    mkdir inner
    pushd inner || exit 1
    run_dodder_init
    popd || exit 1
  )

  run_dodder export -print-time=true +z,e,t
  assert_success
  echo "$output" >list

  list="$(realpath list)"
  pushd inner || exit 1

  # -interactive in a non-TTY (piped) context should fall through gracefully
  run_dodder import \
    -interactive \
    -blob_store-id shared \
    "$list"
  assert_success
}

function import_omit_tags { # @test
  (
    mkdir inner
    pushd inner || exit 1
    run_dodder_init
    popd || exit 1
  )

  # Export from outer repo (objects have tag-1, tag-2, tag-3, tag-4)
  run_dodder export -print-time=true +z,e,t
  assert_success
  echo "$output" >list

  list="$(realpath list)"
  pushd inner || exit 1

  # Import with -omit-tags stripping tag-1 and tag-2
  run_dodder import \
    -blob_store-id shared \
    -omit-tags "^tag-[12]$" \
    "$list"
  assert_success

  # Verify imported objects have tag-3 and tag-4 but not tag-1 or tag-2
  run_dodder show one/uno
  assert_success
  refute_output --partial "tag-1"
  refute_output --partial "tag-2"
  assert_output --partial "tag-3"
  assert_output --partial "tag-4"
}

function import_overwrite_sig_type_lock_consistency { # @test
  # Get the source (outer) repo's !md type signature
  run_dodder show -format inventory_list !md
  assert_success
  source_type_sig="$(echo "$output" | grep -oP 'dodder-object-sig-v\d+@\K\S+')"
  [ -n "$source_type_sig" ]

  (
    mkdir inner
    pushd inner || exit 1
    run_dodder_init
    popd || exit 1
  )

  run_dodder export -print-time=true +z,e,t
  assert_success
  echo "$output" >list

  list="$(realpath list)"
  pushd inner || exit 1

  run_dodder import \
    -overwrite-signatures=true \
    -blob_store-id shared \
    "$list"
  assert_success

  # After import with overwrite-sig, dependent objects should have type
  # locks pointing to the importing repo's !md signature, not the source's
  run_dodder show -format inventory_list one/uno
  assert_success
  # Type lock should be present
  assert_output --regexp '!md@ed25519_sig-'
  # Type lock should NOT reference the source repo's type signature
  refute_output --partial "!md@${source_type_sig}"
}

#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/common.bash"

  # for shellcheck SC2154
  export output

  copy_from_version "$DIR"
}

teardown() {
  chflags_and_rm
}

function import { # @test
  (
    mkdir inner
    pushd inner || exit 1
    run_dodder_init
    popd || exit 1
  )

  run_dodder info-repo pubkey
  assert_success
  # old_pubkey="$output"

  run_dodder export -print-time=true +z,e,t
  assert_success
  echo "$output" >list

  list="$(realpath list)"
  register_source_blob_store inner

  pushd inner || exit 1

  run_dodder info-repo pubkey
  assert_success
  new_pubkey="$output"

  run_dodder import \
    -blob_store-id .source \
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

  run_dodder info-repo pubkey
  assert_success
  # old_pubkey="$output"

  # run_dodder export -print-time=true +z,e,t
  cat >list <<-EOM
		---
		! inventory_list-v2
		---

		[!md @blake2b256-3kj7xgch6rjkq64aa36pnjtn9mdnl89k8pdhtlh33cjfpzy8ek4qnufx0m 2135591162.342034946 dodder-repo-public_key-v1@ed25519_pub-vhhh5p6qfc9q5fpqm2xmjmetgnagmjpxxqlwlac4uvrhrvjvgevsv5z5q6 dodder-object-sig-v1@ed25519_sig-anhgqrkdqnn6uzvcaj93hr7epr72v8vefv0gkrhd7ktskl6pez2cr8kwe3krrndw8lefh8a7k5dzhete4pjk72zfp4vgf8f0srpksqsy6nn8g !toml-type-v1]
		[one/uno @blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 2135591162.520209927 dodder-repo-public_key-v1@ed25519_pub-vhhh5p6qfc9q5fpqm2xmjmetgnagmjpxxqlwlac4uvrhrvjvgevsv5z5q6 dodder-object-sig-v1@ed25519_sig-jr7jqjh6rq0zd42n03z5vcl2grqr3eg9eqwnuwxj809h3eaxqw58mm3garf4nzenptmu9mhamhtlt9uuxsrt5wl4dshsfsnak3zvgrcelwkhr !md "wow ok" tag-1 tag-2]
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd 2135591162.606407248 dodder-repo-public_key-v1@ed25519_pub-vhhh5p6qfc9q5fpqm2xmjmetgnagmjpxxqlwlac4uvrhrvjvgevsv5z5q6 dodder-object-sig-v1@ed25519_sig-3ya9fl5nlx7e77qk4vvx2ae7cez8uagywym8f2h5r6f4ern2fhslgtvqjge6fzxjwkkgfr9qjpt0kjjq6slzrm7phraq9jm4z42q2qqnnh2eu !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd 2135591162.697539117 dodder-repo-public_key-v1@ed25519_pub-vhhh5p6qfc9q5fpqm2xmjmetgnagmjpxxqlwlac4uvrhrvjvgevsv5z5q6 dodder-object-mother-sig-v1@ed25519_sig-jr7jqjh6rq0zd42n03z5vcl2grqr3eg9eqwnuwxj809h3eaxqw58mm3garf4nzenptmu9mhamhtlt9uuxsrt5wl4dshsfsnak3zvgrcelwkhr dodder-object-sig-v1@ed25519_sig-3ngs79lfywr6ewtdze0c9d3mwk824mymu8xjavzn3uc5s26fzwdy6mz487yasxhd2nqwefjuq3rtnfsj6a4p2u4dcj0wt2h4s2yl6qgm73qt6 !md "wow the first" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd 2135591162.697539118 dodder-repo-public_key-v1@ed25519_pub-vhhh5p6qfc9q5fpqm2xmjmetgnagmjpxxqlwlac4uvrhrvjvgevsv5z5q6 dodder-object-mother-sig-v1@ed25519_sig-jr7jqjh6rq0zd42n03z5vcl2grqr3eg9eqwnuwxj809h3eaxqw58mm3garf4nzenptmu9mhamhtlt9uuxsrt5wl4dshsfsnak3zvgrcelwkhr dodder-object-sig-v1@ed25519_sig-3ngs79lfywr6ewtdze0c9d3mwk824mymu8xjavzn3uc5s26fzwdy6mz487yasxhd2nqwefjuq3rtnfsj6a4p2u4dcj0wt2h4s2yl6qgm73qt6 !md "wow the first" tag-3 tag-4]
	EOM

  list="$(realpath list)"
  register_source_blob_store inner

  pushd inner || exit 1

  run_dodder info-repo pubkey
  assert_success
  new_pubkey="$output"

  run_dodder import \
    -overwrite-signatures=true \
    -blob_store-id .source \
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
  register_source_blob_store inner

  pushd inner || exit 1

  run_dodder info-repo pubkey
  assert_success
  new_pubkey="$output"

  run_dodder import \
    -overwrite-signatures=true \
    -blob_store-id .source \
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

  run_dodder info-repo pubkey
  assert_success
  # old_pubkey="$output"

  # run_dodder export -print-time=true +z,e,t
  cat >list <<-EOM
		---
		! inventory_list-v2
		---

		[!md @blake2b256-3kj7xgch6rjkq64aa36pnjtn9mdnl89k8pdhtlh33cjfpzy8ek4qnufx0m 2135591162.342034946 !toml-type-v1]
		[one/uno @blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 2135591162.520209927 !md "wow ok" tag-1 tag-2]
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd 2135591162.606407248 !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd 2135591162.697539117 !md "wow the first" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd 2135591162.697539118 !md "wow the first" tag-3 tag-4]
	EOM

  list="$(realpath list)"
  register_source_blob_store inner

  pushd inner || exit 1

  run_dodder info-repo pubkey
  assert_success
  new_pubkey="$output"

  run_dodder import \
    -overwrite-signatures=true \
    -blob_store-id .source \
    "$list"
  assert_success
  assert_output - <<-EOM
		copied Blob blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 (27 B)
		[one/uno @blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 !md "wow ok" tag-1 tag-2]
		copied Blob blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd (16 B)
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		copied Blob blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd (10 B)
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
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
  register_source_blob_store inner

  pushd inner || exit 1

  run_dodder import \
    -blob_store-id .source \
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
  register_source_blob_store inner

  pushd inner || exit 1

  run_dodder \
    import \
    -blob_store-id .source \
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
    -blob_store-id .source \
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
  register_source_blob_store inner

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

  run_dodder import \
    -print-copies=false \
    -blob_store-id .source \
    "$list"
  assert_failure
  assert_output --partial - <<-EOM
		       conflicted [one/uno]
		       conflicted [one/uno]
	EOM

  assert_output --partial - <<-EOM
		import failed with conflicts, merging required
	EOM
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
  register_source_blob_store inner

  pushd inner || exit 1

  run_dodder import \
    -blob_store-id .source \
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
    -blob_store-id .source \
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
		[!md @blake2b256-3kj7xgch6rjkq64aa36pnjtn9mdnl89k8pdhtlh33cjfpzy8ek4qnufx0m !toml-type-v1]
		[one/dos @blake2b256-z3zpdf6uhqd3tx6nehjtvyjsjqelgyxfjkx46pq04l6qryxz4efs37xhkd !md "wow ok again" tag-3 tag-4]
		[one/uno @blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd !md "wow the first" tag-3 tag-4]
		[one/uno @blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0 !md "wow ok" tag-1 tag-2]
	EOM
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
  register_source_blob_store inner

  pushd inner || exit 1

  export BATS_TEST_BODY=true
  run_dodder \
    import \
    -blob_store-id .source \
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

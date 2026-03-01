#! /bin/bash -e

if [[ -z $BATS_TEST_TMPDIR ]]; then
  echo 'common.bash loaded before $BATS_TEST_TMPDIR set. aborting.' >&2

  cat >&2 <<-'EOM'
    only load this file from `.bats` files like so:

    setup() {
      load "$(dirname "$BATS_TEST_FILE")/common.bash"

      # for shellcheck SC2154
      export output
    }

    as there is a hard assumption on $BATS_TEST_TMPDIR being set
EOM

  exit 1
fi

pushd "$BATS_TEST_TMPDIR" >/dev/null || exit 1

bats_load_library "bats-support"
bats_load_library "bats-assert"
bats_load_library "bats-assert-additions"
bats_load_library "bats-island"

# get the containing directory of this file
# use $BATS_TEST_FILENAME instead of ${BASH_SOURCE[0]} or $0,
# as those will point to the bats executable's location or the preprocessed file respectively
DIR="$(cd "$(dirname "$BATS_TEST_FILENAME")" >/dev/null 2>&1 && pwd)"

cat_yin() (
  echo "one"
  echo "two"
  echo "three"
  echo "four"
  echo "five"
  echo "six"
)

cat_yang() (
  echo "uno"
  echo "dos"
  echo "tres"
  echo "quatro"
  echo "cinco"
  echo "seis"
)

cmd_dodder_def_no_debug=(
  -abbreviate-zettel-ids=false
  -abbreviate-shas=false
  -predictable-zettel-ids
  -print-types=false
  -print-time=false
  -print-tags=true
  -print-empty-shas=true
  -print-flush=false
  -print-unchanged=false
  -print-inventory_list=false
  -boxed-description=true
  -print-colors=false
)

export cmd_dodder_def_no_debug

cmd_dodder_def=(
  "${cmd_dodder_def_no_debug[@]}"
  -debug no-tempdir-cleanup
)

export cmd_dodder_def

DODDER_BIN=dodder

if [[ -z $DODDER_VERSION ]]; then
  export DODDER_VERSION
  DODDER_VERSION="v$("$DODDER_BIN" info store-version)"
fi

function copy_from_version {
  DIR="$1"

  rm -rf "$BATS_TEST_TMPDIR/.dodder"
  rm -rf "$BATS_TEST_TMPDIR/.madder"
  cp -r "$DIR/migration/$DODDER_VERSION/.dodder" "$BATS_TEST_TMPDIR/.dodder"
  cp -r "$DIR/migration/$DODDER_VERSION/.madder" "$BATS_TEST_TMPDIR/.madder"
}

function setup_repo {
  copy_from_version "$DIR" "$DODDER_VERSION"
}

function teardown_repo {
  chflags_and_rm
}

function run_dodder_debug {
  cmd="$1"
  shift
  #shellcheck disable=SC2068
  timeout --preserve-status "2s" "$DODDER_BIN" "$cmd" ${cmd_dodder_def[@]} "$@"
}

function run_dodder {
  cmd="$1"
  shift
  #shellcheck disable=SC2068
  run timeout --preserve-status "2s" "$DODDER_BIN" "$cmd" ${cmd_dodder_def[@]} "$@"
}

# TODO make this actually unify stderr
function run_dodder_stderr_unified {
  cmd="$1"
  shift
  #shellcheck disable=SC2068
  run "$DODDER_BIN" "$cmd" ${cmd_dodder_def[@]} "$@"
}

function run_dodder_init {
  if [[ $# -eq 0 ]]; then
    args=("test")
  else
    args=("$@")
  fi

  run_dodder init \
    -yin <(cat_yin) \
    -yang <(cat_yang) \
    -lock-internal-files=false \
    -override-xdg-with-cwd \
    "${args[@]}"

  assert_success
  assert_output - <<-EOM
[!md @$(get_type_blob_sha) !toml-type-v1]
[konfig @$(get_konfig_sha) !toml-config-v2]
EOM

  run_dodder_init_workspace
}

function run_dodder_init_sha256 {
  if [[ $# -eq 0 ]]; then
    args=("test")
  else
    args=("$@")
  fi

  run_dodder init \
    -yin <(cat_yin) \
    -yang <(cat_yang) \
    -lock-internal-files=false \
    -override-xdg-with-cwd \
    -hash_type-id sha256 \
    "${args[@]}"

  assert_success
  assert_output --regexp - <<-EOM
		\[!md @sha256-.+ !toml-type-v1]
		\[konfig @sha256-.+ !toml-config-v2]
	EOM
}

function run_dodder_init_workspace {
  run_dodder init-workspace
}

function get_konfig_sha() {
  echo -n "blake2b256-nzv90gppjdgn5jyjcal9chkau86ejdrp4ayr9mmfclaqqphuwrrsldsn3z"
}

function get_type_blob_sha() {
  echo -n "blake2b256-3kj7xgch6rjkq64aa36pnjtn9mdnl89k8pdhtlh33cjfpzy8ek4qnufx0m"
}

run_find() {
  run find . \
    -maxdepth 2 \
    ! -ipath './.dodder*' \
    ! -ipath './.madder*' \
    ! -iname '.dodder-workspace'
}

function run_dodder_init_disable_age_xdg {
  if [[ $# -eq 0 ]]; then
    args=("test-repo-id")
  else
    args=("$@")
  fi

  run_dodder init \
    -yin <(cat_yin) \
    -yang <(cat_yang) \
    -encryption none \
    -lock-internal-files=false \
    "${args[@]}"

  assert_success
  # assert_output - <<-EOM
  # [!md @$(get_type_blob_sha) !toml-type-v1]
  # [konfig @$(get_konfig_sha) !toml-config-v2]
  # EOM

  run_dodder blob_store-cat "$(get_konfig_sha)"
  assert_success
  assert_output

  run_dodder init-workspace
}

function run_dodder_init_disable_age {
  if [[ $# -eq 0 ]]; then
    args=("test-repo-id")
  else
    args=("$@")
  fi

  run_dodder init \
    -yin <(cat_yin) \
    -yang <(cat_yang) \
    -encryption none \
    -override-xdg-with-cwd \
    -lock-internal-files=false \
    "${args[@]}"

  assert_success
  assert_output - <<-EOM
[!md @$(get_type_blob_sha) !toml-type-v1]
[konfig @$(get_konfig_sha) !toml-config-v2]
EOM

  run_dodder blob_store-cat "$(get_konfig_sha)"
  assert_success
  assert_output

  run_dodder init-workspace
}

function create_test_zettels {
  export BATS_TEST_BODY=true
  run_dodder new -edit=false - <<EOM
---
# wow ok
- tag-1
- tag-2
! md
---

this is the body aiiiiight
EOM

  assert_success

  run_dodder new -edit=false - <<EOM
---
# wow ok again
- tag-3
- tag-4
! md
---

not another one
EOM

  assert_success

  run_dodder checkout one/uno
  assert_success

  cat >one/uno.zettel <<EOM
---
# wow the first
- tag-3
- tag-4
! md
---

last time
EOM

  run_dodder checkin -delete one/uno.zettel
  assert_success
}

function start_server {
  dir="$1"

  coproc server {
    if [[ -n $dir ]]; then
      cd "$dir"
    fi

    # shellcheck disable=SC2068
    dodder serve ${cmd_dodder_def[@]} tcp :0
  }

  # shellcheck disable=SC2154
  # trap 'kill $server_PID' EXIT

  read -r output <&"${server[0]}"

  if [[ $output =~ (starting HTTP server on port: \"([0-9]+)\") ]]; then
    export port="${BASH_REMATCH[2]}"
  else
    fail <<-EOM
			unable to get port info from dodder server.
			server output: $output
		EOM
  fi
}

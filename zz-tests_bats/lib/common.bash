#! /bin/bash -e

if [[ -z $BATS_TEST_TMPDIR ]]; then
  echo 'common.bash loaded before $BATS_TEST_TMPDIR set. aborting.' >&2

  cat >&2 <<-'EOM'
    only load this file from `.bats` files like so:

    setup() {
      load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

      # for shellcheck SC2154
      export output
    }

    as there is a hard assumption on $BATS_TEST_TMPDIR being set
EOM

  exit 1
fi

pushd "$BATS_TEST_TMPDIR" >/dev/null || exit 1

# Make the test self-sufficient inside a build sandbox where HOME
# defaults to /homeless-shelter and XDG_*_HOME inherits from the parent
# shell. Outside the sandbox this is still a strict tightening:
# BATS_TEST_TMPDIR is per-test, so dodder/madder can't accidentally
# read or write the developer's real ~/.config. Tests that need a
# specific HOME or XDG layout set their own values after sourcing.
export HOME="$BATS_TEST_TMPDIR"
export XDG_DATA_HOME="$BATS_TEST_TMPDIR/.xdg/data"
export XDG_CONFIG_HOME="$BATS_TEST_TMPDIR/.xdg/config"
export XDG_STATE_HOME="$BATS_TEST_TMPDIR/.xdg/state"
export XDG_CACHE_HOME="$BATS_TEST_TMPDIR/.xdg/cache"

bats_load_library "bats-support"
bats_load_library "bats-assert"
bats_load_library "bats-assert-additions"
bats_load_library "bats-island"
bats_load_library "bats-emo"

# get the test infrastructure root (zz-tests_bats/)
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." >/dev/null 2>&1 && pwd)"

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

require_bin DODDER_BIN dodder
DODDER_BIN="${DODDER_BIN:-dodder}"

require_bin DODDER_DER_BIN der
DODDER_DER_BIN="${DODDER_DER_BIN:-der}"

require_bin DODDER_TEST_SFTP_SERVER dodder-test-sftp-server
DODDER_TEST_SFTP_SERVER="${DODDER_TEST_SFTP_SERVER:-dodder-test-sftp-server}"

require_bin MADDER_BIN madder
MADDER_BIN="${MADDER_BIN:-madder}"

if [[ -z $DODDER_VERSION ]]; then
  export DODDER_VERSION
  DODDER_VERSION="v$("$DODDER_BIN" info store-version)"
fi

function copy_from_version {
  DIR_ARG="$1"

  rm -rf "$BATS_TEST_TMPDIR/.dodder"
  rm -rf "$BATS_TEST_TMPDIR/.madder"
  cp -r "$DIR_ARG/previous_versions/$DODDER_VERSION/.dodder" "$BATS_TEST_TMPDIR/.dodder"
  cp -r "$DIR_ARG/previous_versions/$DODDER_VERSION/.madder" "$BATS_TEST_TMPDIR/.madder"
}

function setup_repo {
  copy_from_version "$DIR" "$DODDER_VERSION"
}

function teardown_repo {
  chflags_nouchg
}

function run_dodder_debug {
  cmd="$1"
  shift
  #shellcheck disable=SC2068
  timeout --preserve-status "5s" "$DODDER_BIN" "$cmd" ${cmd_dodder_def[@]} "$@"
}

function run_dodder {
  cmd="$1"
  shift
  # Default ceiling = $PWD (NOT $BATS_TEST_TMPDIR) blocks the walk-up
  # above the caller's CWD. Under git-matching ceiling semantics (block
  # ABOVE the listed dir), this is what's needed so that a fresh-init
  # in a sub-CWD doesn't discover a fixture's `.dodder/` / `.madder/`
  # parked in $BATS_TEST_TMPDIR by setup. Without this, the fresh init
  # silently split-brains its writes across CWD's and the parent's
  # stores. See https://github.com/amarbel-llc/dodder/issues/40.
  #
  # `DODDER_CEILING_DIRECTORIES` itself is already set by the outer just
  # recipe (typically to /build or the worktree root) to bound the
  # dev-loop's walk-up, so we can't use `${VAR:-$PWD}` here -- it would
  # never default. Instead, tests that need walk-up (e.g. workspace
  # discovery from a sub-CWD) export `DODDER_TEST_CEILING` (or
  # `MADDER_TEST_CEILING`) -- variables the outer recipe leaves alone.
  #shellcheck disable=SC2068
  run env \
    DODDER_CEILING_DIRECTORIES="${DODDER_TEST_CEILING:-$PWD}" \
    MADDER_CEILING_DIRECTORIES="${MADDER_TEST_CEILING:-$PWD}" \
    timeout --preserve-status "5s" "$DODDER_BIN" "$cmd" ${cmd_dodder_def[@]} "$@"
}

function run_madder {
  cmd="$1"
  shift
  # See run_dodder for ceiling=$PWD rationale and the per-test override path.
  run env \
    DODDER_CEILING_DIRECTORIES="${DODDER_TEST_CEILING:-$PWD}" \
    MADDER_CEILING_DIRECTORIES="${MADDER_TEST_CEILING:-$PWD}" \
    XDG_LOG_HOME="$BATS_TEST_TMPDIR/.xdg/log" \
    timeout --preserve-status "5s" "$MADDER_BIN" "$cmd" "$@"
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
    -repo_id . \
    "${args[@]}"

  assert_success
  assert_output - <<-EOM
[!md @$(get_type_blob_sha) !toml-type-v2]
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
    -repo_id . \
    -hash_type-id sha256 \
    "${args[@]}"

  assert_success
  assert_output --regexp - <<-EOM
		\[!md @sha256-.+ !toml-type-v2]
		\[konfig @sha256-.+ !toml-config-v2]
	EOM
}

function run_dodder_init_workspace {
  run_dodder init-workspace -experimental-repo=false
}

# Source .fixtures.env for fixture-specific values
# shellcheck disable=SC1090
source "$DIR/previous_versions/$DODDER_VERSION/.fixtures.env"

function get_konfig_sha() { echo -n "$FIXTURE_KONFIG_SHA"; }
function get_type_blob_sha() { echo -n "$FIXTURE_TYPE_BLOB_SHA"; }
function get_fixture_type_sig() { echo -n "$FIXTURE_TYPE_SIG"; }

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
    "${args[@]}"

  assert_success
  # assert_output - <<-EOM
  # [!md @$(get_type_blob_sha) !toml-type-v2]
  # [konfig @$(get_konfig_sha) !toml-config-v2]
  # EOM

  # Sanity check: the freshly-written konfig blob is reachable
  # via madder. Confirms #151 bucket B's two-env composition
  # actually routes dodder XDG-init blob storage through madder's
  # XDG namespace (the gate that #144 phase 1 had to remove this
  # check for, via #159).
  #
  # Extract the konfig sha from init's output — the fixture's
  # FIXTURE_KONFIG_SHA can drift from what fresh init produces if
  # dodder's default toml changes between fixture regenerations.
  local konfig_sha
  konfig_sha="$(echo "$output" | grep -oE 'konfig @blake2b256-[[:alnum:]]+' | grep -oE 'blake2b256-[[:alnum:]]+' | head -1)"
  [[ -n $konfig_sha ]] || fail "could not extract konfig sha from init output: $output"

  run_madder cat default "$konfig_sha"
  assert_success
  assert_output

  run_dodder init-workspace -experimental-repo=false
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
    -repo_id . \
    "${args[@]}"

  assert_success
  assert_output --regexp - <<-EOM
\[!md @blake2b256-.+ !toml-type-v2]
\[konfig @blake2b256-.+ !toml-config-v2]
EOM

  run_dodder init-workspace -experimental-repo=false
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
    "$DODDER_BIN" serve ${cmd_dodder_def[@]} -handshake
  }

  # Wait up to 5s for the handshake line. dodder serve -handshake binds
  # to 127.0.0.1:0, then writes a single pipe-delimited line to stdout
  # with the OS-assigned port:
  #   1|1|tcp|127.0.0.1:PORT|dodder-http-v1
  local line
  if ! IFS= read -r -t 5 -u "${server[0]}" line; then
    fail <<-EOM
			no handshake from dodder serve within 5s.
			server pid: ${server_PID:-unknown}
		EOM
  fi

  # 1|1|tcp|127.0.0.1:PORT|dodder-http-v1
  local _core _app _net addr _proto
  IFS='|' read -r _core _app _net addr _proto <<<"$line"

  if [[ -z $addr ]]; then
    fail <<-EOM
			could not parse handshake line from dodder serve.
			line: $line
		EOM
  fi

  # Export the address (host:port) and just the port for callers.
  # shellcheck disable=SC2154
  export server_addr="$addr"
  export port="${addr##*:}"
}

# stop_server tears down the coproc started by start_server. Solves the
# subprocess-leak problem flagged in clone.bats:199's TODO. Safe to
# call multiple times and from teardown.
function stop_server {
  if [[ -n ${server_PID:-} ]]; then
    kill "$server_PID" 2>/dev/null || true
    wait "$server_PID" 2>/dev/null || true
    unset server_PID
  fi
  unset port server_addr
}

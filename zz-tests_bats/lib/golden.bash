# SC2154: bats injects $output and the $BATS_* vars into the test scope at
# runtime; they are not assigned in this helper.
# shellcheck disable=SC2154

# Golden-file (approval-testing) assertions -- pilot replacing the
# .fixtures.env global-constant getters for genesis-snapshot assertions.
# Regenerate with: just test-bats-update-goldens <files>  (sets
# DODDER_UPDATE_GOLDENS=1). Goldens are git-tracked so the whole-tree-staged
# nix lane can read them.

# golden_normalize masks the only non-deterministic tokens in dodder output:
# per-init ed25519 signatures + public keys. Content-addressed digests
# (blake2b256-/sha256-) stay VERBATIM so a wrong digest still fails.
golden_normalize() {
  sed -E \
    -e 's/ed25519_sig-[a-z0-9]+/ed25519_sig-<SIG>/g' \
    -e 's/ed25519_pub-[a-z0-9]+/ed25519_pub-<PUB>/g'
}

_golden_path() {
  local file_base
  file_base="$(basename "$BATS_TEST_FILENAME" .bats)"
  printf '%s' "$BATS_TEST_DIRNAME/goldens/$file_base/$1.txt"
}

# _assert_golden <name> <sort:0|1>
_assert_golden() {
  local name="$1" sort="$2" golden normalized expected
  golden="$(_golden_path "$name")"
  normalized="$(printf '%s\n' "$output" | golden_normalize)"
  [[ $sort == 1 ]] && normalized="$(printf '%s\n' "$normalized" | LC_ALL=C sort)"

  if [[ -n ${DODDER_UPDATE_GOLDENS:-} ]]; then
    mkdir -p "$(dirname "$golden")"
    printf '%s\n' "$normalized" >"$golden"
    return 0
  fi

  [[ -f $golden ]] || {
    fail "golden missing: $golden (run: just test-bats-update-goldens)"
    return 1
  }

  expected="$(cat "$golden")"
  [[ $sort == 1 ]] && expected="$(printf '%s\n' "$expected" | LC_ALL=C sort)"
  assert_equal "$normalized" "$expected"
}

# assert_golden <name>: exact-order golden compare of $output.
assert_golden() { _assert_golden "$1" 0; }
# assert_golden_unsorted <name>: line-order-agnostic (for query results that
# were previously asserted with assert_output_unsorted).
assert_golden_unsorted() { _assert_golden "$1" 1; }

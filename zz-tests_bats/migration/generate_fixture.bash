#! /bin/bash -e

v="$DODDER_VERSION"

if [[ -z "$v" ]]; then
  echo "no \$DODDER_VERSION set" >&2
  exit 1
fi

cmd_bats=(
  bats
  --no-tempdir-cleanup
  --tap
  migration/generate_fixture.bats
)

if ! bats_run="$(BATS_TEST_TIMEOUT=3 "${cmd_bats[@]}" 2>&1)"; then
  echo "$bats_run" >&2
  exit 1
else
  bats_dir="$(echo "$bats_run" | grep "BATS_RUN_TMPDIR" | cut -d' ' -f2)"
fi

dir_git_root="$(git rev-parse --show-toplevel)"
dir_base="$(realpath "$(dirname "$0")")"
d="${2:-$dir_base/$v}"

if [[ -d $d ]]; then
  "$dir_git_root/bin/chflags.bash" -R nouchg "$d"
  rm -rf "$d"
fi

mkdir -p "$d"
cp -r "$bats_dir/test/1/.dodder" "$d/.dodder"
cp -r "$bats_dir/test/1/.madder" "$d/.madder"

# Extract the type signature from the fixture for use in test assertions.
# The signature is fixture-specific (tied to the signing key generated during
# init) and changes each time fixtures are regenerated.
# Query the type object (:t), not a zettel — the ! md@<sig> line in text format
# uses the type's object signature. Strip the purpose prefix (dodder-object-sig-v2@)
# since the text format omits it.
type_sig=$(cd "$d" && dodder show -format sig :t 2>/dev/null)
type_sig="${type_sig#*@}"

if [[ -n "$type_sig" ]]; then
  echo "FIXTURE_TYPE_SIG=$type_sig" > "$d/.signature.env"
fi

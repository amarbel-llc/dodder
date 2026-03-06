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
  previous_versions/generate_fixture.bats
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

# Extract fixture-specific values for test assertions.
# Signature is tied to the signing key generated during init and changes each
# time fixtures are regenerated. Blob hashes are content-addressed and stable.
pushd "$d" >/dev/null

type_sig=$(dodder show -format type-sig one/uno 2>/dev/null)
konfig_sha=$(dodder show \
  -abbreviate-shas=false \
  -print-empty-shas=true \
  -format log :konfig 2>/dev/null \
  | sed 's/.*@\([^ ]*\) .*/\1/')
type_blob_sha=$(dodder show \
  -abbreviate-shas=false \
  -print-empty-shas=true \
  -format log '!md:t' 2>/dev/null \
  | sed 's/.*@\([^ ]*\) .*/\1/')

cat > .fixtures.env <<EOF
# Auto-generated during fixture generation -- do not edit
FIXTURE_TYPE_SIG=$type_sig
FIXTURE_KONFIG_SHA=$konfig_sha
FIXTURE_TYPE_BLOB_SHA=$type_blob_sha
EOF

popd >/dev/null

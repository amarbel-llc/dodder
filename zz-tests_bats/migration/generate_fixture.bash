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

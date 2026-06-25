#! /usr/bin/env bash
set -e

tmp="$(mktemp)"
# shellcheck disable=SC2064
trap "rm -rf '$tmp'" EXIT

# rg replaces ag here. The pattern uses a backreference (\1), which rg's
# default Rust regex engine does not support, so -P (PCRE2) is required.
# -l = files-with-matches, --null = NUL-separated (ag's -l0).
cmd_rg=(
  rg
  -P
  -l
  --null
  '\b(\w+)\b "github.com/friedenberg/dodder/src/\w+/\1"'
)

if "${cmd_rg[@]}" >"$tmp"; then
  xargs -0 sed -E -i'' 's#(\w+) ("github.com/friedenberg/dodder/src/\w+/\1")#\2#g' <"$tmp"
fi

goimports -w ./

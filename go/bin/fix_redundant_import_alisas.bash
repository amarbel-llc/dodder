#! /usr/bin/env bash
set -e

tmp="$(mktemp)"
# shellcheck disable=SC2064
trap "rm -rf '$tmp'" EXIT

cmd_ag=(
  ag
  '\b(\w+)\b "github.com/friedenberg/dodder/src/\w+/\1"'
  -l0
)

if "${cmd_ag[@]}" >"$tmp"; then
  xargs -0 sed -E -i'' 's#(\w+) ("github.com/friedenberg/dodder/src/\w+/\1")#\2#g' <"$tmp"
fi

goimports -w ./

#! /usr/bin/env bash
set -xe

# TODO move this to justfile and simplify the semantics
old="$1"
shift
new="$1"
shift
super_package="$1"
shift
package="$1"
shift

# file_diff="$(mktemp)"

(
  go_refactor_args=(
    "-w"
    "-r"
    "$package.$old -> $package.$new"
  )

  gofmt "${go_refactor_args[@]}" src/
)

(
  go_refactor_args=(
    "-w"
    "-r"
    "$old -> $new"
  )

  gofmt "${go_refactor_args[@]}" "src/$super_package/$package"
)

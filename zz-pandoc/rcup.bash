#! /usr/bin/env bash
set -e

while IFS= read -r -d '' file; do
  name="$(basename "$file")"
  dir="$(dirname "$file")"
  dir_dst="$HOME/.local/share/pandoc/$dir"
  mkdir -p "$dir_dst"
  path_dst="$dir_dst/$name"
  ln -fs "$(realpath "$file")" "$path_dst"
done < <(find filters defaults -type f -print0)

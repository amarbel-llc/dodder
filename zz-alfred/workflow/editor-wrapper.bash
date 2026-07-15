#!/usr/bin/env bash
# Dispatch to nvim/vim, spawning a kitty window when invoked with no
# controlling TTY (e.g. an Alfred script action, which never has one).
# Reconstructs the behavior of the deleted eng home/local-bin/editor
# script that dodder's Alfred workflow's edit/new actions depend on --
# dodder's editor integration (go/lib/alfa/editor) never spawns a
# terminal itself, so this wrapper is what makes `der new` / `der open`
# open a visible editor window instead of exec'ing headless and exiting
# silently.
set -e

which_vim=vim
if command -v nvim >/dev/null 2>&1; then
  which_vim=nvim
fi

if ! tty >/dev/null 2>&1; then
  if command -v kitty >/dev/null 2>&1; then
    which_kitty=kitty
  else
    which_kitty="/Applications/Nix Apps/kitty.app/Contents/MacOS/kitty"
  fi
  exec "$which_kitty" "$which_vim" "$@"
else
  exec "$which_vim" "$@"
fi

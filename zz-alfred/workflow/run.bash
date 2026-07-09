#! /usr/bin/env bash
#
# Entry point every Alfred object in info.plist shells out to. It changes
# into the dodder workspace (so dodder resolves its cwd-ancestor .dodder
# scope) and execs the dodder binary with the object's arguments.
#
# @dodder@ and @workspace@ are placeholders the home-manager module
# rewrites at activation time (see hm-module.nix): @dodder@ becomes the
# dodder nix-store binary path, @workspace@ becomes the required
# `workspace` option. When left unsubstituted — e.g. running the raw
# workflow source in a dev loop, or importing the .alfredworkflow by hand
# — they fall back to the DODDER_BIN / DODDER_WORKSPACE env vars, then to
# `dodder` on PATH and $PWD. The `case` guards detect the still-literal
# placeholder so an un-templated copy stays runnable.
set -euo pipefail

# The guard sentinels are assembled at runtime around a lone '@' so the
# literal '@dodder@' / '@workspace@' tokens only ever appear whole in the
# placeholder assignments — otherwise `substitute --replace-fail` would
# rewrite the guards too, and a baked-in path could never be told apart
# from the un-templated source.
at='@'
dodder_bin='@dodder@'
if [ "$dodder_bin" = "${at}dodder${at}" ]; then
  dodder_bin="${DODDER_BIN:-dodder}"
fi

workspace='@workspace@'
if [ "$workspace" = "${at}workspace${at}" ]; then
  workspace="${DODDER_WORKSPACE:-$PWD}"
fi

cd "$workspace"
exec "$dodder_bin" "$@"

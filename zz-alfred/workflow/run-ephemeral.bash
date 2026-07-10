#! /usr/bin/env bash
#
# Ephemeral entry point for the write actions (edit / new). Unlike run.bash —
# which cd's into a single workspace and relies on cwd-ancestor .dodder scope
# resolution — this runs the dodder subcommand with `-ephemeral -parent
# <workspace>`, so dodder spins a throwaway repo-backed workspace against the
# resolved PARENT repo, applies the change, pushes it back, and tears the temp
# workspace down (FDR-0023). No persistent workspace / cwd is required, which
# is what lets a launcher edit or create objects from anywhere.
#
# Usage: run-ephemeral.bash <subcommand> [args...]
#   e.g. run-ephemeral.bash edit -mode both "$1"
#        run-ephemeral.bash new  -edit=true -description "$*"
#
# @dodder@ and @workspace@ are the same placeholders run.bash uses (see its
# header): @dodder@ is baked to the dodder nix-store binary by the package;
# @workspace@ is baked to the parent repo path by the home-manager module.
# Here @workspace@ is passed as `-parent`, NOT cd'd into. Unsubstituted, they
# fall back to DODDER_BIN / DODDER_WORKSPACE then `dodder` on PATH; an unset
# workspace fallback omits -parent so dodder targets the home repo.
set -euo pipefail

at='@'
dodder_bin='@dodder@'
if [ "$dodder_bin" = "${at}dodder${at}" ]; then
  dodder_bin="${DODDER_BIN:-dodder}"
fi

workspace='@workspace@'
if [ "$workspace" = "${at}workspace${at}" ]; then
  workspace="${DODDER_WORKSPACE:-}"
fi

subcommand="$1"
shift

# -parent is optional: omit it (targeting the home repo) when no workspace is
# configured, otherwise point the ephemeral workspace at the configured parent.
if [ -n "$workspace" ]; then
  exec "$dodder_bin" "$subcommand" -ephemeral -parent "$workspace" "$@"
else
  exec "$dodder_bin" "$subcommand" -ephemeral "$@"
fi

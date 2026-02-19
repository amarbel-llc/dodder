#!/usr/bin/env bash
set -e

srt_config="$(mktemp)"
trap 'rm -f "$srt_config"' EXIT

cat >"$srt_config" <<SETTINGS
{
  "filesystem": {
    "denyRead": [
      "$HOME/.ssh",
      "$HOME/.aws",
      "$HOME/.gnupg",
      "$HOME/.config",
      "$HOME/.local",
      "$HOME/.password-store",
      "$HOME/.kube"
    ],
    "denyWrite": [],
    "allowWrite": [
      "/tmp"
    ]
  },
  "network": {
    "allowedDomains": [],
    "deniedDomains": []
  }
}
SETTINGS

export GIT_CONFIG_GLOBAL=/dev/null
export GIT_CONFIG_SYSTEM=/dev/null

exec sandcastle \
  --shell bash \
  --config "$srt_config" \
  "$@"

#! /bin/bash -e

if [[ "$(uname -s)" != "Darwin" ]]; then
  exit 0
fi

chflags "$@"

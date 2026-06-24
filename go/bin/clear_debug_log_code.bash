#! /usr/bin/env bash
set -e

ag "log.Debug" -l src/ | xargs sed -i '' '/log.Debug/d'
goimports -w ./

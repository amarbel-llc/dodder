#! /usr/bin/env bash
set -e

ag "github.com/friedenberg/dodder/\w+" "$@" -o --nofile --nocolor --nogroup | sort -u

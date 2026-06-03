#! /bin/bash -e

da_test="$1"
shift

if output="$(bats --tap "$da_test")"; then
  echo "$output"
  exit 0
fi

echo "$output" | grep -Pzo "(?s)(?<=\n)not ok.*?(?=\nok)"

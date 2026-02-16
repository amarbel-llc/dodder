#! /usr/bin/env bash -e

echo "1..$(ag '^do_lint \\' "$0" | wc -l | tr -d ' ')"

do_lint() (
	msg="$1"
	shift
	cmd=("$@")

	if [[ "$(($("${cmd[@]}" | wc -l)))" -gt 0 ]]; then
		echo "not ok $msg" >&2
		"${cmd[@]}"
		echo
		echo "cmd: " "$(echo -n "${cmd[@]@Q}")" >&2
		exit 1
	fi

	echo "ok $msg" >&2
)

do_lint \
	"todos without priorities" \
	ag \
	--go \
	'// todo(?!(\s*|-)p)' \
	--ignore-case \
	-l

# do_lint \
# 	"bats test files have skips" \
# 	ag \
# 	skip \
# 	zz-tests_bats/ \
# 	-l

do_lint \
	"debug logs remaining" \
	ag \
	"^\s*log.Debug\\b" \
	-l \
	src/

do_lint \
	"P0 comment todos" \
	ag \
	--go \
	'// todo(\s*|-)p0' \
	--ignore-case \
	-l

do_lint \
	"P0 compiled todos" \
	ag \
	--go \
	'errors\.TodoP0' \
	--ignore-case \
	-l

do_lint \
	"no root packages" \
  find src -mindepth 2 -maxdepth 2 -type f

# TODO-P2 enable after annotating existing intentional discards with //repool:owned
# do_lint \
# 	"no new discarded repool functions" \
# 	bash -c 'ag --go ", _ :=.*(GetWithRepool|catgut\.Make)" src/ | grep -v "//repool:owned"'

#! /usr/bin/env bats

setup() {
	load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

	# for shellcheck SC2154
	export output
}

teardown() {
	chflags_nouchg
}

# bats file_tags=user_story:zettel_ids,format

# The zettel_id_log is an append-only log of provider word-list mutations
# (yin / yang inits and add-zettel-ids-* operations). The on-disk shape
# should mirror the existing inventory_lists_log: exactly one hyphence
# header at the top of the file (`---\n! <type>\n---\n`), then a blank
# line, then one entry per line (or per record) appended below.
#
# The bug today is that go/internal/delta/zettel_id_log/log.go's
# AppendEntry wraps each entry in a fresh hyphence.TypedBlob and emits
# `Coder.EncodeTo` for it, so every append re-writes the type header.
# The resulting file is stacked hyphence docs, not header + body. The
# reader (segmentEntries in the same file) compensates by detecting
# every odd boundary as a new entry, but the wire shape is wrong.
#
# These tests pin the *desired* shape: exactly one `! zettel_id_log-*`
# line in the whole file regardless of entry count. They fail today and
# pass when the writer is reshaped to "emit header on file creation,
# then append bodies only" (or equivalent).
#
# Companion test: inventory_lists_log_has_single_header is the
# inverse — it pins the working contract of the inventory log so we
# notice if it ever drifts to the broken shape.

function zettel_id_log_path {
	echo "$PWD/.dodder/local/share/zettel_id_log"
}

function inventory_lists_log_path {
	echo "$PWD/.dodder/local/share/inventory_lists_log"
}

function zettel_id_log_has_single_header_after_init { # @test
	wd="$(mktemp -d)"
	cd "$wd" || exit 1

	run_dodder_init_disable_age

	path="$(zettel_id_log_path)"
	if [[ ! -f $path ]]; then
		fail <<-EOM
		expected zettel_id_log at $path

		directory listing under .dodder/local/share/:
		$(find .dodder/local/share -type f 2>&1 | sort)
		EOM
	fi

	# Init writes a yin entry and a yang entry. The broken writer emits
	# two full hyphence frames (two type headers). The desired shape has
	# exactly one type header at the top of the file.
	local header_count
	# `grep -c` exits nonzero on zero matches; tolerate that so we can
	# fail with a useful diagnostic instead of bailing here under set -e.
	header_count="$(grep -c '^! zettel_id_log' "$path" || true)"

	if [[ $header_count -ne 1 ]]; then
		fail <<-EOM
		zettel_id_log should have exactly 1 type-header line; got $header_count
		path: $path
		full content:
		$(cat "$path")
		EOM
	fi
}

function zettel_id_log_has_single_header_after_add_zettel_ids { # @test
	wd="$(mktemp -d)"
	cd "$wd" || exit 1

	run_dodder_init_disable_age

	# Append a third entry so the bug (re-emitted header per append) is
	# extra obvious: 3 entries -> 3 headers under the broken writer.
	run bash -c 'echo -e "alpha\nbravo" | '"$DODDER_BIN"' add-zettel-ids-yin'
	assert_success

	path="$(zettel_id_log_path)"
	local header_count
	# `grep -c` exits nonzero on zero matches; tolerate that so we can
	# fail with a useful diagnostic instead of bailing here under set -e.
	header_count="$(grep -c '^! zettel_id_log' "$path" || true)"

	if [[ $header_count -ne 1 ]]; then
		fail <<-EOM
		zettel_id_log should have exactly 1 type-header line after init + add-zettel-ids-yin; got $header_count
		path: $path
		full content:
		$(cat "$path")
		EOM
	fi
}

function inventory_lists_log_has_single_header { # @test
	# Inverse of the zettel_id_log tests: pin the inventory log's
	# already-correct shape so we notice if it ever regresses to the
	# stacked-doc format.
	wd="$(mktemp -d)"
	cd "$wd" || exit 1

	run_dodder_init_disable_age

	create_test_zettels

	path="$(inventory_lists_log_path)"
	if [[ ! -f $path ]]; then
		fail <<-EOM
		expected inventory_lists_log at $path

		directory listing under .dodder/local/share/:
		$(find .dodder/local/share -type f 2>&1 | sort)
		EOM
	fi

	local header_count
	header_count="$(grep -c '^! inventory_list' "$path")"

	if [[ $header_count -ne 1 ]]; then
		fail <<-EOM
		inventory_lists_log should have exactly 1 type-header line; got $header_count
		path: $path
		full content:
		$(cat "$path")
		EOM
	fi
}

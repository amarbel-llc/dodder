#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"

  # for shellcheck SC2154
  export output

  copy_from_version "$DIR"

  run_dodder_init_workspace
}

teardown() {
  chflags_nouchg
}

# bats file_tags=status

# These tests exercise `dodder status` against workspace contents whose
# relative path does NOT map to a valid zettel id. store_fs walks the
# workspace and parses each file's relative path (minus extension) as an
# object id via ids.ValidateSeqAndGetGenre, which accepts a single token
# (e.g. `foo` -> Tag) or the two-token `prefix/suffix` zettel shape but
# rejects three+ tokens (a file nested two or more directories deep).
# Such files must surface as untracked blobs, not abort the scan with
# "unsupported seq ... objectId3" (regression guard for
# store_fs/dir_info.go recognizeGenre).

# Single-level untracked blob at the workspace root: a single-token key.
function status_root_untracked_blob_contrast { # @test
  cat >top-level-note.md <<-EOM
		# a top-level note, not checked out
	EOM

  run_dodder status .
  assert_success
  assert_output '        untracked [top-level-note.md @blake2b256-0fe2llnlw0jzluvp6ee9yxnseqktx04c956fp6xevh9zwssegcfs6fs87x]'
}

# A file nested ONE directory deep is a `prefix/suffix` (two-token)
# zettel-id shape, which the validator accepts.
function status_single_level_dir_contrast { # @test
  mkdir -p subdir
  cat >subdir/note.md <<-EOM
		# a note one dir deep
	EOM

  run_dodder status .
  assert_success
  assert_output '        untracked [subdir/note.md @blake2b256-t49mhptv3h04xgetlzq2xfqnfavxrd52u4fv0e7vjzmn7hn0s8lqftq4v5]'
}

# A file nested TWO directories deep (docs/plans/<file>) yields a
# three-token seq the validator rejects; it must still surface as an
# untracked blob rather than crashing the scan.
function status_nested_non_zettel_path_dot { # @test
  mkdir -p docs/plans
  cat >docs/plans/2026-03-24-mcp-workspace-tools-design.md <<-EOM
		# a plan doc, not a zettel
	EOM

  run_dodder status .
  assert_success
  assert_output '        untracked [docs/plans/2026-03-24-mcp-workspace-tools-design.md @blake2b256-3dgzgz72kwlq0uh4nxxh5w2gnghzz7qzlg3w5rs4mrek5dfefk6qkjkwz9]'
}

# Same nested file addressed as an explicit target rather than via the
# dot scan.
function status_nested_non_zettel_path_explicit { # @test
  mkdir -p docs/plans
  cat >docs/plans/2026-03-24-mcp-workspace-tools-design.md <<-EOM
		# a plan doc, not a zettel
	EOM

  run_dodder status docs/plans/2026-03-24-mcp-workspace-tools-design.md
  assert_success
  assert_output '        untracked [docs/plans/2026-03-24-mcp-workspace-tools-design.md @blake2b256-3dgzgz72kwlq0uh4nxxh5w2gnghzz7qzlg3w5rs4mrek5dfefk6qkjkwz9]'
}

# Deeper nesting (four directories) is handled the same way.
function status_deeply_nested_path { # @test
  mkdir -p a/b/c
  cat >a/b/c/deep.md <<-EOM
		# deeply nested blob
	EOM

  run_dodder status .
  assert_success
  assert_output '        untracked [a/b/c/deep.md @blake2b256-0mxf7cq07gaw0g6ajh9krw8ur3fc4pqku76cjqr7r4r2atpas2jqrz8zap]'
}

# A file nested >=2 deep with an OBJECT extension (.zettel) whose path
# is not a representable zettel id is also handled gracefully: it is
# surfaced as an untracked blob (with its parsed metadata) rather than
# crashing the scan or being misread as a zettel at an invalid id. The
# blob digest is over the file's content (deterministic).
function status_nested_object_extension_zettel { # @test
  mkdir -p docs/plans
  cat >docs/plans/some-plan.zettel <<-EOM
		---
		# a plan
		! md
		---

		body
	EOM

  run_dodder status .
  assert_success
  assert_output '        untracked [docs/plans/some-plan.zettel @blake2b256-gu738nunyrnsqukgqkuaau9zslu0fhwg4dgs9ltuyvnlp42wal8sdpn2hc !md "a plan"]'
}

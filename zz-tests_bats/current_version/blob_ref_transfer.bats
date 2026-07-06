#! /usr/bin/env bats

setup() {
  load "$(dirname "$BATS_TEST_FILE")/../lib/common.bash"
  load "$(dirname "$BATS_TEST_FILE")/../lib/clone.bash"

  # for shellcheck SC2154
  export output
}

teardown() {
  chflags_nouchg
}

# bats file_tags=user_story:clone,user_story:referenced_objects,user_story:remote

# Regression tests for #325: blob-reference CONTENT blobs must be part of the
# clone/transfer blob closure. Genesis re-writes the embedded pandoc tool
# blobs in the clone (#324), which masks a missing-closure bug for the
# builtin !md refs -- so these tests reference a blob genesis does NOT
# provide (a unique madder-written blob) and assert it lands in the clone.
#
# NOTE: `dodder fsck` verifies each object's OWN blob digest AND the
# presence of every typed blob reference (#330): a repo missing a referenced
# content blob fails fsck with a distinct `missing blob reference` finding
# (see fsck_reports_missing_blob_reference below). The madder cat assertions
# in the clone tests still directly pin where the referenced bytes land.

# bootstrap_source_with_blob_ref initializes a source repo in $1, writes a
# unique content blob to its store, and commits a zettel carrying a metadata
# blob reference (RFC 0001 `alias < @digest !type` hyphence line) to that
# blob. Exports ref_blob_sha for the caller.
function bootstrap_source_with_blob_ref {
  mkdir -p "$1"
  pushd "$1" || exit 1

  run_dodder_init_disable_age

  run_madder write -format tap <(echo "custom tool payload not provided by genesis")
  assert_success
  ref_blob_sha="$(echo "$output" | grep -oP 'blake2b256-\S+' | head -1)"
  [[ -n $ref_blob_sha ]] || fail "could not extract blob sha from madder write output: $output"
  export ref_blob_sha

  run_dodder new -edit=false - <<-EOM
		---
		# zettel carrying custom blob reference
		- custom-tool < @${ref_blob_sha} !md
		! md
		---

		body referencing a non-genesis blob
	EOM
  assert_success

  popd || exit 1
}

# The default clone query resolves to inventory lists (+history). The
# closure traversal must still deliver the blob-reference content blobs of
# the objects contained in those lists -- a blob genesis does not provide
# would otherwise be missing from the clone's store (#325).
function clone_default_query_transfers_metadata_blob_reference { # @test
  bootstrap_source_with_blob_ref them

  run_clone_default_with \
    -direct "$(realpath ./them)" \
    .default

  assert_success

  # The transferred zettel is present with its blob reference intact.
  run_dodder show -format text one/uno:
  assert_success
  assert_output --regexp "custom-tool < @${ref_blob_sha}"

  # The referenced CONTENT blob itself must be in the clone's blob store.
  run_madder cat "$ref_blob_sha"
  assert_success
  assert_output "custom tool payload not provided by genesis"
}

# Same as above but with an explicit-genre query (the objects themselves in
# the transfer list, not inventory lists). Guards the expand-edges path that
# already collects AllBlobReferences from listed objects.
function clone_explicit_genres_transfers_metadata_blob_reference { # @test
  bootstrap_source_with_blob_ref them

  run_clone_default_with \
    -direct "$(realpath ./them)" \
    .default \
    +zettel,typ,etikett

  assert_success

  run_dodder show -format text one/uno:
  assert_success
  assert_output --regexp "custom-tool < @${ref_blob_sha}"

  run_madder cat "$ref_blob_sha"
  assert_success
  assert_output "custom tool payload not provided by genesis"
}

# fsck must flag an object whose metadata blob reference points at a content
# blob absent from every blob store (#330). The referenced digest is real --
# content-addressed bytes written into a SIBLING repo's CWD-scoped store --
# but the repo under test never receives the blob, so the reference dangles.
function fsck_reports_missing_blob_reference { # @test
  mkdir -p elsewhere
  pushd elsewhere || exit 1

  run_dodder_init_disable_age

  run_madder write -format tap <(echo "payload never present in the repo under test")
  assert_success
  ref_blob_sha="$(echo "$output" | grep -oP 'blake2b256-\S+' | head -1)"
  [[ -n $ref_blob_sha ]] || fail "could not extract blob sha from madder write output: $output"

  popd || exit 1

  run_dodder_init_disable_age

  run_dodder new -edit=false - <<-EOM
		---
		# zettel carrying a dangling blob reference
		- missing-tool < @${ref_blob_sha} !md
		! md
		---

		body referencing an absent blob
	EOM
  assert_success

  run_dodder fsck
  assert_output --regexp "not ok .*one/uno"
  assert_output --regexp "missing blob reference missing-tool<@${ref_blob_sha} on one/uno"
}

# The positive twin: when the referenced content blob IS present in the
# repo's blob store, fsck stays clean (#330 must not flag healthy refs).
function fsck_passes_with_present_blob_reference { # @test
  bootstrap_source_with_blob_ref here

  pushd here || exit 1

  run_dodder fsck
  assert_success
  refute_output --partial "not ok"
  refute_output --partial "missing blob reference"

  popd || exit 1
}

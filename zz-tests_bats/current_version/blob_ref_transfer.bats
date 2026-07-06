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
# NOTE: `dodder fsck` verifies only each object's OWN blob digest, not its
# blob references (fsck.go checks object.GetBlobDigest() only), so a clone
# missing a referenced content blob still passes fsck clean. The madder cat
# assertion below is therefore the load-bearing check.

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

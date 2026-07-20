package env_workspace

import (
	"testing"

	"code.linenisgreat.com/piggy/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
)

// #287b: assertParentPubkeyMatches verifies a workspace's resolved parent is
// the repo whose pubkey was pinned. The comparison is purpose-agnostic (parse
// stored string -> markl.Id -> markl.Equals), so these tests use deterministic
// blake2b256 digest strings as stand-in identities.

const (
	pubkeyA = "blake2b256-9j5cj9mjnk43k9rq4k2h3lezpl2sn3ura7cf8pa58cgfujw6nwgst7gtwz"
	pubkeyB = "blake2b256-40mtcwggatwwql4pp9ty93nyugn3r3ppvzs48uza0ze9zltneh3qez5yrs"
)

func mustSetId(t *ui.TestContext, s string) markl.Id {
	var id markl.Id
	t.AssertNoError(id.Set(s))
	return id
}

func TestAssertParentPubkeyMatches_Match(t1 *testing.T) {
	ui.RunTestContext(t1, func(t *ui.TestContext) {
		live := mustSetId(t, pubkeyA)

		// The pin is stored in StringWithFormat() form; feed that exact form
		// back so the test also covers the store->parse round-trip.
		err := AssertParentPubkeyMatches(live.StringWithFormat(), &live)
		t.AssertNoError(err)
	})
}

func TestAssertParentPubkeyMatches_Mismatch(t1 *testing.T) {
	ui.RunTestContext(t1, func(t *ui.TestContext) {
		pinned := mustSetId(t, pubkeyA)
		live := mustSetId(t, pubkeyB)

		err := AssertParentPubkeyMatches(pinned.StringWithFormat(), &live)
		t.AssertError(err)
	})
}

func TestAssertParentPubkeyMatches_UnpinnedSentinel(t1 *testing.T) {
	ui.RunTestContext(t1, func(t *ui.TestContext) {
		live := mustSetId(t, pubkeyA)

		err := AssertParentPubkeyMatches("", &live)
		t.AssertTrue(
			IsErrParentUnpinned(err),
			"empty pin must report ErrParentUnpinned",
		)
	})
}

func TestAssertParentPubkeyMatches_UnparseablePin(t1 *testing.T) {
	ui.RunTestContext(t1, func(t *ui.TestContext) {
		live := mustSetId(t, pubkeyA)

		// A stored pin that is neither empty nor a valid markl id is a
		// corrupt config, not the unpinned case: surface an error, but NOT
		// the unpinned sentinel.
		err := AssertParentPubkeyMatches("not-a-valid-markl-id", &live)
		t.AssertError(err)
		t.AssertFalse(
			IsErrParentUnpinned(err),
			"unparseable pin must not be reported as unpinned",
		)
	})
}

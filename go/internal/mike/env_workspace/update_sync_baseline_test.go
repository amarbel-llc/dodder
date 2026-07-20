package env_workspace

import (
	"testing"

	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"
	"code.linenisgreat.com/piggy/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
)

// #286: assertSyncBaselineBlobPresent must refuse to advance the sync
// baseline to an inventory list whose blob is absent from the read store,
// so the workspace is never pinned to a sync point it cannot read.

func makeListWithBlobDigest(t *ui.TestContext) *sku.Transacted {
	last := &sku.Transacted{}

	var blobDigest markl.Id
	t.AssertNoError(blobDigest.Set(
		"blake2b256-9j5cj9mjnk43k9rq4k2h3lezpl2sn3ura7cf8pa58cgfujw6nwgst7gtwz",
	))

	t.AssertNoError(last.SetBlobDigest(&blobDigest))

	return last
}

func TestAssertSyncBaselineBlobPresent_RejectsAbsentBlob(t1 *testing.T) {
	ui.RunTestContext(t1, func(t *ui.TestContext) {
		last := makeListWithBlobDigest(t)

		hasBlob := func(mad_domain_interfaces.MarklId) bool { return false }

		err := assertSyncBaselineBlobPresent(last, hasBlob)
		t.AssertError(err)
	})
}

func TestAssertSyncBaselineBlobPresent_AcceptsPresentBlob(t1 *testing.T) {
	ui.RunTestContext(t1, func(t *ui.TestContext) {
		last := makeListWithBlobDigest(t)

		var queried mad_domain_interfaces.MarklId
		hasBlob := func(id mad_domain_interfaces.MarklId) bool {
			queried = id
			return true
		}

		err := assertSyncBaselineBlobPresent(last, hasBlob)
		t.AssertNoError(err)

		// The guard must query the list's own blob digest, not something else.
		t.AssertTrue(
			markl.Equals(queried, last.GetBlobDigest()),
			"guard queried the wrong digest",
		)
	})
}

func TestAssertSyncBaselineBlobPresent_AcceptsNullBlob(t1 *testing.T) {
	ui.RunTestContext(t1, func(t *ui.TestContext) {
		// A blobless list (null digest) has nothing to fetch and cannot
		// brick a later read, so the guard must accept it without ever
		// consulting hasBlob.
		last := &sku.Transacted{}

		hasBlob := func(mad_domain_interfaces.MarklId) bool {
			t.Fatalf("hasBlob must not be called for a null blob digest")
			return false
		}

		err := assertSyncBaselineBlobPresent(last, hasBlob)
		t.AssertNoError(err)
	})
}

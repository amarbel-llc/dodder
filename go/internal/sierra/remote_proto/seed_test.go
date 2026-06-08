package remote_proto

import (
	"testing"

	"code.linenisgreat.com/dodder/go/internal/alfa/store_version"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
)

// TestSeedFieldsRoundTrip pins that the public seed fields (RepoId,
// StoreVersion) survive a capabilities-frame encode/decode, so a peer
// actually receives the sender's provenance and version guard (#253, B).
func TestSeedFieldsRoundTrip(t1 *testing.T) {
	t := ui.MakeT(t1)

	original := control{
		ProtocolVersion: ProtocolVersion,
		Role:            RoleServer,
		RepoId:          "some-repo-id",
		StoreVersion:    store_version.VCurrent.String(),
	}

	encoded, err := encodeControl(TypeCapabilities, original)
	t.AssertNoError(err)

	_, decoded, err := decodeControl(encoded)
	t.AssertNoError(err)
	t.AssertEqual(original.RepoId, decoded.RepoId)
	t.AssertEqual(original.StoreVersion, decoded.StoreVersion)
}

// TestAssertStoreVersionCompatible pins the fail-fast guard: an empty
// value (a peer that predates the seed fields) and any version this build
// can decode are accepted; a future version is rejected before transfer.
func TestAssertStoreVersionCompatible(t1 *testing.T) {
	t := ui.MakeT(t1)

	if err := assertStoreVersionCompatible(""); err != nil {
		t.Errorf("empty store version must be accepted, got %v", err)
	}

	if err := assertStoreVersionCompatible(store_version.VCurrent.String()); err != nil {
		t.Errorf("current store version must be accepted, got %v", err)
	}

	future := store_version.Version(store_version.VCurrent.GetInt() + 1)

	if err := assertStoreVersionCompatible(future.String()); err == nil {
		t.Errorf("a future store version (%s) must be rejected", future)
	}
}

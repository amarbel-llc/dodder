package markl_registrations_test

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/0/markl_registrations"
	"github.com/amarbel-llc/piggy/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
)

// Relocated from madder's internal/charlie/markl_registrations tests
// when the dodder-* purposes moved here (madder#255 step 3, madder
// 69e4fa6): the canonical sig→digest / sig→mother-sig mapping tables,
// the repo-key Related[public_key] pairing, and the Id.GetPublicKey
// end-to-end test all cross dodder-* purposes, so they live with the
// registrations that define them.

// AllPurposes is the canonical, ordered list of dodder's purposes.
// Every entry must be installed in markl's registry by the package's
// init(); iterate and assert the registered Type matches the canonical
// Type.
func TestAllPurposes_Registered(t1 *testing.T) {
	t := ui.MakeT(t1)

	for _, opts := range markl_registrations.AllPurposes {
		got := markl.GetPurpose(opts.Id)

		if got.GetPurposeType() != opts.Type {
			t.Errorf(
				"Type for %q: got %v, want %v",
				opts.Id,
				got.GetPurposeType(),
				opts.Type,
			)
		}
	}
}

// AllPurposes' Related metadata round-trips through GetRelated for
// every entry that declares related purposes.
func TestAllPurposes_RelatedRoundTrip(t1 *testing.T) {
	t := ui.MakeT(t1)

	for _, opts := range markl_registrations.AllPurposes {
		if len(opts.Related) == 0 {
			continue
		}

		purpose := markl.GetPurpose(opts.Id)

		for role, want := range opts.Related {
			got, ok := purpose.GetRelated(role)
			if !ok {
				t.Errorf("%s: GetRelated(%q): not found", opts.Id, role)
				continue
			}
			if got != want {
				t.Errorf(
					"%s: GetRelated(%q): got %q, want %q",
					opts.Id,
					role,
					got,
					want,
				)
			}
		}
	}
}

// PurposeRepoPrivateKeyV1's Related[public_key] mapping is what
// Id.GetPublicKey reads to stamp the result. Pin the canonical pairing
// so a registration drift in markl_registrations would surface here.
func TestPurposeRepoPrivateKeyV1_RelatedPublicKey(t1 *testing.T) {
	t := ui.MakeT(t1)

	priv := markl.GetPurpose(markl.PurposeRepoPrivateKeyV1)

	got, ok := priv.GetRelated(markl.RelatedRolePublicKey)
	if !ok {
		t.Fatalf("GetRelated(RelatedRolePublicKey) on %q: not found",
			markl.PurposeRepoPrivateKeyV1)
	}

	if got != markl.PurposeRepoPubKeyV1 {
		t.Errorf(
			"Related[public_key] = %q, want %q",
			got,
			markl.PurposeRepoPubKeyV1,
		)
	}
}

// The canonical sig→digest mapping for each dodder-object-sig version,
// asserted via the registered Related["digest"] entries. v0 predates
// the Related metadata and must expose no digest mapping.
func TestObjectSigRelatedDigest_Canonical(t1 *testing.T) {
	t := ui.MakeT(t1)

	if _, ok := markl.GetPurpose(markl.PurposeObjectSigV0).GetRelated(
		markl.RelatedRoleDigest,
	); ok {
		t.Errorf("%q: unexpected Related[digest]", markl.PurposeObjectSigV0)
	}

	for _, testCase := range []struct {
		sigId    string
		digestId string
	}{
		{markl.PurposeObjectSigV1, markl.PurposeObjectDigestV1},
		{markl.PurposeObjectSigV2, markl.PurposeObjectDigestV2},
		{markl.PurposeObjectSigV3, markl.PurposeObjectDigestV3},
	} {
		got, ok := markl.GetPurpose(testCase.sigId).GetRelated(
			markl.RelatedRoleDigest,
		)
		if !ok {
			t.Errorf("%q: GetRelated(digest): not found", testCase.sigId)
			continue
		}
		if got != testCase.digestId {
			t.Errorf(
				"%q: got %q, want %q",
				testCase.sigId,
				got,
				testCase.digestId,
			)
		}
	}
}

// Same shape as the digest table but for the mother_sig role.
func TestObjectSigRelatedMotherSig_Canonical(t1 *testing.T) {
	t := ui.MakeT(t1)

	if _, ok := markl.GetPurpose(markl.PurposeObjectSigV0).GetRelated(
		markl.RelatedRoleMotherSig,
	); ok {
		t.Errorf("%q: unexpected Related[mother_sig]", markl.PurposeObjectSigV0)
	}

	for _, testCase := range []struct {
		sigId       string
		motherSigId string
	}{
		{markl.PurposeObjectSigV1, markl.PurposeObjectMotherSigV1},
		{markl.PurposeObjectSigV2, markl.PurposeObjectMotherSigV2},
		{markl.PurposeObjectSigV3, markl.PurposeObjectMotherSigV3},
	} {
		got, ok := markl.GetPurpose(testCase.sigId).GetRelated(
			markl.RelatedRoleMotherSig,
		)
		if !ok {
			t.Errorf("%q: GetRelated(mother_sig): not found", testCase.sigId)
			continue
		}
		if got != testCase.motherSigId {
			t.Errorf(
				"%q: got %q, want %q",
				testCase.sigId,
				got,
				testCase.motherSigId,
			)
		}
	}
}

// End-to-end: Id.GetPublicKey delegates to the registered FormatSec via
// PurposeRepoPrivateKeyV1, stamps the result with PurposeRepoPubKeyV1,
// and the result bytes match Go's stdlib ed25519.
func TestIdGetPublicKey_Ed25519_MatchesStdlib(t1 *testing.T) {
	t := ui.MakeT(t1)

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey: %s", err)
	}

	var secId markl.Id
	if err := secId.SetPurposeId(markl.PurposeRepoPrivateKeyV1); err != nil {
		t.Fatalf("SetPurposeId: %s", err)
	}
	if err := secId.SetMarklId(markl.FormatIdEd25519Sec, priv); err != nil {
		t.Fatalf("SetMarklId: %s", err)
	}

	pubId, err := secId.GetPublicKey(markl.PurposeRepoPrivateKeyV1)
	if err != nil {
		t.Fatalf("Id.GetPublicKey: %s", err)
	}

	want := priv.Public().(ed25519.PublicKey)
	if !bytes.Equal(pubId.GetBytes(), want) {
		t.Errorf("pubkey mismatch:\n got  %x\n want %x", pubId.GetBytes(), want)
	}
}

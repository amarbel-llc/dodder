//go:build test

package markl

import (
	"crypto/ed25519"
	"testing"

	"code.linenisgreat.com/dodder/go/src/bravo/ui"
)

func TestPIVSignThenVerifyWithSoftwareKey(t1 *testing.T) {
	ui.RunTestContext(t1, func(t *ui.TestContext) {
		mock := makeMockPIVSigner()
		resetPIVFormatForTesting()
		RegisterPIVEd25519Format(mock)

		// Create the PIV-backed private key markl.Id
		var pivKey Id
		ref := EncodePIVReference(mock.GUID(), mock.SlotId())
		err := pivKey.SetPurposeId(PurposeRepoPrivateKeyV1)
		t.AssertNoError(err)
		err = pivKey.SetMarklId(FormatIdEd25519PIV, ref[:])
		t.AssertNoError(err)

		// Derive the public key via PIV format
		pubKey, err := pivKey.GetPublicKey(PurposeRepoPrivateKeyV1)
		t.AssertNoError(err)

		// Create a message (simulating an object digest)
		message, repool := FormatHashSha256.GetMarklIdForString("object digest content")
		defer repool()

		// Sign the message via PIV
		var sig Id
		err = pivKey.Sign(message, &sig, PurposeObjectSigV2)
		t.AssertNoError(err)
		t.AssertNoError(AssertIdIsNotNull(sig))

		// Verify with the derived public key
		err = pubKey.Verify(message, sig)
		t.AssertNoError(err)

		// Also verify with a separately constructed public key from raw bytes
		expectedPub := mock.privateKey.Public().(ed25519.PublicKey)
		var standalonePub Id
		err = standalonePub.SetPurposeId(PurposeRepoPubKeyV1)
		t.AssertNoError(err)
		err = standalonePub.SetMarklId(FormatIdEd25519Pub, expectedPub)
		t.AssertNoError(err)

		err = standalonePub.Verify(message, sig)
		t.AssertNoError(err)
	})
}

func TestPIVSignatureMatchesSoftwareSignature(t1 *testing.T) {
	ui.RunTestContext(t1, func(t *ui.TestContext) {
		mock := makeMockPIVSigner()
		resetPIVFormatForTesting()
		RegisterPIVEd25519Format(mock)

		message, repool := FormatHashSha256.GetMarklIdForString("deterministic test")
		defer repool()

		// Sign via PIV format
		var pivKey Id
		ref := EncodePIVReference(mock.GUID(), mock.SlotId())
		err := pivKey.SetPurposeId(PurposeRepoPrivateKeyV1)
		t.AssertNoError(err)
		err = pivKey.SetMarklId(FormatIdEd25519PIV, ref[:])
		t.AssertNoError(err)

		var pivSig Id
		err = pivKey.Sign(message, &pivSig, PurposeObjectSigV2)
		t.AssertNoError(err)

		// Sign via software format (same underlying key)
		var softKey Id
		err = softKey.SetPurposeId(PurposeRepoPrivateKeyV1)
		t.AssertNoError(err)
		err = softKey.SetMarklId(FormatIdEd25519Sec, []byte(mock.privateKey))
		t.AssertNoError(err)

		var softSig Id
		err = softKey.Sign(message, &softSig, PurposeObjectSigV2)
		t.AssertNoError(err)

		// Both signatures should verify with the same public key
		pubKey, err := softKey.GetPublicKey(PurposeRepoPrivateKeyV1)
		t.AssertNoError(err)

		t.AssertNoError(pubKey.Verify(message, pivSig))
		t.AssertNoError(pubKey.Verify(message, softSig))

		// Ed25519 signatures are deterministic, so with the same
		// underlying key, both signatures should be byte-identical.
		t.AssertNoError(AssertEqual(pivSig, softSig))
	})
}

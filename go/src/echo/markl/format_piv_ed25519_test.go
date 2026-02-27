//go:build test

package markl

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"io"
	"testing"

	"code.linenisgreat.com/dodder/go/src/bravo/ui"
)

func TestPIVEd25519FormatIdRegistered(t1 *testing.T) {
	ui.RunTestContext(t1, func(t *ui.TestContext) {
		purpose := GetPurpose(PurposeRepoPrivateKeyV1)
		if _, ok := purpose.formatIds[FormatIdEd25519PIV]; !ok {
			t.Fatalf(
				"format %q not registered for purpose %q",
				FormatIdEd25519PIV,
				PurposeRepoPrivateKeyV1,
			)
		}
	})
}

func TestPIVReferenceEncodeDecode(t1 *testing.T) {
	ui.RunTestContext(t1, func(t *ui.TestContext) {
		var guid [16]byte
		for i := range guid {
			guid[i] = byte(i)
		}
		slotId := byte(0x9c)

		ref := EncodePIVReference(guid, slotId)

		decodedGUID, decodedSlot, err := DecodePIVReference(ref[:])
		t.AssertNoError(err)

		if decodedGUID != guid {
			t.Fatalf("GUID mismatch: %x != %x", decodedGUID, guid)
		}
		if decodedSlot != slotId {
			t.Fatalf("slot mismatch: %x != %x", decodedSlot, slotId)
		}
	})
}

func TestPIVReferenceDecodeInvalidSize(t1 *testing.T) {
	ui.RunTestContext(t1, func(t *ui.TestContext) {
		_, _, err := DecodePIVReference([]byte{0x01, 0x02})
		t.AssertError(err)
	})
}

type mockPIVSigner struct {
	privateKey ed25519.PrivateKey
	guid       [16]byte
	slotId     byte
}

func (m *mockPIVSigner) Public() crypto.PublicKey {
	return m.privateKey.Public()
}

func (m *mockPIVSigner) Sign(
	rand io.Reader,
	digest []byte,
	opts crypto.SignerOpts,
) ([]byte, error) {
	return m.privateKey.Sign(rand, digest, opts)
}

func (m *mockPIVSigner) GUID() [16]byte { return m.guid }
func (m *mockPIVSigner) SlotId() byte   { return m.slotId }
func (m *mockPIVSigner) Close() error   { return nil }

func makeMockPIVSigner() *mockPIVSigner {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}

	var guid [16]byte
	for i := range guid {
		guid[i] = byte(i + 0xA0)
	}

	return &mockPIVSigner{
		privateKey: priv,
		guid:       guid,
		slotId:     0x9c,
	}
}

func TestRegisterPIVEd25519FormatAndSign(t1 *testing.T) {
	ui.RunTestContext(t1, func(t *ui.TestContext) {
		mock := makeMockPIVSigner()
		RegisterPIVEd25519Format(mock)

		// Create a markl.Id with the PIV format
		var pivKey Id
		ref := EncodePIVReference(mock.GUID(), mock.SlotId())
		err := pivKey.SetPurposeId(PurposeRepoPrivateKeyV1)
		t.AssertNoError(err)
		err = pivKey.SetMarklId(FormatIdEd25519PIV, ref[:])
		t.AssertNoError(err)

		// Create a message to sign (simulate object digest)
		message, repool := FormatHashSha256.GetMarklIdForString("test message")
		defer repool()

		// Sign via the PIV format
		var sig Id
		err = pivKey.Sign(message, &sig, PurposeObjectSigV2)
		t.AssertNoError(err)
		t.AssertNoError(AssertIdIsNotNull(sig))

		// Verify using standard Ed25519 public key
		pubBytes := mock.privateKey.Public().(ed25519.PublicKey)
		var pubKey Id
		err = pubKey.SetPurposeId(PurposeRepoPubKeyV1)
		t.AssertNoError(err)
		err = pubKey.SetMarklId(FormatIdEd25519Pub, pubBytes)
		t.AssertNoError(err)
		err = pubKey.Verify(message, sig)
		t.AssertNoError(err)
	})
}

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

func TestPIVStubParseWithoutSigner(t1 *testing.T) {
	ui.RunTestContext(t1, func(t *ui.TestContext) {
		resetPIVFormatForTesting()

		// Parsing a PIV markl.Id should succeed via the stub format
		var pivKey Id
		var guid [16]byte
		for i := range guid {
			guid[i] = byte(i)
		}
		ref := EncodePIVReference(guid, 0x9c)
		err := pivKey.SetPurposeId(PurposeRepoPrivateKeyV1)
		t.AssertNoError(err)
		err = pivKey.SetMarklId(FormatIdEd25519PIV, ref[:])
		t.AssertNoError(err)

		if pivKey.GetMarklFormat().GetMarklFormatId() != FormatIdEd25519PIV {
			t.Fatalf("expected format %q", FormatIdEd25519PIV)
		}
	})
}

func TestPIVStubSignReturnsConnectionError(t1 *testing.T) {
	ui.RunTestContext(t1, func(t *ui.TestContext) {
		resetPIVFormatForTesting()

		var pivKey Id
		var guid [16]byte
		ref := EncodePIVReference(guid, 0x9c)
		err := pivKey.SetPurposeId(PurposeRepoPrivateKeyV1)
		t.AssertNoError(err)
		err = pivKey.SetMarklId(FormatIdEd25519PIV, ref[:])
		t.AssertNoError(err)

		message, repool := FormatHashSha256.GetMarklIdForString("test")
		defer repool()

		var sig Id
		err = pivKey.Sign(message, &sig, PurposeObjectSigV2)
		if err == nil {
			t.Fatal("expected error from stub Sign")
		}

		if !IsErrEd25519PIVSignerConnectionNotEstablished(err) {
			t.Fatalf("expected PIV connection error, got: %s", err)
		}
	})
}

func TestPIVStubGetPublicKeyReturnsConnectionError(t1 *testing.T) {
	ui.RunTestContext(t1, func(t *ui.TestContext) {
		resetPIVFormatForTesting()

		var pivKey Id
		var guid [16]byte
		ref := EncodePIVReference(guid, 0x9c)
		err := pivKey.SetPurposeId(PurposeRepoPrivateKeyV1)
		t.AssertNoError(err)
		err = pivKey.SetMarklId(FormatIdEd25519PIV, ref[:])
		t.AssertNoError(err)

		_, err = pivKey.GetPublicKey(PurposeRepoPrivateKeyV1)
		if err == nil {
			t.Fatal("expected error from stub GetPublicKey")
		}

		if !IsErrEd25519PIVSignerConnectionNotEstablished(err) {
			t.Fatalf("expected PIV connection error, got: %s", err)
		}
	})
}

func TestRegisterPIVEd25519FormatAndSign(t1 *testing.T) {
	ui.RunTestContext(t1, func(t *ui.TestContext) {
		mock := makeMockPIVSigner()
		// Remove any previously registered PIV format so this test's mock
		// signer is the one captured by the closure.
		resetPIVFormatForTesting()
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

func TestPIVMarklIdTextRoundTrip(t1 *testing.T) {
	ui.RunTestContext(t1, func(t *ui.TestContext) {
		mock := makeMockPIVSigner()

		// Ensure format is registered (idempotent)
		RegisterPIVEd25519Format(mock)

		var original Id
		ref := EncodePIVReference(mock.GUID(), mock.SlotId())
		err := original.SetPurposeId(PurposeRepoPrivateKeyV1)
		t.AssertNoError(err)
		err = original.SetMarklId(FormatIdEd25519PIV, ref[:])
		t.AssertNoError(err)

		// Marshal to text
		text, err := original.MarshalText()
		t.AssertNoError(err)
		t.Log("serialized:", string(text))

		if len(text) == 0 {
			t.Fatal("marshaled text is empty")
		}

		// Unmarshal from text
		var decoded Id
		err = decoded.UnmarshalText(text)
		t.AssertNoError(err)

		// Verify round-trip
		t.AssertNoError(AssertEqual(original, decoded))

		if decoded.GetPurposeId() != PurposeRepoPrivateKeyV1 {
			t.Fatalf(
				"purpose mismatch: %q != %q",
				decoded.GetPurposeId(),
				PurposeRepoPrivateKeyV1,
			)
		}

		if decoded.GetMarklFormat().GetMarklFormatId() != FormatIdEd25519PIV {
			t.Fatalf(
				"format mismatch: %q != %q",
				decoded.GetMarklFormat().GetMarklFormatId(),
				FormatIdEd25519PIV,
			)
		}

		// Verify reference bytes survived
		decodedGUID, decodedSlot, err := DecodePIVReference(decoded.GetBytes())
		t.AssertNoError(err)
		if decodedGUID != mock.GUID() {
			t.Fatalf("GUID mismatch after round-trip")
		}
		if decodedSlot != mock.SlotId() {
			t.Fatalf("slot mismatch after round-trip")
		}
	})
}

func TestPIVGetPublicKey(t1 *testing.T) {
	ui.RunTestContext(t1, func(t *ui.TestContext) {
		mock := makeMockPIVSigner()
		// Remove any previously registered PIV format so this test's mock
		// signer is the one captured by the closure.
		resetPIVFormatForTesting()
		RegisterPIVEd25519Format(mock)

		var pivKey Id
		ref := EncodePIVReference(mock.GUID(), mock.SlotId())
		err := pivKey.SetPurposeId(PurposeRepoPrivateKeyV1)
		t.AssertNoError(err)
		err = pivKey.SetMarklId(FormatIdEd25519PIV, ref[:])
		t.AssertNoError(err)

		// GetPublicKey should return the signer's public key
		pubKey, err := pivKey.GetPublicKey(PurposeRepoPrivateKeyV1)
		t.AssertNoError(err)
		t.AssertNoError(AssertIdIsNotNull(pubKey))

		expectedPub := mock.privateKey.Public().(ed25519.PublicKey)

		err = MakeErrNotEqualBytes(expectedPub, pubKey.GetBytes())
		t.AssertNoError(err)

		if pubKey.GetMarklFormat().GetMarklFormatId() != FormatIdEd25519Pub {
			t.Fatalf(
				"expected format %q but got %q",
				FormatIdEd25519Pub,
				pubKey.GetMarklFormat().GetMarklFormatId(),
			)
		}
	})
}

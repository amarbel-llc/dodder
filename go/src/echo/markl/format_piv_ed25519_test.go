//go:build test

package markl

import (
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

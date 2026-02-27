package markl

import (
	"crypto"

	"code.linenisgreat.com/dodder/go/src/alfa/errors"
)

const PIVReferenceSize = 17 // 16-byte GUID + 1-byte slot ID

type PIVSigner interface {
	crypto.Signer
	GUID() [16]byte
	SlotId() byte
	Close() error
}

func EncodePIVReference(guid [16]byte, slotId byte) [PIVReferenceSize]byte {
	var ref [PIVReferenceSize]byte
	copy(ref[:16], guid[:])
	ref[16] = slotId
	return ref
}

func DecodePIVReference(data []byte) (guid [16]byte, slotId byte, err error) {
	if len(data) != PIVReferenceSize {
		err = errors.Errorf(
			"invalid PIV reference size: expected %d, got %d",
			PIVReferenceSize,
			len(data),
		)
		return guid, slotId, err
	}

	copy(guid[:], data[:16])
	slotId = data[16]
	return guid, slotId, err
}

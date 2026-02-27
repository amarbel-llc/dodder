package markl

type PurposeType interface {
	purposeType()
}

type purposeType byte

var _ PurposeType = purposeType(0)

func (purposeType) purposeType() {}

const (
	PurposeTypeUnknown = purposeType(iota)
	PurposeTypeBlobDigest
	PurposeTypeObjectDigest
	PurposeTypeObjectMotherSig
	PurposeTypeObjectSig
	PurposeTypePrivateKey
	PurposeTypePubKey
	PurposeTypeRepoPubKey
	PurposeTypeRequestAuth
)

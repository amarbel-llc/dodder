package markl

import (
	"fmt"
)

// purposes currently treated as formats
const (
	// TODO move to ids' builtin types
	// and then add registration
	// keep sorted

	// Blob Digests
	PurposeBlobDigestV1 = "dodder-blob-digest-sha256-v1"

	// Object Digests
	PurposeObjectDigestV1             = "dodder-object-digest-sha256-v1"
	PurposeObjectDigestV2             = "dodder-object-digest-v2"
	PurposeV5MetadataDigestWithoutTai = "dodder-object-metadata-digest-without_tai-v1"

	// Object Mother Sigs
	PurposeObjectMotherSigV1 = "dodder-object-mother-sig-v1"
	PurposeObjectMotherSigV2 = "dodder-object-mother-sig-v2"

	// Object Sigs
	PurposeObjectSigV0 = "dodder-repo-sig-v1"
	PurposeObjectSigV1 = "dodder-object-sig-v1"
	PurposeObjectSigV2 = "dodder-object-sig-v2"

	// Request Auth
	PurposeRequestAuthResponseV1  = "dodder-request_auth-response-v1"
	PurposeRequestRepoSigV1       = "dodder-request_auth-repo-sig-v1"
	PurposeRequestAuthChallengeV1 = "dodder-request_auth-challenge-v1"

	// PubKeys
	PurposeRepoPubKeyV1   = "dodder-repo-public_key-v1"
	PurposeMadderPubKeyV1 = "madder-public_key-v1"

	// PrivateKeys
	PurposeRepoPrivateKeyV1   = "dodder-repo-private_key-v1"
	PurposeMadderPrivateKeyV0 = "madder-private_key-v0"
	PurposeMadderPrivateKeyV1 = "madder-private_key-v1"
)

func init() {
	// purposes that need to be reregistered
	makePurpose(
		PurposeBlobDigestV1,
		PurposeTypeBlobDigest,
		FormatIdHashSha256,
		FormatIdHashBlake2b256,
	)

	makePurpose(
		PurposeObjectDigestV1,
		PurposeTypeObjectDigest,
		FormatIdHashSha256,
		FormatIdHashBlake2b256,
	)

	makePurpose(
		PurposeObjectDigestV2,
		PurposeTypeObjectDigest,
		FormatIdHashSha256,
		FormatIdHashBlake2b256,
	)

	makePurpose(
		PurposeV5MetadataDigestWithoutTai,
		PurposeTypeObjectDigest,
		FormatIdHashSha256,
		FormatIdHashBlake2b256,
	)

	makePurpose(
		PurposeObjectMotherSigV1,
		PurposeTypeObjectMotherSig,
		FormatIdEd25519Sig,
	)

	makePurpose(
		PurposeObjectMotherSigV2,
		PurposeTypeObjectMotherSig,
		FormatIdEd25519Sig,
	)

	makePurpose(
		PurposeObjectSigV0,
		PurposeTypeObjectSig,
		FormatIdEd25519Sig,
	)

	makePurpose(
		PurposeObjectSigV1,
		PurposeTypeObjectSig,
		FormatIdEd25519Sig,
	)

	makePurpose(
		PurposeObjectSigV2,
		PurposeTypeObjectSig,
		FormatIdEd25519Sig,
	)

	makePurpose(
		PurposeRepoPrivateKeyV1,
		PurposeTypePrivateKey,
		FormatIdEd25519Sec,
		FormatIdEd25519PIV,
	)

	makePurpose(
		PurposeRepoPubKeyV1,
		PurposeTypeRepoPubKey,
		FormatIdEd25519Pub,
	)

	makePurpose(PurposeRequestAuthChallengeV1, PurposeTypeRequestAuth)
	makePurpose(PurposeRequestAuthResponseV1, PurposeTypeRequestAuth)

	makePurpose(
		PurposeMadderPubKeyV1,
		PurposeTypePubKey,
		FormatIdEd25519Pub,
	)

	makePurpose(
		PurposeMadderPrivateKeyV0,
		PurposeTypePrivateKey,
		FormatIdEd25519Sec,
		FormatIdAgeX25519Sec,
	)

	makePurpose(
		PurposeMadderPrivateKeyV1,
		PurposeTypePrivateKey,
		FormatIdEd25519Sec,
		FormatIdAgeX25519Sec,
	)
}

var purposes = map[string]Purpose{}

type Purpose struct {
	id        string
	tipe      PurposeType
	formatIds map[string]struct{}
}

func GetPurpose(purposeId string) Purpose {
	purpose, ok := purposes[purposeId]

	if !ok {
		panic(fmt.Sprintf("no purpose registered for id %q", purposeId))
	}

	return purpose
}

func makePurpose(purposeId string, purposeType PurposeType, formatIds ...string) {
	_, alreadyExists := purposes[purposeId]

	if alreadyExists {
		panic(fmt.Sprintf("hash type already registered: %q", purposeId))
	}

	purpose := Purpose{
		id:        purposeId,
		tipe:      purposeType,
		formatIds: make(map[string]struct{}),
	}

	for _, formatId := range formatIds {
		_, ok := purpose.formatIds[formatId]

		if ok {
			panic(
				fmt.Sprintf("format id (%q) registered for purpose (%q) more than once",
					formatId,
					purposeId,
				),
			)
		}

		purpose.formatIds[formatId] = struct{}{}
	}

	purposes[purposeId] = purpose
}

func (purpose Purpose) GetPurposeType() PurposeType {
	return purpose.tipe
}

func GetDigestTypeForSigType(sigId string) string {
	sig := GetPurpose(sigId)

	switch sig.id {
	default:
		panic(fmt.Sprintf("unsupported sig purpose: %q", sigId))

	case PurposeObjectSigV1:
		return PurposeObjectDigestV1

	case PurposeObjectSigV2:
		return PurposeObjectDigestV2
	}
}

func GetMotherSigTypeForSigType(sigId string) string {
	sig := GetPurpose(sigId)

	switch sig.id {
	default:
		panic(fmt.Sprintf("unsupported sig purpose: %q", sigId))

	case PurposeObjectSigV1:
		return PurposeObjectMotherSigV1

	case PurposeObjectSigV2:
		return PurposeObjectMotherSigV2
	}
}

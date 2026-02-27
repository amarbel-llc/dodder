package objects

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/markl"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/quiter_set"
)

func SetTags[TAG ids.Tag](metadata MetadataMutable, otherTags ids.Set[TAG]) {
	{
		metadata := metadata.(*MetadataStruct)
		metadata.Tags.Reset()

		if otherTags == nil {
			return
		}

		if otherTags.Len() == 1 && quiter_set.Any(otherTags).String() == "" {
			panic("empty tag set")
		}

		for tag := range otherTags.All() {
			errors.PanicIfError(metadata.AddTagString(tag.String()))
		}
	}
}

func GetMarklIdForPurpose(
	metadata Metadata,
	purposeId string,
) domain_interfaces.MarklId {
	purposeType := markl.GetPurpose(purposeId).GetPurposeType()

	switch purposeType {

	case markl.PurposeTypeBlobDigest:
		return metadata.GetBlobDigest()

	case markl.PurposeTypeObjectMotherSig:
		return metadata.GetMotherObjectSig()

	case markl.PurposeTypeObjectSig:
		return metadata.GetObjectSig()

	case markl.PurposeTypeRepoPubKey:
		return metadata.GetRepoPubKey()

	default:
		panic(fmt.Sprintf("unsupported purpose type: %q", purposeType))
	}
}

func GetMarklIdMutableForPurpose(
	metadata MetadataMutable,
	purposeId string,
) domain_interfaces.MarklIdMutable {
	purposeType := markl.GetPurpose(purposeId).GetPurposeType()

	switch purposeType {

	case markl.PurposeTypeBlobDigest:
		return metadata.GetBlobDigestMutable()

	case markl.PurposeTypeObjectMotherSig:
		return metadata.GetMotherObjectSigMutable()

	case markl.PurposeTypeObjectSig:
		return metadata.GetObjectSigMutable()

	case markl.PurposeTypeRepoPubKey:
		return metadata.GetRepoPubKeyMutable()

	default:
		panic(fmt.Sprintf("unsupported purpose type: %q", purposeType))
	}
}

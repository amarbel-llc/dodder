package objects

import (
	"fmt"

	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"

	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/lib/alfa/quiter_set"
	"code.linenisgreat.com/piggy/go/pkgs/markl"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
)

func SetTags[TAG ids.Tag](metadata MetadataMutable, otherTags ids.Set[TAG]) {
	{
		metadata := metadata.(*MetadataStruct)
		metadata.contents.ResetTags()

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
) mad_domain_interfaces.MarklId {
	purposeType := markl.GetPurpose(purposeId).GetPurposeType()

	switch purposeType {

	case markl.PurposeTypeBlobDigest:
		return metadata.GetBlobDigest()

	case markl.PurposeTypeObjectMotherSig:
		return metadata.GetMotherObjectSig()

	case markl.PurposeTypeDodderObjectSig:
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
) mad_domain_interfaces.MarklIdMutable {
	purposeType := markl.GetPurpose(purposeId).GetPurposeType()

	switch purposeType {

	case markl.PurposeTypeBlobDigest:
		return metadata.GetBlobDigestMutable()

	case markl.PurposeTypeObjectMotherSig:
		return metadata.GetMotherObjectSigMutable()

	case markl.PurposeTypeDodderObjectSig:
		return metadata.GetObjectSigMutable()

	case markl.PurposeTypeRepoPubKey:
		return metadata.GetRepoPubKeyMutable()

	default:
		panic(fmt.Sprintf("unsupported purpose type: %q", purposeType))
	}
}

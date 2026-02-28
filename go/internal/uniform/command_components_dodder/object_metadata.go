package command_components_dodder

import (
	"strings"

	"code.linenisgreat.com/dodder/go/internal/delta/objects"
	"code.linenisgreat.com/dodder/go/lib/_/flag_policy"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/flags"
)

type ObjectMetadata struct{}

func (cmd ObjectMetadata) GetFlagValueMetadataTags(
	metadata objects.MetadataMutable,
) interfaces.FlagValue {
	// TODO add support for tag_paths
	fes := flags.MakeWithPolicy(
		flag_policy.FlagPolicyAppend,
		func() string {
			return metadata.GetIndex().GetTagPaths().String()
		},
		func(combinedTag string) (err error) {
			tagStrings := strings.SplitSeq(combinedTag, ",")

			for tag := range tagStrings {
				if err = metadata.AddTagString(tag); err != nil {
					err = errors.Wrap(err)
					return err
				}
			}

			return err
		},
		func() {
			metadata.ResetTags()
		},
	)

	return fes
}

func (cmd ObjectMetadata) GetFlagValueMetadataDescription(
	metadata objects.MetadataMutable,
) interfaces.FlagValue {
	return metadata.GetDescriptionMutable()
}

func (cmd ObjectMetadata) GetFlagValueMetadataType(
	metadata objects.MetadataMutable,
) interfaces.FlagValue {
	return flags.MakeWithPolicy(
		flag_policy.FlagPolicyReset,
		func() string {
			return metadata.GetType().String()
		},
		func(value string) (err error) {
			if err = metadata.GetTypeMutable().SetType(value); err != nil {
				err = errors.Wrap(err)
				return err
			}

			return err
		},
		func() {
			metadata.GetTypeMutable().Reset()
		},
	)
}
